package vfshash

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"

	"golang.org/x/tools/godoc/vfs/httpfs"
	"golang.org/x/tools/godoc/vfs/mapfs"

	"github.com/stretchr/testify/assert"
)

func TestAssetNames(t *testing.T) {
	fs := NewFileSystem(fs)
	names := NewAssetNames(fs)

	for _, name := range []string{
		// directories
		"/",
		"/example",

		// non-existant
		"/does.not.exist",

		// assets file
		"/.vfshash-assets.json",

		// non-clean non-existant paths
		"//does.not.exist",
		"/does//not.exist",
	} {
		assert.Equalf(t, cleanPath(name),
			names.Lookup(name),
			"%s not passed through as is", name)
	}

	for _, name := range []string{
		// relative path to file
		"test1",
		"./test1",
		"example/../test1",
		"example/test2",
		"example/./test2",

		// non-clean paths
		"//test1",
		"/example//test2",
	} {
		assert.Equalf(t, expectedHashes[cleanPath(name)],
			names.Lookup(name),
			"%s not transformed into hashed name", name)
	}

	for name, hashed := range expectedHashes {
		assert.Equalf(t, hashed, names.Lookup(name),
			"%s not transformed into hashed name", name)
		assert.Equalf(t, hashed, names.Lookup(hashed),
			"%s not passed through as is", hashed)
	}
}

func TestAssetNamesNonExistant(t *testing.T) {
	names := NewAssetNames(fs)

	assert.NotPanics(t, func() {
		assert.Equal(t, "/test1",
			names.Lookup("/test1"))
	})
}

func TestAssertNamesInvalidJSON(t *testing.T) {
	fs := httpfs.New(mapfs.New(map[string]string{
		path.Base(assetsJSONPath): "this is invalid json",
		"test1":                   "test1",
	}))
	names := NewAssetNames(fs)

	assert.Panics(t, func() {
		names.Lookup("/test1")
	})
}

func TestAssetNamesOpen(t *testing.T) {
	fs := NewFileSystem(fs)
	names := NewAssetNames(fs)

	f, err := names.Open("/test1")
	require.NoError(t, err, "open")
	defer f.Close()

	info, err := f.Stat()
	require.NoError(t, err, "stat")

	assert.Equal(t, path.Base(names.Lookup("/test1")),
		info.Name())

	_, err = names.Open("/does.not.exist")
	assert.True(t, os.IsNotExist(err),
		"non-existant file should return os.ErrNotExist")

	_, err = names.Open("does.not.exist")
	assert.True(t, os.IsNotExist(err),
		"non-existant file should return os.ErrNotExist")

	for _, name := range []string{
		"/test1",
		"test1",
		names.Lookup("/test1"),
		names.Lookup("test1"),
	} {
		f, err := names.Open(name)
		if assert.NoError(t, err, "open") {
			f.Close()
		}
	}
}
