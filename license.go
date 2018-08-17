package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"strings"
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
				cmd := exec.Command("xdg-open", "http://dead.archi/")
				cmd.Run()
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

func ensureValidLicense() {
	publicKey, err := lk.PublicKeyFromB64String(licensePublicKey)
	if err != nil {
		fatalln(
			karma.Format(err, "BUG: unable to decode public license key"),
			2,
		)
	}

	path := getLicensePath()

	context := karma.Describe("license", path)

	licenseData, err := ioutil.ReadFile(path)
	if err != nil {
		fatalln(
			context.Format(err, "unable to read license file"),
			2,
		)
	}

	license, err := lk.LicenseFromB64String(string(licenseData))
	if err != nil {
		fatalln(
			context.Format(err, "unable to decode license file data"),
			2,
		)
	}

	ok, err := license.Verify(publicKey)
	if err != nil {
		fatalln(
			context.Format(err, "unable to verify license"),
			2,
		)
	}

	if !ok {
		fatalln(context.Format(nil, "invalid license"), 2)
	}
}
