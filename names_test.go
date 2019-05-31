package vfshash

import (
	"io"
	"net/http"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/godoc/vfs/httpfs"
	"golang.org/x/tools/godoc/vfs/mapfs"
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

func TestIsContentAddressable(t *testing.T) {
	assert.True(t, NewAssetNames(NewFileSystem(fs)).IsContentAddressable(),
		"IsContentAddressable for mangled FileSystem")
	assert.False(t, NewAssetNames(fs).IsContentAddressable(),
		"IsContentAddressable for plain http.FileSystem")

	assert.True(t, NewAssetNames(NewFileSystem(emptyDirFS{})).IsContentAddressable(),
		"IsContentAddressable for empty mangled FileSystem")
	assert.False(t, NewAssetNames(emptyDirFS{}).IsContentAddressable(),
		"IsContentAddressable for empty plain http.FileSystem")
}

type emptyDirFS struct{}

func (emptyDirFS) Open(name string) (http.File, error) {
	if name != "/" {
		return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
	}

	return emptyDir{}, nil
}

type emptyDir struct{}

func (emptyDir) Close() error                       { return nil }
func (emptyDir) Read([]byte) (int, error)           { return 0, io.EOF }
func (emptyDir) Seek(int64, int) (int64, error)     { return 0, nil }
func (emptyDir) Readdir(int) ([]os.FileInfo, error) { return []os.FileInfo{}, nil }
func (emptyDir) Stat() (os.FileInfo, error)         { return emptyDirInfo{}, nil }

type emptyDirInfo struct{}

func (emptyDirInfo) Name() string       { return "" }
func (emptyDirInfo) Size() int64        { return 0 }
func (emptyDirInfo) Mode() os.FileMode  { return 0 }
func (emptyDirInfo) ModTime() time.Time { return time.Now() }
func (emptyDirInfo) IsDir() bool        { return true }
func (emptyDirInfo) Sys() interface{}   { return nil }
