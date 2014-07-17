package zim

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"unicode/utf8"
)

const (
	ZIM = 72173914
)

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
}

func NewReader(path string) (*ZimReader, error) {
	f, err := os.Open(path)
	z := ZimReader{f: f, mainPage: 0xffffffff, layoutPage: 0xffffffff}
	if err != nil {
		return nil, err
	}

	err = z.readFileHeaders()
	return &z, err
}

func (z *ZimReader) readFileHeaders() error {
	// get size
	z.f.Stat()
	_, err := z.f.Seek(0, 0)
	if err != nil {
		panic(err)
	}

	b := make([]byte, 1024)

	_, err = z.f.Read(b)
	if err != nil {
		panic(err)
	}

	// checking for file type
	v, err := readInt32(b[0:4])
	if err != nil {
		panic(err)
	}
	if v != ZIM {
		return errors.New("Not a ZIM file")
	}

	// checking for version
	v, err = readInt32(b[4:9])
	if err != nil {
		panic(err)
	}
	if v != 5 {
		return errors.New("Unsupported version 5 only")
	}

	// checking for articles count
	v, err = readInt32(b[24:29])
	if err != nil {
		panic(err)
	}
	z.ArticleCount = v

	// checking for cluster count
	v, err = readInt32(b[28:33])
	if err != nil {
		panic(err)
	}
	z.clusterCount = v

	// checking for urlPtrPos
	vb, err := readInt64(b[32:41])
	if err != nil {
		panic(err)
	}
	z.urlPtrPos = vb

	// checking for titlePtrPos
	vb, err = readInt64(b[40:49])
	if err != nil {
		panic(err)
	}
	z.titlePtrPos = vb

	// checking for clusterPtrPos
	vb, err = readInt64(b[48:57])
	if err != nil {
		panic(err)
	}
	z.clusterPtrPos = vb

	// checking for mimeListPos
	vb, err = readInt64(b[56:65])
	if err != nil {
		panic(err)
	}
	z.mimeListPos = vb

	// checking for mainPage
	v, err = readInt32(b[64:69])
	if err != nil {
		panic(err)
	}
	z.mainPage = v

	// checking for layoutPage
	v, err = readInt32(b[68:73])
	if err != nil {
		panic(err)
	}
	z.layoutPage = v

	// Mime type list
	z.f.Seek(int64(z.mimeListPos), 0)

	eos := make([]byte, 1)
	utf8.EncodeRune(eos, '\x00')
	for {
		// read a chunk
		n, err := z.f.Read(b)
		if err != nil && err != io.EOF {
			panic(err)
		}
		if n == 0 {
			break
		}

		a := bytes.Split(b, eos)
		fmt.Println(len(a))
		for m := range a {
			fmt.Println(string(m))
		}
		break

	}

	return err
}

func readInt32(b []byte) (v uint32, err error) {
	buf := bytes.NewBuffer(b)
	err = binary.Read(buf, binary.LittleEndian, &v)
	if err != nil {
		return v, err
	}
	return v, nil
}

func readInt64(b []byte) (v uint64, err error) {
	buf := bytes.NewBuffer(b)
	err = binary.Read(buf, binary.LittleEndian, &v)
	if err != nil {
		return v, err
	}
	return v, nil
}

func (z *ZimReader) FindByName(string) error {
	return nil
}
