package main

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/reconquest/karma-go"
)

func mkfifo() (string, error) {
	const permissions = 0o600

	var token [16]byte

	_, err := rand.Read(token[:])
	if err != nil {
		return "", karma.Format(err, "generate fifo name")
	}

	name := filepath.Join(
		os.TempDir(),
		fmt.Sprintf("tmux-autocomplete_%x", token),
	)

	err = syscall.Mkfifo(name, permissions)
	if err != nil {
		return "", karma.Describe("path", name).Format(err, "create fifo")
	}

	return name, nil
}
