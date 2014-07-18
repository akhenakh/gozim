package zim

import (
	"bytes"
	"encoding/binary"
)

func readInt64(b []byte) (v uint64, err error) {
	buf := bytes.NewBuffer(b)
	err = binary.Read(buf, binary.LittleEndian, &v)
	if err != nil {
		return v, err
	}
	return v, nil
}

func readInt32(b []byte) (v uint32, err error) {
	buf := bytes.NewBuffer(b)
	err = binary.Read(buf, binary.LittleEndian, &v)
	if err != nil {
		return v, err
	}
	return v, nil
}

func readInt16(b []byte) (v uint16, err error) {
	buf := bytes.NewBuffer(b)
	err = binary.Read(buf, binary.LittleEndian, &v)
	if err != nil {
		return v, err
	}
	return v, nil
}
