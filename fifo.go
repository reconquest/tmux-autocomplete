package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func mkfifo() (string, error) {
	name := filepath.Join(
		os.TempDir(),
		fmt.Sprintf("tmux-autocomplete_%d", time.Now().UnixNano()),
	)

	return name, nil
}
