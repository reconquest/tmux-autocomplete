package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/docopt/docopt-go"
	"github.com/mattn/go-isatty"
	"github.com/mgutz/ansi"
	"github.com/nsf/termbox-go"
	"github.com/reconquest/executil-go"
	"github.com/reconquest/karma-go"
)

var (
	release = "dev"
	version = "dev-0-000000"
)

const (
	defaultRegexpCursor    = `[!-~]+`
	defaultRegexpCandidate = `[^<>]+`
)

var usage = `tmux-autocomplete - provides autocomplete interface for pane contents.

Usage:
  tmux-autocomplete -h | --help
  tmux-autocomplete [options]
  tmux-autocomplete [options] -W <pane> <cursor-x> <cursor-y>

Options:
  --license                       Show license information.
  -c --regexp-cursor <regexp>     Identifier regexp to match.
                                   [default: ` + defaultRegexpCursor + `]
  -r --regexp-candidate <regexp>  Candidate regexp to match.
                                   [default: ` + defaultRegexpCandidate + `]
  -n --no-prefix                  Don't use identifier under cursor as prefix.
  -e --exec <program>             Exec specified program and pass specified candidate as argument.
  --theme <name>                  Name of theme to use. [default: light]
  --theme-path <dir>              Path to directories with themes. Default:
                                   * ` + defaultSystemThemePath + `
                                   * ` + defaultUserThemePath + `
                                   You can specify multiple directories using : separator.
  -v --version                    Print version.
  -h --help                       Show this help.
`

func main() {
	args, err := docopt.Parse(
		usage,
		nil,
		true,
		"tmux-autocomplete "+version,
		false,
	)
	if err != nil {
		panic(err)
	}

	if args["--license"].(bool) {
		handleLicenseInfo()
	}

	if isLicenseExists() {
		ensureValidLicense()
	}

	themePath, ok := args["--theme-path"].(string)
	if !ok {
		themePath = defaultThemePath
	}

	theme, err := LoadTheme(themePath, args["--theme"].(string))
	if err != nil {
		fatalln(
			karma.
				Describe("path", themePath).
				Describe("theme", args["--theme"].(string)).
				Format(err, "unable to load theme"),
			2,
		)
	}

	tmux := &Tmux{}

	if !args["-W"].(bool) {
		if isatty.IsTerminal(os.Stdin.Fd()) {
			printIntroductionMessage()
			os.Exit(1)
		}

		err := start(args, themePath, tmux)
		if err != nil {
			fatalln(
				karma.Format(
					err,
					"unable to start tmux-autocomplete",
				),
				3,
			)
		}

		return
	}

	var (
		cursorX int
		cursorY int

		program, _ = args["--exec"].(string)
		withPrefix = !args["--no-prefix"].(bool)
	)

	_, err = fmt.Sscan(args["<cursor-x>"].(string), &cursorX)
	if err != nil {
		log.Fatalln(err)
	}

	_, err = fmt.Sscan(args["<cursor-y>"].(string), &cursorY)
	if err != nil {
		log.Fatalln(err)
	}

	pane, err := CapturePane(tmux, args["<pane>"].(string), "-eJ")
	if err != nil {
		log.Fatalln(err)
	}

	lines := pane.GetPrintable()

	x, y := pane.GetBufferXY(lines, cursorX, cursorY)

	report.CursorX = cursorX
	report.CursorY = cursorY
	report.Pane = pane
	report.Lines = lines

	defer func() {
		writeReport(recover())
	}()

	var identifier *Identifier
	if withPrefix {
		identifier, err = getIdentifierToComplete(args["--regexp-cursor"].(string), lines, x, y)
		if err != nil {
			log.Fatalln(err)
		}

		if identifier == nil {
			return
		}
	}

	moveCursor(cursorX, cursorY)

	candidates, err := getCompletionCandidates(
		args["--regexp-candidate"].(string),
		lines,
		identifier,
	)
	if err != nil {
		log.Fatalln(err)
	}

	if len(candidates) == 0 {
		return
	}

	if identifier == nil {
		identifier = candidates[len(candidates)-1].Identifier
	}

	selectDefaultCandidate(candidates, identifier.X, identifier.Y)

	if len(getUniqueCandidates(candidates)) == 1 {
		useCurrentCandidate(
			tmux,
			pane,
			theme,
			identifier,
			candidates,
			program,
			withPrefix,
		)

		return
	}

	err = termbox.Init()
	if err != nil {
		log.Fatalln(err)
	}

	for {
		renderPane(pane, theme)
		renderIdentifier(tmux, lines, pane, theme, identifier)
		renderCandidates(tmux, lines, pane, theme, candidates)

		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			switch {
			case ev.Ch == 'k':
				fallthrough
			case ev.Key == termbox.KeyArrowUp:
				selectNextCandidate(candidates, 0, -1)

			case ev.Ch == 'j':
				fallthrough
			case ev.Key == termbox.KeyArrowDown:
				selectNextCandidate(candidates, 0, 1)

			case ev.Ch == 'h':
				fallthrough
			case ev.Key == termbox.KeyArrowLeft:
				selectNextCandidate(candidates, -1, 0)

			case ev.Ch == 'l':
				fallthrough
			case ev.Key == termbox.KeyArrowRight:
				selectNextCandidate(candidates, 1, 0)

			case ev.Key == termbox.KeyEnter:
				useCurrentCandidate(
					tmux,
					pane,
					theme,
					identifier,
					candidates,
					program,
					withPrefix,
				)

				return

			case ev.Key == termbox.KeyCtrlC:
				return
			}

		case termbox.EventError:
			log.Fatalln(ev.Err)
		}
	}
}

