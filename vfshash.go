// Package vfshash offers a http.FileSystem that wraps an underlying
// http.FileSystem to make resources content-addressable by adding a
// truncated cryptographic digest to the file names.
package vfshash

import (
	"bytes"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	pathpkg "path"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/shurcooL/httpfs/vfsutil"
)

const assetsJSONPath = "/.vfshash-assets.json"

// FileSystem wraps a http.FileSystem to make assets content-addressable by
// adding a truncated cryptographic digest to the file names.
type FileSystem struct {
	fs http.FileSystem

	once      sync.Once
	names     map[string]string // /example.css -> /example-deadbeef.css
	namesRev  map[string]string // /example-deadbeef.css -> /example.css
	namesJSON []byte            // {"/example.css": "/example-deadbeef.css"}
	err       error
}

// NewFileSystem returns an FileSystem using the given http.FileSystem.
func NewFileSystem(fs http.FileSystem) *FileSystem {
	return &FileSystem{fs: fs}
}

// Open calls Open on the underlying http.FileSystem while making every file
// content-addressable.
func (fs *FileSystem) Open(name string) (http.File, error) {
	fs.once.Do(fs.computeHashes)
	if fs.err != nil {
		return nil, fs.err
	}

	switch name {
	case "/":
		f, err := fs.fs.Open(name)
		if err != nil {
			return nil, err
		}

		return rootFile{&dirNamesFile{f, fs, name}}, nil
	case assetsJSONPath:
		return assetsFile{bytes.NewReader(fs.namesJSON)}, nil
	}

	origName, ok := fs.namesRev[name]
	if ok {
		f, err := fs.fs.Open(origName)
		if err != nil {
			return nil, err
		}

		return &newNameFile{f, name}, nil
	}

	f, err := fs.fs.Open(name)
	if err != nil {
		return nil, err
	}

	if info, err := f.Stat(); err != nil {
		f.Close()
		return nil, err
	} else if !info.IsDir() {
		f.Close()
		return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
	}

	return &dirNamesFile{f, fs, name}, nil
}

func (fs *FileSystem) computeHashes() {
	fs.names, fs.namesRev = make(map[string]string), make(map[string]string)

	h := sha512.New()
	var sum [sha512.Size]byte

	fs.err = vfsutil.Walk(fs.fs, "/", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		f, err := fs.fs.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		h.Reset()
		if _, err := io.Copy(h, f); err != nil {
			return err
		}

		hashed := addHashToPath(path, h.Sum(sum[:0]))
		fs.names[path], fs.namesRev[hashed] = hashed, path
		return nil
	})
	if fs.err != nil {
		return
	}

	fs.namesJSON, fs.err = json.Marshal(fs.names)
}

func addHashToPath(path string, sum []byte) string {
	dir, name := pathpkg.Split(path)
	hash := base64.RawURLEncoding.EncodeToString(sum)[:16]

	var hashed string
	if ext := pathpkg.Ext(name); name == ext { // dot file
		hashed = "." + hash + name
	} else {
		hashed = strings.TrimSuffix(name, ext) + "-" + hash + ext
	}

	return pathpkg.Join(dir, hashed)
}

type dirNamesFile struct {
	http.File
	fs  *FileSystem
	dir string
}

func (f *dirNamesFile) Readdir(count int) ([]os.FileInfo, error) {
	info, err := f.File.Readdir(count)

	names := f.fs.names
	for i, v := range info {
		path := pathpkg.Join(f.dir, v.Name())
		if name, ok := names[path]; ok {
			info[i] = &newNameFileInfo{v, name}
		}
	}

	return info, err
}

type rootFile struct{ *dirNamesFile }

func (f rootFile) Readdir(count int) ([]os.FileInfo, error) {
	if pos, _ := f.Seek(0, io.SeekCurrent); pos != 0 || count > 0 {
		return nil, syscall.ENOTSUP
	}

	info, err := f.dirNamesFile.Readdir(count)
	if err == nil || len(info) > 0 {
		info = append(info, assetsFileInfo{int64(len(f.fs.namesJSON))})
	}

	return info, err
}

type newNameFileInfo struct {
	os.FileInfo
	name string
}

func (fi newNameFileInfo) Name() string { return pathpkg.Base(fi.name) }

type newNameFile struct {
	http.File
	name string
}

func (f *newNameFile) Stat() (os.FileInfo, error) {
	info, err := f.File.Stat()
	if err != nil {
		return nil, err
	}

	return &newNameFileInfo{info, f.name}, nil
}

type assetsFileInfo struct{ size int64 }

func (assetsFileInfo) Name() string       { return pathpkg.Base(assetsJSONPath) }
func (fi assetsFileInfo) Size() int64     { return fi.size }
func (assetsFileInfo) Mode() os.FileMode  { return 0400 }
func (assetsFileInfo) ModTime() time.Time { return time.Time{} }
func (assetsFileInfo) IsDir() bool        { return false }
func (assetsFileInfo) Sys() interface{}   { return nil }

type assetsFile struct{ *bytes.Reader }

func (assetsFile) Close() error                             { return nil }
func (assetsFile) Readdir(count int) ([]os.FileInfo, error) { return nil, syscall.ENOTDIR }
func (f assetsFile) Stat() (os.FileInfo, error)             { return assetsFileInfo{f.Size()}, nil }
