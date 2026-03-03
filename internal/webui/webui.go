// Package webui provides the embedded static assets for the Hangar web UI.
// The assets are built from the webui/ directory by running 'make build-ui'
// and embedded into the binary at compile time.
package webui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:assets/dist
var webAssets embed.FS

// Assets returns an http.FileSystem serving the compiled web UI assets.
// The files are served from the embedded assets/dist directory.
func Assets() http.FileSystem {
	sub, err := fs.Sub(webAssets, "assets/dist")
	if err != nil {
		// This should never happen with a valid embed directive.
		panic("webui: failed to sub assets/dist: " + err.Error())
	}
	return http.FS(sub)
}