func fatalln(err interface{}, exitcode int) {
	fmt.Println(err)
	log.Println(err)
	os.Exit(exitcode)
}

func start(args map[string]interface{}, themePath string, tmux *Tmux) error {
	var (
		pane    string
		cursorX string
		cursorY string
	)

	err := tmux.Eval(
		map[string]interface{}{
			"pane_id":  &pane,
			"cursor_x": &cursorX,
			"cursor_y": &cursorY,
		},
	)
	if err != nil {
		return karma.Format(
			err,
			"unable to get current pane/cursor",
		)
	}

	logsPipe, err := mkfifo()
	if err != nil {
		return karma.Format(
			err,
			"unable to make fifo",
		)
	}

	defer os.Remove(logsPipe)

	cmd := []string{os.Args[0]}

	for flag, value := range args {
		switch flag {
		case "--theme-path":
			cmd = append(cmd, flag, fmt.Sprintf("%q", themePath))
		case "--regexp":
			cmd = append(cmd, flag, fmt.Sprintf("%q", value))
		default:
			switch typed := value.(type) {
			case string:
				cmd = append(cmd, flag, fmt.Sprintf("%q", typed))
			case bool:
				if typed {
					cmd = append(cmd, flag)
				}
			case nil:
				//
			default:
				panic(
					fmt.Sprintf(
						"unexpected type of flag %s: %#v (%T)",
						flag, value, value,
					),
				)
			}
		}
	}

	cmd = append(cmd, pane, cursorX, cursorY, "-W", "2>"+logsPipe)

	err = tmux.NewWindow(cmd...)
	if err != nil {
		return karma.
			Format(
				err,
				"unable to create new tmux window",
			)
	}

	file, err := os.Open(logsPipe)
	if err != nil {
		return karma.Format(
			err,
			"unable to open logs file",
		)
	}

	// Fd() does side effect - set nonblocking mode
	// set nonblocking mode
	// https://github.com/golang/go/issues/24164#issuecomment-370097226
	file.Fd()

	logs, err := ioutil.ReadAll(file)
	if err != nil {
		return karma.Format(
			err,
			"unable to read logs fifo",
		)
	}

	if len(logs) > 0 {
		return errors.New(string(logs))
	}

	return nil
}

func renderPane(pane *Pane, theme *Theme) {
	moveCursor(0, 0)

	fmt.Print(ansi.ColorCode(theme.Fog.Text))

	text := reEscapeSequence.ReplaceAllStringFunc(
		pane.String(),
		func(sequence string) string {
			return decolorize(sequence, theme)
		},
	)

	fmt.Print(text)
}

