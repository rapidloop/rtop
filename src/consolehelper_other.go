// +build !windows

package main

import (
	"io"
	"os"
)

func clearConsole() {}

func getOutput() io.Writer {
	return os.Stdout
}
