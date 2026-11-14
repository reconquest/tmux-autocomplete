package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	reportFileMode    = 0o600
	stackBufferSize   = 1024
	stackBufferGrowth = 2
)

type panicReport struct {
	//nolint:tagliatelle // Existing panic-report JSON schema uses snake_case.
	CursorX int `json:"cursor_x,omitempty"`
	//nolint:tagliatelle // Existing panic-report JSON schema uses snake_case.
	CursorY int      `json:"cursor_y,omitempty"`
	Pane    *Pane    `json:"pane,omitempty"`
	Lines   []string `json:"lines,omitempty"`
}

func (report *panicReport) write(reason any) error {
	encodedReport, _ := json.MarshalIndent(report, " ", "  ")

	filename := filepath.Join(
		os.TempDir(),
		"tmux-autocomplete_panic_"+time.Now().Format(time.RFC3339Nano)+".log",
	)
	data := fmt.Appendf(nil,
		"bug report: tmux-autocomplete %s %s\n\n%s\n%s\n%s\n",
		release, version, reason, getStacktrace(), encodedReport,
	)

	err := os.WriteFile(filename, data, reportFileMode)
	if err != nil {
		log.Printf("unable to write bug report into file: %q", filename)
		log.Println(string(data))

		return fmt.Errorf("write bug report: %w", err)
	}

	_, err = fmt.Fprintf(
		os.Stderr,
		"The program exited unexpectedly, it means that you've "+
			"encountered a bug,\n"+
			"but we have collected the bug report.\n\n"+
			"Make sure the file doesn't have any sensitive information: %s\n"+
			"Please help us to solve the bug by sending this report to "+
			"we@reconquest.io\n",
		filename,
	)
	if err != nil {
		return fmt.Errorf("write bug report notice: %w", err)
	}

	return nil
}

func getStacktrace() []byte {
	//nolint:makezero // runtime.Stack writes into the slice's full length.
	buffer := make([]byte, stackBufferSize)
	for {
		stack := runtime.Stack(buffer, true)
		if stack < len(buffer) {
			return buffer[:stack]
		}

		//nolint:makezero // runtime.Stack needs writable length, not capacity.
		buffer = make([]byte, stackBufferGrowth*len(buffer))
	}
}