func decolorize(sequence string, theme *Theme) string {
	var (
		matches = reEscapeSequence.FindStringSubmatch(sequence)
		code    = matches[1]
	)

	switch code {
	// 49 means reset background color to default,
	// we remove background color and set dim color for foreground
	case "49":
		return ansi.DefaultBG + ansi.ColorCode(theme.Fog.Text)

	// 39 means reset foreground color to default,
	// 0 means reset all style to default,
	// we reset style and set dim color for foreground
	case "0", "39":
		return ansi.Reset + ansi.ColorCode(theme.Fog.Text)

	// 1 means bold,
	// we remove bold
	case "1":
		return ""

	// 7 means reverse,
	// we set dim background color
	case "7":
		return ansi.ColorCode(theme.Fog.Background)

	default:
		// all single-number codes,
		// we remove
		if len(code) == 1 {
			return ""
		}

		// all foreground theme,
		// we replace with dim foreground color
		if code[0] == '3' || code[0] == '9' {
			return ansi.ColorCode(theme.Fog.Text)
		}

		// all background theme,
		// we replace with dim background color
		if code[0] == '4' || code[0] == '1' {
			return ansi.ColorCode(theme.Fog.Background)
		}
	}

	return ansi.Reset
}

func renderIdentifier(
	tmux *Tmux,
	lines []string,
	pane *Pane,
	theme *Theme,
	identifier *Identifier,
) {
	moveCursor(pane.GetScreenXY(lines, identifier.X, identifier.Y))

	fmt.Print(ansi.ColorFunc(theme.Identifier)(identifier.Value))
}

func renderCandidates(
	tmux *Tmux,
	lines []string,
	pane *Pane,
	theme *Theme,
	candidates []*Candidate,
) {
	for _, candidate := range candidates {
		x := candidate.X
		y := candidate.Y

		moveCursor(pane.GetScreenXY(lines, x, y))

		color := theme.Candidate.Normal

		if candidate.Selected {
			color = theme.Candidate.Selected
		}

		fmt.Print(ansi.ColorFunc(color)(candidate.Value))
	}
}

func useCurrentCandidate(
	tmux *Tmux,
	pane *Pane,
	theme *Theme,
	identifier *Identifier,
	candidates []*Candidate,
	program string,
	withPrefix bool,
) {
	selected := getSelectedCandidate(candidates)
	if selected == nil {
		return
	}

	if !isLicenseExists() {
		err := nagLicense(tmux, pane, theme)
		if err != nil {
			log.Fatalln(err)
		}
	}

	var text string
	if program != "" {
		// if we want to run program then we don't need to remove existing
		// identifier prefix
		text = selected.Value
	} else {
		text = selected.Value
		if withPrefix {
			text = string([]rune(text)[identifier.Length():])
		}
	}

	if program != "" {
		_, _, err := executil.Run(exec.Command(program, text))
		if err != nil {
			log.Fatalln(err)
		}
	} else {
		err := tmux.Paste(text, "-t", pane.ID)
		if err != nil {
			log.Fatalln(err)
		}
	}
}

func getStack(skip int) string {
	buffer := make([]byte, 1024)
	for {
		written := runtime.Stack(buffer, true)
		if written < len(buffer) {
			// call stack contains of goroutine number and set of calls
			//   goroutine NN [running]:
			//   github.com/user/project.(*Type).MethodFoo()
			//        path/to/src.go:line
			//   github.com/user/project.MethodBar()
			//        path/to/src.go:line
			// so if we need to skip 2 calls than we must split stack on
			// following parts:
			//   2(call)+2(call path)+1(goroutine header) + 1(callstack)
			// and extract first and last parts of resulting slice
			stack := strings.SplitN(string(buffer[:written]), "\n", skip*2+2)
			return stack[0] + "\n" + stack[skip*2+1]
		}

		buffer = make([]byte, 2*len(buffer))
	}
}
