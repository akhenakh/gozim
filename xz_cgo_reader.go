package zim

import (
	"io"

	xz "github.com/remyoudompheng/go-liblzma"
)

type XZReader struct {
	*xz.Decompressor
}

func NewXZReader(r io.Reader) (*XZReader, error) {
	dec, err := xz.NewReader(r)
	if err != nil {
		return nil, err
	}
	return &XZReader{dec}, nil
}

func (zr *XZReader) Close() error {
	return zr.Decompressor.Close()
}
