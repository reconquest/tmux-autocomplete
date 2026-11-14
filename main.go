package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/docopt/docopt-go"
	"github.com/mattn/go-isatty"
	"github.com/reconquest/karma-go"
)

//nolint:gochecknoglobals // Release metadata is set by build-time -X flags.
var (
	release = "dev"
	version = "dev-0-000000"
)

const (
	defaultRegexpCursor    = `[!-~]+`
	defaultRegexpCandidate = `[^<> ]+`
	exitTheme              = 2
	exitStart              = 3
	exitPanic              = 137
	debugFileMode          = 0o600
)

//nolint:gochecknoglobals // Completion shares this process-wide debug logger.
var debug = log.New(io.Discard, "", 0)

func cliUsage() string {
	return `tmux-autocomplete - provides autocomplete interface ` +
		`for pane contents.

Usage:
  tmux-autocomplete -h | --help
  tmux-autocomplete [options]
  tmux-autocomplete [options] -W <pane> <cursor-x> <cursor-y>

Options:
  -c --regexp-cursor <regexp>     Identifier regexp to match.
                                   [default: ` + defaultRegexpCursor + `]
  -r --regexp-candidate <regexp>  Candidate regexp to match.
                                   [default: ` + defaultRegexpCandidate + `]
  -n --no-prefix                  Don't use identifier under cursor as prefix.
  -e --exec <program>             Exec specified program and pass specified ` +
		`candidate as argument.
  --theme <name>                  Name of theme to use. [default: light]
  --theme-path <dir>              Path to directories with themes. Default:
                                   * ` + defaultSystemThemePath + `
                                   * ` + defaultUserThemePath + `
                                   You can specify multiple directories ` +
		`using : separator.
  --debug <file>                  Print debug messages into specified file.
  -v --version                    Print version.
  -h --help                       Show this help.
`
}

type command struct {
	exitCode int
	report   *panicReport
}

func main() {
	command := &command{}
	command.run()
	os.Exit(command.exitCode)
}

func (command *command) run() {
	defer func() {
		reason := recover()
		if reason != nil {
			command.recoverPanic(reason)
		}
	}()

	options, err := docopt.ParseArgs(
		cliUsage(),
		nil,
		"tmux-autocomplete "+version,
	)
	if err != nil {
		panic(err)
	}

	args, err := bindArguments(options)
	if err != nil {
		panic(err)
	}

	if options["--debug"] != nil {
		out, openErr := openDebug(args.ValueDebug)
		if openErr != nil {
			panic(openErr)
		}
		defer closeLogged(out)
	}

	command.dispatch(&args, options)
}

func (command *command) dispatch(args *arguments, options docopt.Opts) {
	theme, err := LoadTheme(args.ValueThemePath, args.ValueTheme)
	if err != nil {
		command.fail(karma.Describe("path", args.ValueThemePath).
			Describe("theme", args.ValueTheme).
			Format(err, "unable to load theme"), exitTheme)

		return
	}

	tmux := &Tmux{}
	if !args.FlagWorker {
		command.launch(options, args.ValueThemePath, tmux)

		return
	}

	command.work(args, tmux, theme)
}

func (command *command) recoverPanic(reason any) {
	if command.report == nil {
		panic(reason)
	}

	err := command.report.write(reason)
	if err != nil {
		log.Print(err)
	}

	command.exitCode = exitPanic
}

func (command *command) launch(
	options docopt.Opts, themePath string, tmux *Tmux,
) {
	if isatty.IsTerminal(os.Stdin.Fd()) {
		err := printIntroductionMessage()
		if err != nil {
			log.Print(err)
		}

		command.exitCode = 1

		return
	}

	err := start(options, themePath, tmux)
	if err != nil {
		command.fail(
			karma.Format(err, "unable to start tmux-autocomplete"), exitStart,
		)
	}
}

func (command *command) work(args *arguments, tmux *Tmux, theme *Theme) {
	worker, err := captureWorker(args, tmux)
	if err != nil {
		log.Print(err)

		command.exitCode = 1

		return
	}

	command.report = &panicReport{
		CursorX: worker.cursorX,
		CursorY: worker.cursorY,
		Pane:    worker.pane,
		Lines:   worker.lines,
	}

	err = worker.complete(theme)
	if err != nil {
		log.Print(err)

		command.exitCode = 1
	}
}

func (command *command) fail(reason any, code int) {
	_, err := fmt.Fprintln(os.Stdout, reason)
	if err != nil {
		log.Printf("unable to write error: %v", err)
	}

	log.Println(reason)

	command.exitCode = code
}

func openDebug(path string) (*os.File, error) {
	// #nosec G304 -- The --debug option selects a local log path.
	out, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		debugFileMode,
	)
	if err != nil {
		return nil, fmt.Errorf("open debug log: %w", err)
	}

	debug = log.New(out, "", log.Lshortfile|log.Ltime)

	return out, nil
}

func closeLogged(file *os.File) {
	err := file.Close()
	if err != nil {
		log.Printf("unable to close %q: %v", file.Name(), err)
	}
}
