package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMkfifo(t *testing.T) {
	directory := t.TempDir()
	t.Setenv("TMPDIR", directory)

	name, err := mkfifo()
	require.NoError(t, err)
	require.Equal(t, directory, filepath.Dir(name))

	info, err := os.Lstat(name)
	require.NoError(t, err)
	require.NotZero(t, info.Mode()&os.ModeNamedPipe)
	require.Zero(t, info.Mode().Perm()&0o077)

	other, err := mkfifo()
	require.NoError(t, err)
	require.NotEqual(t, name, other)
	require.NoError(t, os.Remove(name))
	require.NoError(t, os.Remove(other))

	entries, err := os.ReadDir(directory)
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestMkfifoFailure(t *testing.T) {
	directory := t.TempDir()
	t.Setenv("TMPDIR", filepath.Join(directory, "missing"))

	name, err := mkfifo()
	require.ErrorContains(t, err, "create fifo")
	require.ErrorIs(t, err, os.ErrNotExist)
	require.Empty(t, name)

	entries, err := os.ReadDir(directory)
	require.NoError(t, err)
	require.Empty(t, entries)
}
