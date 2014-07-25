package zim

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"syscall"
)

const (
	ZIM = 72173914
)

type ZimReader struct {
	f             *os.File
	mmap          [][]byte
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

// create a new zim reader
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
	z.mmap = make([][]byte, 1)

	if int(fi.Size()) < 1<<31 || runtime.GOARCH == "amd64" {
		mmap, err := syscall.Mmap(int(f.Fd()), 0, int(fi.Size()), syscall.PROT_READ, syscall.MAP_SHARED)
		if err != nil {
			panic(err)
		}
		z.mmap[0] = mmap
	} else {
		// 32-bit architecture such as Intel's IA-32 can only directly address 4 GiB or smaller portions
		// of files. An even smaller amount of addressible space is available to individual programs
		// typically in the range of 2 to 3 GiB, depending on the operating system kernel.

	}

	err = z.readFileHeaders()
	return &z, err
}

func (z *ZimReader) getBytesRangeAt(start, end uint64) []byte {
	if len(z.mmap) == 1 {
		return z.mmap[0][start:end]
	}
	fns := start / 1 << 31
	fne := end / 1 << 31
	if fns == fne {
		return z.mmap[fns][start%1<<31 : end%1<<31]
	}

	//TODO: end is on another segment
	return nil
}

func (z *ZimReader) getByteAt(offset uint64) byte {
	if len(z.mmap) == 1 {
		return z.mmap[0][offset]
	}

	fn := offset / 1 << 31
	return z.mmap[fn][offset%1<<31]
}

// populate the ZimReader structs with headers
// It is decided to panic on corrupted zim file
func (z *ZimReader) readFileHeaders() error {
	// checking for file type
	v, err := readInt32(z.getBytesRangeAt(0, 0+4))
	if err != nil {
		panic(err)
	}
	if v != ZIM {
		return errors.New("Not a ZIM file")
	}

	// checking for version
	v, err = readInt32(z.getBytesRangeAt(4, 4+4))
	if err != nil {
		panic(err)
	}
	if v != 5 {
		return errors.New("Unsupported version 5 only")
	}

	// checking for articles count
	v, err = readInt32(z.getBytesRangeAt(24, 24+4))
	if err != nil {
		panic(err)
	}
	z.ArticleCount = v

	// checking for cluster count
	v, err = readInt32(z.getBytesRangeAt(28, 28+4))
	if err != nil {
		panic(err)
	}
	z.clusterCount = v

	// checking for urlPtrPos
	vb, err := readInt64(z.getBytesRangeAt(32, 32+8))
	if err != nil {
		panic(err)
	}
	z.urlPtrPos = vb

	// checking for titlePtrPos
	vb, err = readInt64(z.getBytesRangeAt(40, 40+8))
	if err != nil {
		panic(err)
	}
	z.titlePtrPos = vb

	// checking for clusterPtrPos
	vb, err = readInt64(z.getBytesRangeAt(48, 48+8))
	if err != nil {
		panic(err)
	}
	z.clusterPtrPos = vb

	// checking for mimeListPos
	vb, err = readInt64(z.getBytesRangeAt(56, 56+8))
	if err != nil {
		panic(err)
	}
	z.mimeListPos = vb

	// checking for mainPage
	v, err = readInt32(z.getBytesRangeAt(64, 64+4))
	if err != nil {
		panic(err)
	}
	z.mainPage = v

	// checking for layoutPage
	v, err = readInt32(z.getBytesRangeAt(68, 68+4))
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

	var s []string
	// assume mime list fit in 2k
	b := bytes.NewBuffer(z.getBytesRangeAt(z.mimeListPos, z.mimeListPos+2048))

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

// list all urls contained in a zim file
// note that this is a slow implementation, a real iterator is faster
// but you are not suppose to use this method on big zim files, use indexes
func (z *ZimReader) ListArticles() <-chan *Article {
	ch := make(chan *Article, 10)

	go func() {
		var idx uint32
		// starting at 1 to avoid "con" entry
		var start uint32 = 1

		for idx = start; idx < z.ArticleCount; idx++ {
			offset := z.GetUrlOffsetAtIdx(idx)
			art := z.getArticleAt(offset)
			if art == nil {
				//TODO: deal with redirect continue
			}
			ch <- art
		}
		close(ch)
	}()
	return ch
}

// get the offset in the zim pointing to URL at index pos
func (z *ZimReader) GetUrlOffsetAtIdx(idx uint32) uint64 {
	offset := z.urlPtrPos + uint64(idx)*8
	v, err := readInt64(z.getBytesRangeAt(offset, offset+8))
	if err != nil {
		panic(err)
	}
	return v
}

// return the start and end offsets for cluster idx
func (z *ZimReader) getClusterOffsetsAtIdx(idx uint32) (start, end uint64) {
	offset := z.clusterPtrPos + (uint64(idx) * 8)
	start, err := readInt64(z.getBytesRangeAt(offset, offset+8))
	if err != nil {
		panic(err)
	}
	fmt.Println(start)
	offset = z.clusterPtrPos + (uint64(idx+1) * 8)
	end, err = readInt64(z.getBytesRangeAt(offset, offset+8))
	if err != nil {
		panic(err)
	}
	end -= 1
	return
}

// return the article at the exact url not using any index this is really slow on big ZIM
func (z *ZimReader) GetPageNoIndex(url string) *Article {
	var idx uint32
	// starting at 1 to avoid "con" entry
	var start uint32 = 1

	art := new(Article)
	for idx = start; idx < z.ArticleCount; idx++ {
		offset := z.GetUrlOffsetAtIdx(idx)
		art = z.FillArticleAt(art, offset)
		if art.FullURL() == url {
			return art
		}
	}
	return nil
}

// return the article main page if it exists
func (z *ZimReader) GetMainPage() *Article {
	if z.mainPage == 0xffffffff {
		return nil
	}
	return z.getArticleAt(z.GetUrlOffsetAtIdx(z.mainPage))
}

func (z *ZimReader) Close() {
	for _, m := range z.mmap {
		syscall.Munmap(m)
	}
	z.f.Close()
}

func (z *ZimReader) String() string {
	fi, err := z.f.Stat()
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("Size: %d, ArticleCount: %d urlPtrPos: 0x%x titlePtrPos: 0x%x mimeListPos: 0x%x clusterPtrPos: 0x%x\nMimeTypes: %v",
		fi.Size(), z.ArticleCount, z.urlPtrPos, z.titlePtrPos, z.mimeListPos, z.clusterPtrPos, z.MimeTypes())
}
