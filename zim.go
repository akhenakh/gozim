package zim

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
)

type DirType int8

const (
	ArticleType DirType = iota
	RedirectType
	LinktargetType
)

type Article struct {
	URLPtr    uint64
	Mimetype  uint16
	Namespace byte
	URL       string
	Title     string
	Blob      uint32
	Cluster   uint32
}

const (
	ZIM = 72173914
)

type ZimReader struct {
	f             *os.File
	mmap          []byte
	ArticleCount  uint32
	clusterCount  uint32
	urlPtrPos     uint64
	titlePtrPos   uint64
	clusterPtrPos uint64
	mimeListPos   uint64
	mainPage      uint32
	layoutPage    uint32
	mimeTypeList  []string
}

func NewReader(path string) (*ZimReader, error) {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	z := ZimReader{f: f, mainPage: 0xffffffff, layoutPage: 0xffffffff}

	fi, err := f.Stat()
	if err != nil {
		panic(err)
	}

	mmap, err := syscall.Mmap(int(f.Fd()), 0, int(fi.Size()), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		panic(err)
	}
	z.mmap = mmap
	err = z.readFileHeaders()
	return &z, err
}

func (z *ZimReader) readFileHeaders() error {
	// checking for file type
	v, err := readInt32(z.mmap[0:4])
	if err != nil {
		panic(err)
	}
	if v != ZIM {
		return errors.New("Not a ZIM file")
	}

	// checking for version
	v, err = readInt32(z.mmap[4:9])
	if err != nil {
		panic(err)
	}
	if v != 5 {
		return errors.New("Unsupported version 5 only")
	}

	// checking for articles count
	v, err = readInt32(z.mmap[24:29])
	if err != nil {
		panic(err)
	}
	z.ArticleCount = v

	// checking for cluster count
	v, err = readInt32(z.mmap[28:33])
	if err != nil {
		panic(err)
	}
	z.clusterCount = v

	// checking for urlPtrPos
	vb, err := readInt64(z.mmap[32:41])
	if err != nil {
		panic(err)
	}
	z.urlPtrPos = vb

	// checking for titlePtrPos
	vb, err = readInt64(z.mmap[40:49])
	if err != nil {
		panic(err)
	}
	z.titlePtrPos = vb

	// checking for clusterPtrPos
	vb, err = readInt64(z.mmap[48:57])
	if err != nil {
		panic(err)
	}
	z.clusterPtrPos = vb

	// checking for mimeListPos
	vb, err = readInt64(z.mmap[56:65])
	if err != nil {
		panic(err)
	}
	z.mimeListPos = vb

	// checking for mainPage
	v, err = readInt32(z.mmap[64:69])
	if err != nil {
		panic(err)
	}
	z.mainPage = v

	// checking for layoutPage
	v, err = readInt32(z.mmap[68:73])
	if err != nil {
		panic(err)
	}
	z.layoutPage = v

	z.MimeTypes()
	return err
}

// Return an ordered list of mime types present in the ZIM file
func (z *ZimReader) MimeTypes() []string {
	if len(z.mimeTypeList) != 0 {
		return z.mimeTypeList
	}

	s := make([]string, 1, 1)

	b := bytes.NewBuffer(z.mmap[z.mimeListPos:])

	for {
		line, err := b.ReadBytes('\x00')
		if err != nil && err != io.EOF {
			panic(err)
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

func (z *ZimReader) ListTitles() (func() (string, bool), bool) {
	var idx uint64 = 0
	return func() (string, bool) {
		prev_idx := idx
		idx++
		return z.getTitleAt(prev_idx), (idx < uint64(z.ArticleCount))
	}, (idx < uint64(z.ArticleCount))
}

// note that this is a slow implementation a real iterator is faster
// but you are not suppose to use this method on big zim files use indexes
func (z *ZimReader) ListUrls() <-chan string {
	ch := make(chan string, 10)
	go func() {
		var i uint64
		for i = 0; i < uint64(z.ArticleCount); i++ {
			offset := z.GetUrlAtIdx(i)
			art := z.getArticleAt(offset)
			ch <- art.Title
		}
		close(ch) // Remember to close or the loop never ends!
	}()
	return ch
}

func (z *ZimReader) GetUrlAtIdx(pos uint64) uint64 {
	offset := z.urlPtrPos + pos*8
	v, err := readInt64(z.mmap[offset : offset+8])
	if err != nil {
		panic(err)
	}
	return v
}

// get the article (Directory) pointed by the offset found in URLpos or Titlepos
func (z *ZimReader) getArticleAt(offset uint64) (art Article) {
	a := Article{}
	a.URLPtr = offset

	mimeIdx, err := readInt16(z.mmap[offset : offset+2])
	if err != nil {
		panic(err)
	}
	a.Mimetype = mimeIdx

	// Redirect
	if mimeIdx == 0xffff {
		//TODO
		return
	}
	// Linktarget or Target Entry
	if mimeIdx == 0xfffe || mimeIdx == 0xfffd {
		//TODO
		return
	}

	//mimeType := z.mimeTypeList[mimeIdx]
	a.Namespace = z.mmap[offset+3]
	a.Cluster, err = readInt32(z.mmap[offset+8 : offset+8+4])
	if err != nil {
		panic(err)
	}

	a.Blob, err = readInt32(z.mmap[offset+12 : offset+12+4])
	if err != nil {
		panic(err)
	}

	b := bytes.NewBuffer(z.mmap[offset+16:])
	a.URL, err = b.ReadString('\x00')
	if err != nil {
		panic(err)
	}

	return
}

func (z *ZimReader) getTitleAt(pos uint64) string {
	offset := z.titlePtrPos + (pos-1)*4

	v, err := readInt32(z.mmap[offset : offset+4])
	if err != nil {
		panic(err)
	}

	b := bytes.NewBuffer(z.mmap[v:])
	title, err := b.ReadBytes('\x00')
	if err != nil {
		panic(err)
	}
	return string(title)
}

func (z *ZimReader) Close() {
	syscall.Munmap(z.mmap)
	z.f.Close()
}

func (z *ZimReader) String() string {
	fi, err := z.f.Stat()
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("Size: %d, ArticleCount: %d urlPtrPos: 0x%x mimeListPos: 0x%x\nMimeTypes: %v",
		fi.Size(), z.ArticleCount, z.urlPtrPos, z.mimeListPos, z.MimeTypes())
}
