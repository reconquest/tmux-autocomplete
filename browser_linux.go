package main

import (
	"os/exec"
	"syscall"
)

func openBrowser(url string) {
	cmd := exec.Command("xdg-open", url)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	cmd.Start()
}
