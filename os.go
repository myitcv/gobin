package main

import (
	"os"
	"runtime"
)

// os.UserHomeDir() is implemented in
// https://go-review.googlesource.com/c/go/+/139418 so implement here for now
func userHomeDir() string {
	if runtime.GOOS == "windows" {
		return os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
	}
	if runtime.GOOS == "plan9" {
		return os.Getenv("home")
	}
	if runtime.GOOS == "nacl" {
		return "/"
	}
	return os.Getenv("HOME")
}
