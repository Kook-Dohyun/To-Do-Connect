//go:build !windows

package app

import "errors"

func nativeProtect([]byte) ([]byte, error) {
	return nil, errors.New("native DPAPI encryption is available only on Windows")
}
func nativeUnprotect([]byte) ([]byte, error) {
	return nil, errors.New("this is not a To Do Connect portable credential file; Windows DPAPI files cannot be opened on this OS")
}
