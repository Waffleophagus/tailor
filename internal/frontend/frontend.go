package frontend

import (
	"embed"
	"io/fs"
	"net/http"
)

// dist/app is the Vite build output (gitignored). dist/placeholder is tracked so
// the embed compiles, and is served when the binary was built without the web UI.
//
//go:embed all:dist
var embedded embed.FS

func FileSystem() http.FileSystem {
	if app, err := fs.Sub(embedded, "dist/app"); err == nil {
		if _, err := fs.Stat(app, "index.html"); err == nil {
			return http.FS(app)
		}
	}
	placeholder, err := fs.Sub(embedded, "dist/placeholder")
	if err != nil {
		panic(err)
	}
	return http.FS(placeholder)
}
