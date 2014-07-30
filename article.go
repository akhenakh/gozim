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
	// EntryType is a RedirectEntry/LinkTargetEntry/DeletedEntry or an idx
	// pointing to ZimReader.mimeTypeList
	EntryType  uint16
	Title      string
	uRLPtr     uint64
	namespace  byte
	url        string
	blob       uint32
	cluster    uint32
	RedirectTo *Article
	z          *ZimReader
}

// Fill an article with datas found at offset
func (z *ZimReader) FillArticleAt(a *Article, offset uint64) *Article {
	a.z = z
	a.uRLPtr = offset

	mimeIdx, err := readInt16(z.getBytesRangeAt(offset, offset+2))
	if err != nil {
		panic(err)
	}
	a.EntryType = mimeIdx

	// Linktarget or Target Entry
	if mimeIdx == LinkTargetEntry || mimeIdx == DeletedEntry {
		//TODO
		return nil
	}

	s := z.getBytesRangeAt(offset+3, offset+4)
	a.namespace = s[0]

	a.cluster, err = readInt32(z.getBytesRangeAt(offset+8, offset+8+4))
	if err != nil {
		panic(err)
	}

	a.blob, err = readInt32(z.getBytesRangeAt(offset+12, offset+12+4))
	if err != nil {
		panic(err)
	}

	// Redirect
	if mimeIdx == RedirectEntry {
		// check for a possible loop: the redirect could point to the same target
		if z.getURLOffsetAtIdx(a.cluster) != offset {
			// redirect ptr share the same memory offset than Cluster number
			a.RedirectTo = z.getArticleAt(z.getURLOffsetAtIdx(a.cluster))
		}

		// assume the url + title won't be longer than 2k
		b := bytes.NewBuffer(z.getBytesRangeAt(offset+12, offset+12+2048))
		a.url, err = b.ReadString('\x00')
		if err != nil {
			panic(err)
		}
		a.url = strings.TrimRight(string(a.url), "\x00")

		a.Title, err = b.ReadString('\x00')
		if err != nil {
			panic(err)
		}
		a.Title = strings.TrimRight(string(a.Title), "\x00")

		return a
	}

	b := bytes.NewBuffer(z.getBytesRangeAt(offset+16, offset+16+2048))
	a.url, err = b.ReadString('\x00')
	if err != nil {
		panic(err)
	}
	a.url = strings.TrimRight(string(a.url), "\x00")

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
func (a *Article) Data() []byte {
	// ensure we have data to read
	if a.EntryType == RedirectEntry || a.EntryType == LinkTargetEntry || a.EntryType == DeletedEntry {
		return nil
	}
	start, end := a.z.getClusterOffsetsAtIdx(a.cluster)
	s := a.z.getBytesRangeAt(start, start+1)
	compression := uint8(s[0])

	// blob starts at offset, blob ends at offset
	var bs, be uint32

	// LZMA
	if compression == 4 {
		b := bytes.NewBuffer(a.z.getBytesRangeAt(start+1, end+1))
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

		bs, err = readInt32(blob[a.blob*4 : a.blob*4+4])
		if err != nil {
			panic(err)
		}

		be, err = readInt32(blob[a.blob*4+4 : a.blob*4+4+4])
		if err != nil {
			panic(err)
		}

		return blob[bs:be]

	} else if compression == 0 || compression == 1 {
		// un compresssed
		startPos := start + 1
		blobOffset := uint64(a.blob * 4)

		bs, err := readInt32(a.z.getBytesRangeAt(startPos+blobOffset, startPos+blobOffset+4))
		if err != nil {
			panic(err)
		}

		be, err = readInt32(a.z.getBytesRangeAt(startPos+blobOffset+4, startPos+blobOffset+4+4))
		if err != nil {
			panic(err)
		}

		return a.z.getBytesRangeAt(startPos+uint64(bs), startPos+uint64(be))
	}

	fmt.Println("Unhandled compression")

	return nil
}

func (a *Article) MimeType() string {
	if a.EntryType == RedirectEntry || a.EntryType == LinkTargetEntry || a.EntryType == DeletedEntry {
		return ""
	}

	return a.z.mimeTypeList[a.EntryType]
}

// return the url prefixed by the namespace
func (a *Article) FullURL() string {
	return string(a.namespace) + "/" + a.url
}

func (a *Article) String() string {
	return fmt.Sprintf("Mime: 0x%x URL: [%s], Title: [%s], Cluster: 0x%x Blob: 0x%x",
		a.EntryType, a.FullURL(), a.Title, a.cluster, a.blob)
}

func (a *Article) getBlobOffsetsAtIdx(z *ZimReader) (start, end uint64) {
	idx := a.blob
	offset := z.clusterPtrPos + uint64(idx)*8
	start, err := readInt64(z.getBytesRangeAt(offset, offset+8))
	if err != nil {
		panic(err)
	}
	offset = z.clusterPtrPos + uint64(idx+1)*8
	end, err = readInt64(z.getBytesRangeAt(offset, offset+8))
	if err != nil {
		panic(err)
	}
	return
}
