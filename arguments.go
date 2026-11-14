package main

import (
	"github.com/docopt/docopt-go"
	"github.com/reconquest/karma-go"
)

type arguments struct {
	ValueCursorRegexp    string `docopt:"--regexp-cursor"`
	ValueCandidateRegexp string `docopt:"--regexp-candidate"`
	ValueProgram         string `docopt:"--exec"`
	ValueTheme           string `docopt:"--theme"`
	ValueThemePath       string `docopt:"--theme-path"`
	ValueDebug           string `docopt:"--debug"`
	ValuePane            string `docopt:"<pane>"`
	ValueCursorX         string `docopt:"<cursor-x>"`
	ValueCursorY         string `docopt:"<cursor-y>"`
	FlagNoPrefix         bool   `docopt:"--no-prefix"`
	FlagWorker           bool   `docopt:"-W"`
}

func bindArguments(options docopt.Opts) (arguments, error) {
	var args arguments

	err := options.Bind(&args)
	if err != nil {
		return arguments{}, karma.Format(err, "bind command arguments")
	}

	if options["--theme-path"] == nil {
		args.ValueThemePath = defaultThemePath
	}

	return args, nil
}
