package main

import (
	"github.com/mattn/go-colorable"
	"io"
	"os"
	"os/exec"
)

func clearConsole() {
	cmd := exec.Command("cmd", "/c", "cls")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func getOutput() io.Writer {
	return colorable.NewColorableStdout()
}
