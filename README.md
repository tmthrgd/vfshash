# vfshash

[![GoDoc](https://godoc.org/go.tmthrgd.dev/vfshash?status.svg)](https://godoc.org/go.tmthrgd.dev/vfshash)
[![Build Status](https://travis-ci.com/tmthrgd/vfshash.svg?branch=master)](https://travis-ci.com/tmthrgd/vfshash)

A package to make a [`http.FileSystem`](https://godoc.org/net/http#FileSystem) content-addressable for use with [shurcooL/vfsgen](https://github.com/shurcooL/vfsgen). It adds a truncated cryptographic digest to file names.

This package is useful for giving web assets content-addressable URLs which can safely be used with long cache times (a year or more).

## Instalation

```bash
go get go.tmthrgd.dev/vfshash
```

## Usage

Follow the usage instructions for [shurcooL/vfsgen](https://github.com/shurcooL/vfsgen#usage) and wrap the fs with [`NewFileSystem`](https://godoc.org/go.tmthrgd.dev/vfshash#NewFileSystem) to generate a content-addressable file system.

```go
var fs http.FileSystem = http.Dir("/path/to/assets")

fs = vfshash.NewFileSystem(fs)

err := vfsgen.Generate(fs, vfsgen.Options{})
if err != nil {
	log.Fatalln(err)
}
```

Embedded in the file system is a manifest that maps the original names for files to their content-addressable equivalents. This can be accessed with the [`AssetsName`](https://godoc.org/go.tmthrgd.dev/vfshash#AssetsName) API.

```go
names := vfshash.NewAssetNames(assets)

// returns /example-deadbeef.css
names.Lookup("/example.css")

// returns /does.not.exist.txt as is
names.Lookup("/does.not.exist.txt")

// opens /example-deadbeef.js
names.Open("/example.js")
```

`AssetsName` implements `http.FileSystem` so it can be passed to [`http.FileServer`](https://godoc.org/net/http#FileServer) to serve assets with their original names.

If the `http.FileSystem` passed to [`NewAssetNames`](https://godoc.org/go.tmthrgd.dev/vfshash#NewAssetNames) doesn't contain the manifest, [`Lookup`](https://godoc.org/go.tmthrgd.dev/vfshash#AssetsName.Lookup) will return the name as is. This makes development easier as a regular [`http.Dir`](https://godoc.org/net/http#Dir) can be passed in without problem.

## License

[BSD 3-Clause License](LICENSE)