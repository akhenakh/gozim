package zim

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	zimHeader = 72173914
)

// ZimReader keep tracks of everything related to ZIM reading
type ZimReader struct {
	f             *os.File
	ArticleCount  uint32
	clusterCount  uint32
	urlPtrPos     uint64
	titlePtrPos   uint64
	clusterPtrPos uint64
	mimeListPos   uint64
	mainPage      uint32
	layoutPage    uint32
	mimeTypeList  []string
	mmap          []byte
}

// create a new zim reader
func NewReader(path string, mmap bool) (*ZimReader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	z := ZimReader{f: f, mainPage: 0xffffffff, layoutPage: 0xffffffff}

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	size := fi.Size()

	if mmap {
		// we need a multiple of page size bigger than the file
		pc := size / int64(os.Getpagesize())
		totalMmap := pc*int64(os.Getpagesize()) + int64(os.Getpagesize())
		if (size % int64(os.Getpagesize())) == 0 {
			totalMmap = size
		}

		mmap, err := NewMmap(int(f.Fd()), 0, int(totalMmap))
		if err != nil {
			return nil, err
		}
		z.mmap = mmap
	}

	err = z.readFileHeaders()
	return &z, err
}

// Return an ordered list of mime types present in the ZIM file
func (z *ZimReader) MimeTypes() []string {
	if len(z.mimeTypeList) != 0 {
		return z.mimeTypeList
	}

	var s []string
	// assume mime list fit in 2k
	b, err := z.bytesRangeAt(z.mimeListPos, z.mimeListPos+2048)
	if err != nil {
		return s
	}
	bbuf := bytes.NewBuffer(b)

	for {
		line, err := bbuf.ReadBytes('\x00')
		if err != nil && err != io.EOF {
			return s
		}
		// a line of 1 is a line containing only \x00 and it's the marker for the
		// end of mime types list
		if len(line) == 1 {
			break
		}
		s = append(s, strings.TrimRight(string(line), "\x00"))
	}
	z.mimeTypeList = s
	return s
}

// list all articles, using url index, contained in a zim file
// note that this is a slow implementation, a real iterator is faster
// you are not suppose to use this method on big zim files, use indexes
func (z *ZimReader) ListArticles() <-chan *Article {
	ch := make(chan *Article, 10)

	go func() {
		var idx uint32
		// starting at 1 to avoid "con" entry
		var start uint32 = 1

		for idx = start; idx < z.ArticleCount; idx++ {
			art, err := z.ArticleAtURLIdx(idx)
			if err != nil {
				continue
			}

			if art == nil {
				//TODO: deal with redirect continue
			}
			ch <- art
		}
		close(ch)
	}()
	return ch
}

// list all title pointer, Titles by position contained in a zim file
// Titles are pointers to URLpos index, usefull for indexing cause smaller to store: uint32
// note that this is a slow implementation, a real iterator is faster
// you are not suppose to use this method on big zim files prefer ListTitlesPtrIterator to build your index
func (z *ZimReader) ListTitlesPtr() <-chan uint32 {
	ch := make(chan uint32, 10)

	go func() {
		var pos uint64
		var count uint32

		for pos = z.titlePtrPos; count < z.ArticleCount; pos += 4 {
			idx, err := readInt32(z.bytesRangeAt(pos, pos+4))
			if err != nil {
				continue
			}
			ch <- idx
			count++
		}
		close(ch)
	}()
	return ch
}

// list all title pointer, Titles by position contained in a zim file
// Titles are pointers to URLpos index, usefull for indexing cause smaller to store: uint32
func (z *ZimReader) ListTitlesPtrIterator(cb func(uint32)) {
	var count uint32
	for pos := z.titlePtrPos; count < z.ArticleCount; pos += 4 {
		idx, err := readInt32(z.bytesRangeAt(pos, pos+4))
		if err != nil {
			continue
		}
		cb(idx)
		count++
	}
}

