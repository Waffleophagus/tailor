package frontend

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:dist
var embedded embed.FS

func FileSystem() http.FileSystem {
	dist, err := fs.Sub(embedded, "dist")
	if err != nil {
		panic(err)
	}
	return http.FS(dist)
}
