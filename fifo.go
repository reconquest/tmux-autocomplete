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

	fmt.Fprintf(os.Stderr, "XXXXXX fifo.go:16 syscall mkfifo\n")
	err := syscall.Mkfifo(name, 0666)
	if err != nil {
		return "", err
	}

	fmt.Fprintf(os.Stderr, "XXXXXX fifo.go:22 openfile mode\n")

	file, err := os.OpenFile(name, os.O_CREATE, os.ModeNamedPipe)
	if err != nil {
		fmt.Fprintf(os.Stderr, "XXXXXX fifo.go:26 err: %s\n", err)
		return "", err
	}

	fmt.Fprintf(os.Stderr, "XXXXXX fifo.go:29 file close\n")
	file.Close()

	fmt.Fprintf(os.Stderr, "XXXXXX fifo.go:32 return \n")
	return name, nil
}
