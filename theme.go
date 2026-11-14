package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/kovetskiy/ko"
	"github.com/reconquest/karma-go"

	yaml "gopkg.in/coryb/yaml.v2"
)

// Theme defines the required identifier and candidate styles and optional fog.
type Theme struct {
	Identifier string         `required:"true"`
	Candidate  themeCandidate `required:"true"`
	Fog        themeFog
}

type themeCandidate = struct {
	Normal   string `required:"true"`
	Selected string `required:"true"`
}

type themeFog = struct {
	Text       string `required:"true"`
	Background string `required:"true"`
}

const (
	defaultUserThemePath = `~/.config/tmux-autocomplete/themes/`
	defaultThemePath     = defaultSystemThemePath + `:` + defaultUserThemePath
)

// LoadTheme loads the first matching theme from colon-separated directories.
// Missing files are skipped; invalid themes and empty directories are errors.
func LoadTheme(dirs, name string) (*Theme, error) {
	var theme Theme

	for _, dir := range strings.Split(dirs, ":") {
		path, err := themeFilePath(dir, name)
		if err != nil {
			return nil, err
		}

		err = ko.Load(path, &theme, yaml.Unmarshal)
		if err == nil {
			return &theme, nil
		}

		if os.IsNotExist(err) {
			continue
		}

		return nil, karma.Describe("path", path).Format(err, "read theme file")
	}

	return nil, errors.New("no such theme found")
}

func themeFilePath(dir, name string) (string, error) {
	if dir == "" {
		return "", errors.New("empty directory with themes specified")
	}

	if strings.HasPrefix(dir, "~/") {
		dir = os.Getenv("HOME") + dir[1:]
	}

	return filepath.Join(dir, name+".theme"), nil
}
