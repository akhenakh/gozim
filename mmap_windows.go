package zim

import (
	"syscall"
	"unsafe"
)

func NewMmap(fd int, offset int64, length int) (data []byte, err error) {
	var i syscall.ByHandleFileInformation
	err = syscall.GetFileInformationByHandle(syscall.Handle(fd), &i)
	if err != nil {
		return nil, err
	}
	size := int64(i.FileSizeHigh)<<32 + int64(i.FileSizeLow)
	if int64(length) > size {
		length = int(size)
	}
	fmap, err := syscall.CreateFileMapping(syscall.Handle(fd), nil, syscall.PAGE_READONLY, 0, uint32(length), nil)
	if err != nil {
		return nil, err
	}
	defer syscall.CloseHandle(fmap)
	ptr, err := syscall.MapViewOfFile(fmap, syscall.FILE_MAP_READ, 0, 0, uintptr(length))
	if err != nil {
		return nil, err
	}
	return (*[1 << 30]byte)(unsafe.Pointer(ptr))[:length], nil
}
