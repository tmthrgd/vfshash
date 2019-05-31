package vfshash

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"
)

// AssetNames maps asset paths to their content-addressable equivalents.
type AssetNames struct {
	fs http.FileSystem

	once  sync.Once
	names map[string]string // /example.css -> /example-deadbeef.css
}

// NewAssetNames returns an AssetNames using the given http.FileSystem.
func NewAssetNames(fs http.FileSystem) *AssetNames {
	return &AssetNames{fs: fs}
}

func (n *AssetNames) load() {
	f, err := n.fs.Open(assetsJSONPath)
	switch {
	case os.IsNotExist(err):
		// The underlying http.FileSystem might not be one that went
		// through this package. This makes Lookup a no-op which can
		// simplify development.
		return
	case err != nil:
		panic(err)
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&n.names); err != nil {
		panic(err)
	}
}

// IsContentAddressable reports whether the underlying http.FileSystem is using
// content-addressable names. This is the case when it was wrapped with
// FileSystem.
func (n *AssetNames) IsContentAddressable() bool {
	n.once.Do(n.load)
	return n.names != nil
}

// Lookup returns the content-addressable name of an asset that matches the
// given name.
//
// If the name isn't known, or the http.FileSystem wasn't wrapped with
// FileSystem, the name is returned as is.
func (n *AssetNames) Lookup(name string) string {
	n.once.Do(n.load)

	name = cleanPath(name)

	if hashed, ok := n.names[name]; ok {
		return hashed
	}

	return name
}

// Open converts name into the content-addressable name of an asset and then
// calls Open on the underlying http.FileSystem.
func (n *AssetNames) Open(name string) (http.File, error) {
	return n.fs.Open(n.Lookup(name))
}
