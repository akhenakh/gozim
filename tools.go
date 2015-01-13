package zim

import (
	"bytes"
	"encoding/binary"
)

// read a little endian uint64 panic in case of error
func readInt64(b []byte) (v uint64) {
	buf := bytes.NewBuffer(b)
	err := binary.Read(buf, binary.LittleEndian, &v)
	if err != nil {
		panic(err)
	}
	return v
}

// read a little endian uint32 panic in case of error
func readInt32(b []byte) (v uint32) {
	buf := bytes.NewBuffer(b)
	err := binary.Read(buf, binary.LittleEndian, &v)
	if err != nil {
		panic(err)
	}
	return v
}

// read a little endian uint32 panic in case of error
func readInt16(b []byte) (v uint16) {
	buf := bytes.NewBuffer(b)
	err := binary.Read(buf, binary.LittleEndian, &v)
	if err != nil {
		panic(err)
	}
	return v
}
