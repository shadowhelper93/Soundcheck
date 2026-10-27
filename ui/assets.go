// Package ui provides access to the built Soundcheck web user interface assets.
package ui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// Assets returns an fs.FS containing the built UI assets rooted at "dist".
func Assets() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}
