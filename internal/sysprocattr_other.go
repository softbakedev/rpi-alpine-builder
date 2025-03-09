//go:build !windows
// +build !windows

package internal

import "syscall"

// For all non-Windows platforms, return nil or a default struct
func getNoWindowSysProcAttr() *syscall.SysProcAttr {
	return nil
}
