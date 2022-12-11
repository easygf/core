//go:build !linux

package config

import (
	"os"
	"path/filepath"
)

var FsPath = filepath.Join(GetUserHomeDir(), ".config", "easygf", "config")

func GetUserHomeDir() string {
	homeDir, _ := os.UserHomeDir()
	return homeDir
}
