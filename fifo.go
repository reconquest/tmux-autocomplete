package main

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func mkfifo() (string, error) {
	const permissions = 0o600

	var token [16]byte

	_, err := rand.Read(token[:])
	if err != nil {
		return "", fmt.Errorf("generate fifo name: %w", err)
	}

	name := filepath.Join(
		os.TempDir(),
		fmt.Sprintf("tmux-autocomplete_%x", token),
	)

	err = syscall.Mkfifo(name, permissions)
	if err != nil {
		return "", fmt.Errorf("create fifo %q: %w", name, err)
	}

	return name, nil
}
