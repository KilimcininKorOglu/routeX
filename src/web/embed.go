package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static
var staticFiles embed.FS

//go:embed locales
var localesFiles embed.FS

func StaticFS() http.Handler {
	sub, _ := fs.Sub(staticFiles, "static")
	return http.StripPrefix("/static/", http.FileServer(http.FS(sub)))
}

func LocalesFS() fs.FS {
	sub, _ := fs.Sub(localesFiles, "locales")
	return sub
}
