//go:build linux && amd64 && !noattr
// +build linux,amd64,!noattr

package log

import "syscall"

func UMask(mask int) int {
	oldMask := syscall.Umask(mask)
	return oldMask
}
