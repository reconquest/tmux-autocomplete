package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

var report = struct {
	CursorX int `json:"cursor_x,omitempty"`
	CursorY int `json:"cursor_y,omitempty"`

	Pane  *Pane    `json:"pane,omitempty"`
	Lines []string `json:"lines,omitempty"`
}{}

func writeReport(reason interface{}) {
	if reason == nil {
		return
	}

	encodedReport, _ := json.MarshalIndent(report, " ", "  ")
	stacktrace := getStacktrace()

	filename := filepath.Join(
		os.TempDir(),
		"tmux-autocomplete_panic_"+time.Now().Format(time.RFC3339Nano)+".log",
	)

	data := []byte(
		fmt.Sprintf(
			"bug report: tmux-autocomplete %s %s\n\n%s\n%s\n%s\n",
			release,
			version,
			reason,
			stacktrace,
			encodedReport,
		),
	)

	err := ioutil.WriteFile(filename, data, 0644)
	if err != nil {
		log.Printf("unable to write bug report into file: %q", filename)
		log.Println(string(data))
		os.Exit(137)
	}

	fmt.Fprintf(os.Stderr,
		"The program exited unexpectedly, it means that you've encountered a bug,\n"+
			"but we already collected a bug report.\n\n"+
			"Please check that this file doesn't have any sensitive information: %s\n"+
			"and if you don't mind, please help us to solve the bug by "+
			"sending this report to we@reconquest.io\n",
		filename,
	)

	os.Exit(137)
}

func getStacktrace() []byte {
	buffer := make([]byte, 1024)
	for {
		stack := runtime.Stack(buffer, true)
		if stack < len(buffer) {
			return buffer[:stack]
		}
		buffer = make([]byte, 2*len(buffer))
	}
}
