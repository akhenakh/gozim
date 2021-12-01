package zim

import (
	"bytes"
	"encoding/binary"
)

// read a little endian uint64
func readInt64(b []byte, err error) (v uint64, aerr error) {
	if err != nil {
		aerr = err

		return
	}
	buf := bytes.NewBuffer(b)
	aerr = binary.Read(buf, binary.LittleEndian, &v)
	return
}

// read a little endian uint32
func readInt32(b []byte, err error) (v uint32, aerr error) {
	if err != nil {
		aerr = err
		return
	}
	buf := bytes.NewBuffer(b)
	aerr = binary.Read(buf, binary.LittleEndian, &v)

	return
}

// read a little endian uint32
func readInt16(b []byte, err error) (v uint16, aerr error) {
	if err != nil {
		aerr = err

		return
	}
	buf := bytes.NewBuffer(b)
	aerr = binary.Read(buf, binary.LittleEndian, &v)

	return
}
