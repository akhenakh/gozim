// +build !cgo

package zim

import (
	"io"

	"github.com/ulikunitz/xz"
)

type XZReader struct {
	*xz.Reader
}

func NewXZReader(r io.Reader) (*XZReader, error) {
	dec, err := xz.NewReader(r)
	if err != nil {
		return nil, err
	}
	return &XZReader{dec}, nil
}

func (xr *XZReader) Close() error {
	return nil
}
