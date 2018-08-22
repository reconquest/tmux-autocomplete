package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/hyperboloide/lk"
	"github.com/mgutz/ansi"
	"github.com/nsf/termbox-go"
	"github.com/reconquest/karma-go"
)

var licensePublicKey = ""

func init() {
	rand.Seed(time.Now().UnixNano())
}

func nagLicense(tmux *Tmux, pane *Pane, theme *Theme) {
	if rand.Intn(10) != 0 {
		return
	}

	moveCursor(0, 0)

	nag := `

   __________________________________________________________________
  /\                                                                 \
  \_| Hello! Thank you for trying out tmux-autocomplete.             |
    |                                                                |
    | This is an unregistered evaluation version of the program, and |
    | although the trial is untimed, a license must be purchased for |
    | continued use.                                                 |
    |                                                                |
    | Would you like to purchase a license now?                      |
    |   _____________________________________________________________|_
     \_/_______________________________________________________________/
`

	nagLines := strings.Split(nag, "\n")

	for i, nagLine := range nagLines {
		suffix := ""
		diff := pane.Width - len(nagLine)
		if diff > 0 {
			suffix = strings.Repeat(" ", diff)
		}
		nagLines[i] = nagLine + suffix
	}

	if len(nagLines) < pane.Height {
		for i := 0; i < pane.Height-len(nagLines); i++ {
			nagLines = append(nagLines, strings.Repeat(" ", pane.Width))
		}
	}

	nag = strings.Join(nagLines, "\n")

	fmt.Print(ansi.ColorCode("black:231"))
	fmt.Print(nag)
	fmt.Println()

	for {
		ev := termbox.PollEvent()
		switch ev.Type {
		case termbox.EventKey:
			switch ev.Key {
			case termbox.KeyEnter:
				cmd := exec.Command("xdg-open", "https://tmux.reconquest.io/")
				cmd.SysProcAttr = &syscall.SysProcAttr{
					Setpgid: true,
				}

				cmd.Start()
				return

			case termbox.KeyCtrlC:
				return
			}
		}
	}
}

func getLicensePath() string {
	return os.ExpandEnv(
		"$HOME/.config/tmux-autocomplete/" + release + ".license",
	)
}

func isLicenseExists() bool {
	_, err := os.Stat(getLicensePath())
	return !os.IsNotExist(err)
}

func getLicense() (*lk.License, error) {
	publicKey, err := lk.PublicKeyFromB32String(licensePublicKey)
	if err != nil {
		return nil, karma.Format(err, "BUG: unable to decode public license key")
	}

	path := getLicensePath()

	context := karma.Describe("license", path)

	licenseData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, context.Format(err, "unable to read license file")
	}

	license, err := lk.LicenseFromB32String(string(licenseData))
	if err != nil {
		return nil, context.Format(err, "unable to decode license file data")
	}

	ok, err := license.Verify(publicKey)
	if err != nil {
		return nil, context.Format(err, "unable to verify license")
	}

	if !ok {
		return nil, nil
	}

	return license, nil
}

func ensureValidLicense() {
	license, err := getLicense()
	if err != nil {
		fatalln(err, 2)
	}

	if license == nil {
		fatalln(
			karma.Format(
				nil,
				"invalid license: unable to verify using public key",
			),
			2,
		)
	}
}
