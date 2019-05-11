package vfshash

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"golang.org/x/tools/godoc/vfs/httpfs"
	"golang.org/x/tools/godoc/vfs/mapfs"
)

var fs = httpfs.New(mapfs.New(map[string]string{
	"test1":             "test1",
	"example/test2":     "test2",
	"test3.txt":         "test3",
	".test4":            "test4",
	".test.test5":       "test5",
	"test6/.test":       "test6",
	"example/test7.txt": "test7",
}))

var expectedHashes = map[string]string{
	"/test1":             "/test1-sW7X0ks-y9QWTc2t",
	"/example/test2":     "/example/test2-bSAb7u-1ibCO8Gct",
	"/test3.txt":         "/test3-y4ct4rjSUJxUNEQ1.txt",
	"/.test4":            "/.IleqtEtCgTFCqorE.test4",
	"/.test.test5":       "/.test-ZMJv_js1xl37k6j9.test5",
	"/test6/.test":       "/test6/.MGNPwt0o5BKmhHcY.test",
	"/example/test7.txt": "/example/test7-zuH_3DDgV2WktHg3.txt",
}

func TestAssetsJSON(t *testing.T) {
	fs := NewFileSystem(fs)

	f, err := fs.Open("/.vfshash-assets.json")
	require.NoError(t, err, "open")
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	require.NoError(t, err, "read")

	var hashes map[string]string
	if assert.NoError(t, json.Unmarshal(b, &hashes)) {
		assert.Equal(t, expectedHashes, hashes)
	}

	_, err = f.Readdir(-1)
	assert.EqualError(t, err, syscall.ENOTDIR.Error())

	info, err := f.Stat()
	if assert.NoError(t, err, "stat") {
		assert.Equal(t, int64(len(b)), info.Size(), "info.Size")
		assert.Equal(t, os.FileMode(0400), info.Mode(), "info.Mode")
		assert.Empty(t, info.ModTime(), "info.ModTime")
		assert.False(t, info.IsDir(), "info.IsDir")
		assert.Nil(t, info.Sys(), "info.Sys")
	}
}

func TestFilePaths(t *testing.T) {
	fs := NewFileSystem(fs)

	for name, hashed := range expectedHashes {
		_, err := fs.Open(name)
		assert.EqualError(t, err,
			(&os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}).Error(),
			"file accessable by original path")

		f, err := fs.Open(hashed)
		if assert.NoError(t, err, "file not accessable by hashed path") {
			f.Close()
		}
	}
}

func TestRootDirectory(t *testing.T) {
	fs := NewFileSystem(fs)

	f, err := fs.Open("/")
	require.NoError(t, err)
	defer f.Close()

	files, err := f.Readdir(-1)
	require.NoError(t, err)

	var foundAssetsJSON bool
	for _, info := range files {
		if info.Name() == path.Base(assetsJSONPath) {
			foundAssetsJSON = true
		}
	}
	assert.True(t, foundAssetsJSON,
		"expected to find %s in Readdir result",
		assetsJSONPath)

	testDirHashes(t, files, "/")

	_, err = f.Readdir(1)
	assert.Error(t, err, "with count > 0")

	if _, err = f.Seek(1, io.SeekStart); err == nil {
		_, err = f.Readdir(1)
		assert.Error(t, err, "with seeked pos != 0")
	}
}

func TestSubdirectory(t *testing.T) {
	fs := NewFileSystem(fs)

	f, err := fs.Open("/example")
	require.NoError(t, err)
	defer f.Close()

	files, err := f.Readdir(-1)
	require.NoError(t, err)

	testDirHashes(t, files, "/example")
}

func testDirHashes(t *testing.T, files []os.FileInfo, dir string) {
	t.Helper()

	for name, hashed := range expectedHashes {
		if path.Dir(hashed) != dir {
			continue
		}

		var foundName, foundHashed bool
		for _, info := range files {
			switch info.Name() {
			case path.Base(name):
				foundName = true
			case path.Base(hashed):
				foundHashed = true
			}
		}

		assert.False(t, foundName,
			"expected to not find %s in Readdir result", name)
		assert.True(t, foundHashed,
			"expected to find %s in Readdir result", hashed)
	}
}
