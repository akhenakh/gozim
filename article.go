package zim

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"

	xz "github.com/remyoudompheng/go-liblzma"
)

const (
	RedirectEntry   uint16 = 0xffff
	LinkTargetEntry        = 0xfffe
	DeletedEntry           = 0xfffd
)

type Article struct {
	URLPtr     uint64
	Mimetype   uint16
	Namespace  byte
	URL        string
	Title      string
	Blob       uint32
	Cluster    uint32
	RedirectTo *Article
}

func (z *ZimReader) FillArticleAt(a *Article, offset uint64) *Article {
	a.URLPtr = offset

	mimeIdx, err := readInt16(z.mmap[offset : offset+2])
	if err != nil {
		panic(err)
	}
	a.Mimetype = mimeIdx

	// Linktarget or Target Entry
	if mimeIdx == LinkTargetEntry || mimeIdx == DeletedEntry {
		//TODO
		return nil
	}

	a.Namespace = z.mmap[offset+3]

	a.Cluster, err = readInt32(z.mmap[offset+8 : offset+8+4])
	if err != nil {
		panic(err)
	}

	a.Blob, err = readInt32(z.mmap[offset+12 : offset+12+4])
	if err != nil {
		panic(err)
	}

	// Redirect
	if mimeIdx == RedirectEntry {
		// check for a possible loop: the redirect could point to the same target
		if z.GetUrlOffsetAtIdx(a.Cluster) != offset {
			// redirect ptr share the same memory offset than Cluster number
			a.RedirectTo = z.getArticleAt(z.GetUrlOffsetAtIdx(a.Cluster))
		}

		b := bytes.NewBuffer(z.mmap[offset+12:])
		a.URL, err = b.ReadString('\x00')
		if err != nil {
			panic(err)
		}
		a.URL = strings.TrimRight(string(a.URL), "\x00")

		a.Title, err = b.ReadString('\x00')
		if err != nil {
			panic(err)
		}
		a.Title = strings.TrimRight(string(a.Title), "\x00")

		return a
	}

	b := bytes.NewBuffer(z.mmap[offset+16:])
	a.URL, err = b.ReadString('\x00')
	if err != nil {
		panic(err)
	}
	a.URL = strings.TrimRight(string(a.URL), "\x00")

	a.Title, err = b.ReadString('\x00')
	if err != nil {
		panic(err)
	}
	a.Title = strings.TrimRight(string(a.Title), "\x00")

	return a
}

// get the article (Directory) pointed by the offset found in URLpos or Titlepos
func (z *ZimReader) getArticleAt(offset uint64) *Article {
	a := new(Article)
	z.FillArticleAt(a, offset)
	return a
}

// return the uncompressed data associated with this article
func (a *Article) Data(z *ZimReader) []byte {
	start, end := z.getClusterOffsetsAtIdx(a.Cluster)
	compression := uint8(z.mmap[start])

	// LZMA
	if compression == 4 {
		b := bytes.NewBuffer(z.mmap[start+1 : end+1])
		dec, err := xz.NewReader(b)
		defer dec.Close()
		if err != nil {
			panic(err)
		}

		// the decoded chunk are around 1MB
		// TODO: on smaller devices need to read stream rather than ReadAll
		blob, err := ioutil.ReadAll(dec)
		if err != nil {
			panic(err)
		}

		// blob starts at offset, blob ends at offset
		var bs, be uint32

		bs, err = readInt32(blob[a.Blob*4 : a.Blob*4+4])
		if err != nil {
			panic(err)
		}

		be, err = readInt32(blob[a.Blob*4+4 : a.Blob*4+4+4])
		if err != nil {
			panic(err)
		}

		return blob[bs:be]
	}

	return nil
}

// return the url prefixed by the namespace
func (a *Article) FullURL() string {
	return string(a.Namespace) + "/" + a.URL
}

func (a *Article) getBlobOffsetsAtIdx(z *ZimReader) (start, end uint64) {
	idx := a.Blob
	offset := z.clusterPtrPos + uint64(idx)*8
	start, err := readInt64(z.mmap[offset : offset+8])
	if err != nil {
		panic(err)
	}
	offset = z.clusterPtrPos + uint64(idx+1)*8
	end, err = readInt64(z.mmap[offset : offset+8])
	if err != nil {
		panic(err)
	}
	return
}

func (a *Article) String() string {
	return fmt.Sprintf("Mime: 0x%x URL: [%s], Title: [%s], Cluster: 0x%x Blob: 0x%x",
		a.Mimetype, a.URL, a.Title, a.Cluster, a.Blob)
}
