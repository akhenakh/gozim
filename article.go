package zim

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"

	xz "github.com/remyoudompheng/go-liblzma"
)

const (
	RedirectEntry   uint16 = 0xffff
	LinkTargetEntry        = 0xfffe
	DeletedEntry           = 0xfffd
)

var articlePool sync.Pool

type Article struct {
	// EntryType is a RedirectEntry/LinkTargetEntry/DeletedEntry or an idx
	// pointing to ZimReader.mimeTypeList
	EntryType uint16
	Title     string
	URLPtr    uint64
	Namespace byte
	url       string
	blob      uint32
	cluster   uint32
	z         *ZimReader
}

func init() {
	articlePool = sync.Pool{
		New: func() interface{} {
			return new(Article)
		},
	}
}

// convenient method to return the Article at URL index idx
func (z *ZimReader) ArticleAtURLIdx(idx uint32) (*Article, error) {
	o, err := z.OffsetAtURLIdx(idx)
	if err != nil {
		return nil, err
	}
	return z.ArticleAt(o)
}

// return the article main page if it exists
func (z *ZimReader) MainPage() (*Article, error) {
	if z.mainPage == 0xffffffff {
		return nil, nil
	}
	return z.ArticleAtURLIdx(z.mainPage)
}

// get the article (Directory) pointed by the offset found in URLpos or Titlepos
func (z *ZimReader) ArticleAt(offset uint64) (*Article, error) {
	a := articlePool.Get().(*Article)
	err := z.FillArticleAt(a, offset)
	return a, err
}

// Fill an article with datas found at offset
func (z *ZimReader) FillArticleAt(a *Article, offset uint64) error {
	a = a
	a.z = z
	a.URLPtr = offset

	mimeIdx, err := readInt16(z.bytesRangeAt(offset, offset+2))
	a.EntryType = mimeIdx

	// Linktarget or Target Entry
	if mimeIdx == LinkTargetEntry || mimeIdx == DeletedEntry {
		//TODO
		return nil
	}

	s, err := z.bytesRangeAt(offset+3, offset+4)
	if err != nil {
		return err
	}
	a.Namespace = s[0]

	a.cluster, err = readInt32(z.bytesRangeAt(offset+8, offset+8+4))
	if err != nil {
		return err
	}
	a.blob, err = readInt32(z.bytesRangeAt(offset+12, offset+12+4))
	if err != nil {
		return err
	}

	// Redirect
	if mimeIdx == RedirectEntry {
		// assume the url + title won't be longer than 2k
		b, err := z.bytesRangeAt(offset+12, offset+12+2048)
		if err != nil {
			return nil
		}
		bbuf := bytes.NewBuffer(b)
		a.url, err = bbuf.ReadString('\x00')
		if err != nil {
			return err
		}
		a.url = strings.TrimRight(string(a.url), "\x00")

		a.Title, err = bbuf.ReadString('\x00')
		if err != nil {
			return err
		}
		a.Title = strings.TrimRight(string(a.Title), "\x00")
		return err
	}

	b, err := z.bytesRangeAt(offset+16, offset+16+2048)
	if err != nil {
		return nil
	}
	bbuf := bytes.NewBuffer(b)
	a.url, err = bbuf.ReadString('\x00')
	if err != nil {
		return err
	}

	a.url = strings.TrimRight(string(a.url), "\x00")

	title, err := bbuf.ReadString('\x00')
	if err != nil {
		return err
	}
	title = strings.TrimRight(string(title), "\x00")
	// This is a trick to force a copy and avoid retain of the full buffer
	// mainly for indexing title reasons
	if len(title) != 0 {
		a.Title = title[0:1] + title[1:]
	}
	return nil
}

// return the uncompressed data associated with this article
func (a *Article) Data() ([]byte, error) {
	// ensure we have data to read
	if a.EntryType == RedirectEntry || a.EntryType == LinkTargetEntry || a.EntryType == DeletedEntry {
		return nil, nil
	}
	start, end, err := a.z.clusterOffsetsAtIdx(a.cluster)
	if err != nil {
		return nil, err
	}
	s, err := a.z.bytesRangeAt(start, start+1)
	if err != nil {
		return nil, err
	}
	compression := uint8(s[0])

	// blob starts at offset, blob ends at offset
	var bs, be uint32

	// LZMA
	if compression == 4 {
		b, err := a.z.bytesRangeAt(start+1, end+1)
		if err != nil {
			return nil, err
		}
		bbuf := bytes.NewBuffer(b)
		dec, err := xz.NewReader(bbuf)
		defer dec.Close()
		if err != nil {
			return nil, err
		}

		// the decoded chunk are around 1MB
		// TODO: on smaller devices need to read stream rather than ReadAll
		blob, err := ioutil.ReadAll(dec)
		if err != nil {
			return nil, err
		}

		bs, err = readInt32(blob[a.blob*4:a.blob*4+4], nil)
		if err != nil {
			return nil, err
		}
		be, err = readInt32(blob[a.blob*4+4:a.blob*4+4+4], nil)
		if err != nil {
			return nil, err
		}

		// avoid retaining all the chunk
		c := make([]byte, be-bs)
		copy(c, blob[bs:be])
		return c, nil

	} else if compression == 0 || compression == 1 {
		// un compresssed
		startPos := start + 1
		blobOffset := uint64(a.blob * 4)

		bs, err := readInt32(a.z.bytesRangeAt(startPos+blobOffset, startPos+blobOffset+4))
		if err != nil {
			return nil, err
		}

		be, err := readInt32(a.z.bytesRangeAt(startPos+blobOffset+4, startPos+blobOffset+4+4))
		if err != nil {
			return nil, err
		}

		return a.z.bytesRangeAt(startPos+uint64(bs), startPos+uint64(be))
	}

	return nil, errors.New("Unhandled compression")
}

func (a *Article) MimeType() string {
	if a.EntryType == RedirectEntry || a.EntryType == LinkTargetEntry || a.EntryType == DeletedEntry {
		return ""
	}

	return a.z.mimeTypeList[a.EntryType]
}

// return the url prefixed by the namespace
func (a *Article) FullURL() string {
	return string(a.Namespace) + "/" + a.url
}

func (a *Article) String() string {
	return fmt.Sprintf("Mime: 0x%x URL: [%s], Title: [%s], Cluster: 0x%x Blob: 0x%x",
		a.EntryType, a.FullURL(), a.Title, a.cluster, a.blob)
}

// RedirectIndex return the redirect index of RedirectEntry type article
// return an err if not a redirect entry
func (a *Article) RedirectIndex() (uint32, error) {
	if a.EntryType != RedirectEntry {
		return 0, errors.New("Not a RedirectEntry")
	}
	// We use the cluster to save the redirect index position for RedirectEntry type
	return a.cluster, nil
}

func (a *Article) blobOffsetsAtIdx(z *ZimReader) (start, end uint64) {
	idx := a.blob
	offset := z.clusterPtrPos + uint64(idx)*8
	start, err := readInt64(z.bytesRangeAt(offset, offset+8))
	if err != nil {
		return
	}
	offset = z.clusterPtrPos + uint64(idx+1)*8
	end, err = readInt64(z.bytesRangeAt(offset, offset+8))

	return
}
