package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static/*
var staticEmbed embed.FS

// GetFileSystem returns an http.FileSystem rooted at the embedded 'static' directory.
func GetFileSystem() http.FileSystem {
	sub, err := fs.Sub(staticEmbed, "static")
	if err != nil {
		panic(err)
	}
	return http.FS(sub)
}
