package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadThemeShipped(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"dark", "light"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			theme, err := LoadTheme("share/themes", name)
			require.NoError(t, err)
			require.Equal(t, "default+ub:default", theme.Identifier)

			if name == "dark" {
				require.Equal(t, "green:default", theme.Candidate.Normal)
				require.Equal(t, "16+b:green", theme.Candidate.Selected)
				require.Equal(t, "236:default", theme.Fog.Text)
				require.Equal(t, "238:236", theme.Fog.Background)
			} else {
				require.Equal(t, "232:default", theme.Candidate.Normal)
				require.Equal(t, "230+b:232", theme.Candidate.Selected)
				require.Equal(t, "250:default", theme.Fog.Text)
				require.Equal(t, "250:default", theme.Fog.Background)
			}
		})
	}
}

func TestLoadThemeSearchOrder(t *testing.T) {
	t.Parallel()

	first := t.TempDir()
	second := t.TempDir()
	missing := t.TempDir()
	writeThemeFixture(
		t,
		first,
		"identifier: first\ncandidate:\n  normal: green\n  selected: red\n",
	)
	writeThemeFixture(
		t,
		second,
		"identifier: second\ncandidate:\n  normal: blue\n  selected: yellow\n",
	)

	for _, scenario := range []struct {
		name       string
		dirs       string
		identifier string
	}{
		{"explicit", first, "first"},
		{"first match", first + ":" + second, "first"},
		{"reversed", second + ":" + first, "second"},
		{"missing fallback", missing + ":" + second, "second"},
		{"unvisited empty root", first + ":", "first"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			theme, err := LoadTheme(scenario.dirs, "custom")
			require.NoError(t, err)
			require.Equal(t, scenario.identifier, theme.Identifier)
		})
	}

	require.Equal(t,
		defaultSystemThemePath+":"+defaultUserThemePath,
		defaultThemePath,
	)
}

func TestLoadThemeHomeExpansion(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeThemeFixture(
		t,
		home,
		"identifier: home\ncandidate:\n  normal: green\n  selected: red\n",
	)

	theme, err := LoadTheme("~/", "custom")
	require.NoError(t, err)
	require.Equal(t, "home", theme.Identifier)
}

func TestLoadThemeMissingAndEmptyRoots(t *testing.T) {
	t.Parallel()

	missing := t.TempDir()

	for _, scenario := range []struct {
		name    string
		dirs    string
		message string
	}{
		{"missing", missing, "no such theme found"},
		{"all missing", missing + ":" + t.TempDir(), "no such theme found"},
		{"empty", "", "empty directory with themes specified"},
		{
			"leading empty", ":" + missing,
			"empty directory with themes specified",
		},
		{
			"trailing empty", missing + ":",
			"empty directory with themes specified",
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			theme, err := LoadTheme(scenario.dirs, "custom")
			require.EqualError(t, err, scenario.message)
			require.Nil(t, theme)
		})
	}
}

func TestLoadThemeValidation(t *testing.T) {
	t.Parallel()

	const valid = "identifier: custom\ncandidate:\n" +
		"  normal: green\n  selected: red\n"

	fallback := t.TempDir()
	writeThemeFixture(t, fallback, valid)

	for _, scenario := range []struct {
		name    string
		content string
		field   string
	}{
		{
			"identifier",
			strings.ReplaceAll(valid, "identifier: custom\n", ""),
			"identifier",
		},
		{"candidate", "identifier: custom\n", "candidate.normal"},
		{
			"normal", strings.ReplaceAll(valid, "  normal: green\n", ""),
			"candidate.normal",
		},
		{
			"selected", strings.ReplaceAll(valid, "  selected: red\n", ""),
			"candidate.selected",
		},
		{"malformed", "identifier: [", ""},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			writeThemeFixture(t, dir, scenario.content)

			theme, err := LoadTheme(dir+":"+fallback, "custom")
			require.Error(t, err)
			require.Nil(t, theme)
			require.ErrorContains(t, err, filepath.Join(dir, "custom.theme"))

			if scenario.field != "" {
				require.ErrorContains(t, err, scenario.field)
			}
		})
	}

	theme, err := LoadTheme(fallback, "custom")
	require.NoError(t, err)
	require.Empty(t, theme.Fog.Text)
	require.Empty(t, theme.Fog.Background)
}

func writeThemeFixture(t *testing.T, dir, content string) {
	t.Helper()

	err := os.WriteFile(
		filepath.Join(dir, "custom.theme"),
		[]byte(content),
		0o600,
	)
	require.NoError(t, err)
}
