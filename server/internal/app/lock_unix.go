//go:build linux || darwin

package app

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func lockConnection(path string) (func(), error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		f.Close()
		if errors.Is(err, unix.EWOULDBLOCK) {
			return nil, errConnectionBusy
		}
		return nil, err
	}
	return func() { f.Close() }, nil
}
