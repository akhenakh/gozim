package zim

import (
	"fmt"
	"io"

	"github.com/klauspost/compress/zstd"
)

type ZstdReader struct {
	*zstd.Decoder
}

func NewZstdReader(r io.Reader) (*ZstdReader, error) {
	dec, err := zstd.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("can't read from zstd %w", err)
	}
	return &ZstdReader{dec}, nil
}

func (zr *ZstdReader) Close() error {
	zr.Decoder.Close()

	return nil
}
