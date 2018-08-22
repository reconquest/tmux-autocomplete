package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

func mkfifo() (string, error) {
	name := filepath.Join(
		os.TempDir(),
		fmt.Sprintf("tmux-autocomplete_%d", time.Now().UnixNano()),
	)

	err := syscall.Mkfifo(name, 0666)
	if err != nil {
		return "", err
	}

	file, err := os.OpenFile(name, os.O_CREATE, os.ModeNamedPipe)
	if err != nil {
		return "", err
	}

	file.Close()

	return name, nil
}
