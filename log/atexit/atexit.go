//go:build !linux

package atexit

func Register(callback func()) {
}
