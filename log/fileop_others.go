//go:build !linux || !amd64 || noattr
// +build !linux !amd64 noattr

package log

func UMask(mask int) int {
	return 0
}
