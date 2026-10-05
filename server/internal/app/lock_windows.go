package app

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// OS locks are released after a crash. Do not delete lock files while other processes may use them.
func lockConnection(path string) (func(), error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	o := &windows.Overlapped{}
	err = windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, o)
	if err != nil {
		f.Close()
		if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
			return nil, errConnectionBusy
		}
		return nil, err
	}
	return func() { f.Close() }, nil
}
