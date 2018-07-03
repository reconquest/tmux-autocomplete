package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/kovetskiy/ko"
	"github.com/reconquest/karma-go"
)

type Theme struct {
	Identifier string `required:"true"`

	Candidate struct {
		Normal   string `required:"true"`
		Selected string `required:"true"`
	} `required:"true"`

	Fog struct {
		Text       string `required:"true"`
		Background string `required:"true"`
	}
}

var (
	defaultUserThemePath = `~/.config/tmux-autocomplete/themes/`
	defaultThemePath     = defaultSystemThemePath + `:` + defaultUserThemePath
)

func LoadTheme(dirs string, name string) (*Theme, error) {
	var theme Theme
	for _, dir := range strings.Split(dirs, ":") {
		if dir == "" {
			return nil, fmt.Errorf("empty directory with themes specified")
		}

		if strings.HasPrefix(dir, "~/") {
			// 1 because need to trim only ~ symbol, slash is required
			dir = os.Getenv("HOME") + dir[1:]
		}

		path := filepath.Join(dir, name+".theme")

		err := ko.Load(path, &theme, yaml.Unmarshal)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}

			return nil, karma.Format(
				err,
				"unable to read theme file: %s", path,
			)
		}

		return &theme, nil
	}

	return nil, errors.New("no such theme found")
}
