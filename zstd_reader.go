package zim

import (
	"io"

	"github.com/klauspost/compress/zstd"
)

type ZstdReader struct {
	*zstd.Decoder
}

func NewZstdReader(r io.Reader) (*ZstdReader, error) {
	dec, err := zstd.NewReader(r)
	if err != nil {
		return nil, err
	}
	return &ZstdReader{dec}, nil
}

func (zr *ZstdReader) Close() error {
	zr.Close()
	return nil
}
