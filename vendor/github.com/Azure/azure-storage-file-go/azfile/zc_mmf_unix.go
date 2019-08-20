// +build linux darwin freebsd

package azfile

import (
	"os"

	"golang.org/x/sys/unix"
)

type mmf []byte

func newMMF(file *os.File, writable bool, offset int64, length int) (mmf, error) {
	prot, flags := unix.PROT_READ, unix.MAP_SHARED // Assume read-only
	if writable {
		prot, flags = unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED
	}
	addr, err := unix.Mmap(int(file.Fd()), offset, length, prot, flags)
	return mmf(addr), err
}

func (m *mmf) unmap() {
	err := unix.Munmap(*m)
	*m = nil
	if err != nil {
		sanityCheckFailed(err.Error())
	}
}
