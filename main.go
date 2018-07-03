package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/docopt/docopt-go"
	"github.com/mgutz/ansi"
	"github.com/nsf/termbox-go"
	"github.com/reconquest/executil-go"
	"github.com/reconquest/karma-go"
)

var version = "2.0"

var usage = `tmux-autocomplete - provides autocomplete interface for pane contents.

Usage:
  tmux-autocomplete -h | --help
  tmux-autocomplete [options]
  tmux-autocomplete [options] -W <pane> <cursor-x> <cursor-y>

Options:
  -r --regexp <regexp>  Identifier regexp to match.
                         [default: [!-~]+]
  -n --no-prefix        Don't use identifier under cursor as prefix.
  -l --log <log>        Specify log file [default: /dev/stderr]
  -e --exec <program>   Exec specified program and pass specified candidate as argument.
  --theme <name>        Name of theme to use. [default: light]
  --theme-path <dir>    Path to directories with themes. Default:
                         * ` + defaultSystemThemePath + `
                         * ` + defaultUserThemePath + `
                         You can specify multiple directories using : separator.
  -v --version          Print version.
  -h --help             Show this help.
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

	logfile, err := os.OpenFile(
		args["--log"].(string),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0600,
	)
	if err != nil {
		panic(err)
	}

	log.SetOutput(logfile)

	themePath, ok := args["--theme-path"].(string)
	if !ok {
		themePath = defaultThemePath
	}

	theme, err := LoadTheme(themePath, args["--theme"].(string))
	if err != nil {
		log.Fatalln(
			karma.
				Describe("path", themePath).
				Describe("theme", args["--theme"].(string)).
				Format(err, "unable to load theme"),
		)
	}

	defer func() {
		tears := recover()
		if tears != nil {
			log.Printf("%s\n%s", tears, getStack(3))
		}
	}()

	tmux := &Tmux{}

	if !args["-W"].(bool) {
		err := start(args, tmux)
		if err != nil {
			log.Fatalln(err)
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

	var identifier *Identifier
	if withPrefix {
		identifier, err = getIdentifierToComplete(args, lines, x, y)
		if err != nil {
			log.Fatalln(err)
		}

		if identifier == nil {
			return
		}
	}

	moveCursor(cursorX, cursorY)

	candidates, err := getCompletionCandidates(args, lines, pane, identifier)
	if err != nil {
		log.Fatalln(err)
	}

	if identifier == nil {
		identifier = candidates[len(candidates)-1].Identifier
	}

	selectDefaultCandidate(candidates, identifier.X, identifier.Y)

	if len(getUniqueCandidates(candidates)) == 1 {
		useCurrentCandidate(tmux, pane, identifier, candidates, program)

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
			switch ev.Key {
			case termbox.KeyArrowUp:
				selectNextCandidate(candidates, 0, -1)

			case termbox.KeyArrowDown:
				selectNextCandidate(candidates, 0, 1)

			case termbox.KeyArrowLeft:
				selectNextCandidate(candidates, -1, 0)

			case termbox.KeyArrowRight:
				selectNextCandidate(candidates, 1, 0)

			case termbox.KeyEnter:
				useCurrentCandidate(tmux, pane, identifier, candidates, program)

				return

			case termbox.KeyCtrlC:
				return
			}

		case termbox.EventError:
			log.Fatalln(ev.Err)
		}
	}
}

func start(args map[string]interface{}, tmux *Tmux) error {
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
		return err
	}

	cmd := []string{
		os.Args[0],
		"--log", args["--log"].(string),
		"--regexp", fmt.Sprintf("%q", args["--regexp"].(string)),
		"--theme", args["--theme"].(string),
		pane,
		cursorX,
		cursorY,
		"-W",
	}

	if program, ok := args["--exec"].(string); ok {
		cmd = append(cmd, "--exec", program)
	}

	if args["--no-prefix"].(bool) {
		cmd = append(cmd, "--no-prefix")
	}

	return tmux.NewWindow(cmd...)
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
	identifier *Identifier,
	candidates []*Candidate,
	program string,
) {
	selected := getSelectedCandidate(candidates)
	if selected == nil {
		return
	}

	var text string
	if program != "" {
		// if we want to run program then we don't need to remove existing
		// identifier prefix
		text = selected.Value
	} else {
		text = string([]rune(selected.Value)[identifier.Length():])
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
