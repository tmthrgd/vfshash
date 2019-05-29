package vfshash_test

import (
	"log"
	"net/http"

	"github.com/shurcooL/vfsgen"
	"go.tmthrgd.dev/vfshash"
)

func ExampleFileSystem() {
	fs := vfshash.NewFileSystem(http.Dir("assets"))

	err := vfsgen.Generate(fs, vfsgen.Options{})
	if err != nil {
		log.Fatal(err)
	}
}

var assets struct{ http.FileSystem }

func ExampleAssetNames() {
	names := vfshash.NewAssetNames(assets.FileSystem)

	// returns /example-deadbeef.css
	names.Lookup("/example.css")

	// returns /does.not.exist.txt as is
	names.Lookup("/does.not.exist.txt")

	// opens /example-deadbeef.js
	names.Open("/example.js")
}
