package zim

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"syscall"
)

const (
	zimHeader = 72173914
)

// ZimReader keep tracks of everything related to ZIM reading
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
	size          int64
	// this mutex is used on 32 bits system only to securise access to one mmap at a time
	sync.Mutex
	// currentSeg indicates which segment is currently mmaped
	currentSeg int
	// totalSeg indicates how many chunks are used to slice the zim file (file size / chunkSize)
	totalSeg int
	// chunkSize needs to be a multiple of os.Getpagesize() for mmap
	segSize int
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

	z.size = fi.Size()

	// informing now segment are loaded yet
	z.currentSeg = -1

	// if the file is smaller than 2GB or running on amd64 (so 64 bits) use mmap to read the file
	if uint64(z.size) < uint64(z.segSize) || runtime.GOARCH == "amd64" {
		mmap, err := syscall.Mmap(int(f.Fd()), 0, int(z.size), syscall.PROT_READ, syscall.MAP_PRIVATE)
		if err != nil {
			panic(err)
		}
		z.currentSeg = 0
		z.totalSeg = 1
		z.mmap = mmap
	} else {
		// 32-bit architecture such as Intel's IA-32 can only directly address 4 GiB or smaller portions
		// of files. An even smaller amount of addressible space is available to individual programs
		// typically in the range of 2 to 3 GiB, depending on the operating system kernel.

		// we need a multiple of page size smaller than 1 << 31 for mmap segments
		pc := (1 << 31) / os.Getpagesize()
		z.segSize = pc * os.Getpagesize()

		z.totalSeg = int(fi.Size()/int64(z.segSize)) + 1

		fmt.Printf("Using 32 bits addressing %d segments of %d pageSize %d\n", z.totalSeg, z.segSize, os.Getpagesize())

		// mmap the 1st segment
		err = z.mmapSliceIdx(0)
		if err != nil {
			panic(err)
		}
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
			offset := z.getURLOffsetAtIdx(idx)
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

// return the article at the exact url not using any index this is really slow on big ZIM
func (z *ZimReader) GetPageNoIndex(url string) *Article {
	var idx uint32
	// starting at 1 to avoid "con" entry
	var start uint32 = 1

	art := new(Article)
	for idx = start; idx < z.ArticleCount; idx++ {
		offset := z.getURLOffsetAtIdx(idx)
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
	return z.getArticleAt(z.getURLOffsetAtIdx(z.mainPage))
}

// Close & cleanup the zimreader
func (z *ZimReader) Close() {
	syscall.Munmap(z.mmap)
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

// getBytesRangeAt returns bytes from start to end
// it's needed to abstract mmap usages rather than read directly on the mmap slices
func (z *ZimReader) getBytesRangeAt(start, end uint64) []byte {
	if z.totalSeg == 1 {
		return z.mmap[start:end]
	}
	fns := int(start) / z.segSize
	fne := int(end) / z.segSize

	z.Lock()
	defer z.Unlock()

	if fns != z.currentSeg {
		err := z.mmapSliceIdx(fns)
		if err != nil {
			panic(err)
		}
	}

	if fns == fne {
		return z.mmap[start%uint64(z.segSize) : end%uint64(z.segSize)]
	}

	fmt.Println("end is on the over file not implemented yet")

	return nil
}

func (z *ZimReader) getByteAt(offset uint64) byte {
	if z.totalSeg == 1 {
		return z.mmap[offset]
	}

	z.Lock()
	defer z.Unlock()

	seg := int(offset) / z.segSize

	if seg != z.currentSeg {
		err := z.mmapSliceIdx(seg)
		if err != nil {
			panic(err)
		}
	}

	return z.mmap[offset%uint64(z.segSize)]
}

func (z *ZimReader) mmapSliceIdx(idx int) error {
	if idx == z.currentSeg {
		return nil
	}

	if idx >= z.totalSeg {
		return errors.New("idx higher than total segments")
	}

	syscall.Munmap(z.mmap)

	// if this is the last segment mmap the correct size
	mmapSize := z.segSize

	if idx == z.totalSeg-1 {
		mmapSize = int(z.size)%z.segSize + os.Getpagesize()
	}

	mmap, err := syscall.Mmap(int(z.f.Fd()), int64(idx*z.segSize), mmapSize, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		panic(err)
	}
	z.currentSeg = idx
	z.mmap = mmap
	return nil
}

// populate the ZimReader structs with headers
// It is decided to panic on corrupted zim file
func (z *ZimReader) readFileHeaders() error {
	// checking for file type
	v, err := readInt32(z.getBytesRangeAt(0, 0+4))
	if err != nil {
		panic(err)
	}
	if v != zimHeader {
		return errors.New("not a ZIM file")
	}

	// checking for version
	v, err = readInt32(z.getBytesRangeAt(4, 4+4))
	if err != nil {
		panic(err)
	}
	if v != 5 {
		return errors.New("unsupported version, 5 only")
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

// get the offset in the zim pointing to URL at index pos
func (z *ZimReader) getURLOffsetAtIdx(idx uint32) uint64 {
	offset := z.urlPtrPos + uint64(idx)*8
	v, err := readInt64(z.getBytesRangeAt(offset, offset+8))
	if err != nil {
		panic(err)
	}
	return v
}

// return data at start and end offsets for cluster idx
func (z *ZimReader) getClusterOffsetsAtIdx(idx uint32) (start, end uint64) {
	offset := z.clusterPtrPos + (uint64(idx) * 8)
	start, err := readInt64(z.getBytesRangeAt(offset, offset+8))
	if err != nil {
		panic(err)
	}
	offset = z.clusterPtrPos + (uint64(idx+1) * 8)
	end, err = readInt64(z.getBytesRangeAt(offset, offset+8))
	if err != nil {
		panic(err)
	}
	end--
	return
}