// return the article at the exact url not using any index
func (z *ZimReader) GetPageNoIndex(url string) (*Article, error) {
	// starting at 1 to avoid "con" entry
	var start uint32
	stop := z.ArticleCount

	a := new(Article)

	for {
		pos := (start + stop) / 2

		offset, err := z.OffsetAtURLIdx(pos)
		if err != nil {
			return nil, err
		}
		err = z.FillArticleAt(a, offset)
		if err != nil {
			return nil, err
		}

		if a.FullURL() == url {
			return a, nil
		}

		if a.FullURL() > url {
			stop = pos
		} else {
			start = pos
		}
		if stop-start == 1 {
			break
		}

	}
	return nil, errors.New("article not found")
}

// get the offset pointing to Article at pos in the URL idx
func (z *ZimReader) OffsetAtURLIdx(idx uint32) (uint64, error) {
	offset := z.urlPtrPos + uint64(idx)*8
	return readInt64(z.bytesRangeAt(offset, offset+8))
}

// Close & cleanup the zimreader
func (z *ZimReader) Close() error {
	return z.f.Close()
}

func (z *ZimReader) String() string {
	fi, err := z.f.Stat()
	if err != nil {
		return "corrupted zim"
	}
	return fmt.Sprintf("Size: %d, ArticleCount: %d urlPtrPos: 0x%x titlePtrPos: 0x%x mimeListPos: 0x%x clusterPtrPos: 0x%x\nMimeTypes: %v",
		fi.Size(), z.ArticleCount, z.urlPtrPos, z.titlePtrPos, z.mimeListPos, z.clusterPtrPos, z.MimeTypes())
}

// getBytesRangeAt returns bytes from start to end
// it's needed to abstract mmap usages rather than read directly on the mmap slices
func (z *ZimReader) bytesRangeAt(start, end uint64) ([]byte, error) {
	if len(z.mmap) > 0 {
		return z.mmap[start:end], nil
	}

	buf := make([]byte, end-start)
	n, err := z.f.ReadAt(buf, int64(start))
	if err != nil {
		return nil, err
	}

	if n != int(end-start) {
		return nil, errors.New("can't read enough bytes")
	}

	return buf, nil
}

// populate the ZimReader structs with headers
func (z *ZimReader) readFileHeaders() error {
	// checking for file type
	v, err := readInt32(z.bytesRangeAt(0, 0+4))
	if err != nil || v != zimHeader {
		return errors.New("not a ZIM file")
	}

	// checking for version
	v, err = readInt32(z.bytesRangeAt(4, 4+4))
	if err != nil || v != 5 {
		return errors.New("unsupported version, 5 only")
	}

	// checking for articles count
	v, err = readInt32(z.bytesRangeAt(24, 24+4))
	if err != nil {
		return err
	}
	z.ArticleCount = v

	// checking for cluster count
	v, err = readInt32(z.bytesRangeAt(28, 28+4))
	if err != nil {
		return err
	}
	z.clusterCount = v

	// checking for urlPtrPos
	vb, err := readInt64(z.bytesRangeAt(32, 32+8))
	if err != nil {
		return err
	}
	z.urlPtrPos = vb

	// checking for titlePtrPos
	vb, err = readInt64(z.bytesRangeAt(40, 40+8))
	if err != nil {
		return err
	}
	z.titlePtrPos = vb

	// checking for clusterPtrPos
	vb, err = readInt64(z.bytesRangeAt(48, 48+8))
	if err != nil {
		return err
	}
	z.clusterPtrPos = vb

	// checking for mimeListPos
	vb, err = readInt64(z.bytesRangeAt(56, 56+8))
	if err != nil {
		return err
	}
	z.mimeListPos = vb

	// checking for mainPage
	v, err = readInt32(z.bytesRangeAt(64, 64+4))
	if err != nil {
		return err
	}
	z.mainPage = v

	// checking for layoutPage
	v, err = readInt32(z.bytesRangeAt(68, 68+4))
	if err != nil {
		return err
	}
	z.layoutPage = v

	z.MimeTypes()
	return nil
}

// return start and end offsets for cluster at index idx
func (z *ZimReader) clusterOffsetsAtIdx(idx uint32) (start, end uint64, err error) {
	offset := z.clusterPtrPos + (uint64(idx) * 8)
	start, err = readInt64(z.bytesRangeAt(offset, offset+8))
	if err != nil {
		return
	}
	offset = z.clusterPtrPos + (uint64(idx+1) * 8)
	end, err = readInt64(z.bytesRangeAt(offset, offset+8))
	end--
	return
}
