//go:build windows
// +build windows

package internal

import "syscall"

func getNoWindowSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		// CREATE_NO_WINDOW = 0x08000000
		CreationFlags: 0x08000000,
	}
}
