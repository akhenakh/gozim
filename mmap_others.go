// +build !windows

package zim

import (
	"syscall"
)

func NewMmap(fd int, offset int64, length int) (data []byte, err error) {
	return syscall.Mmap(fd, offset, length, syscall.PROT_READ, syscall.MAP_PRIVATE)
}
