package web

import (
	"embed"
	"io/fs"
)

//go:embed dist/*
var content embed.FS

// Assets returns the embedded web assets as an fs.FS
func Assets() (fs.FS, error) {
	return fs.Sub(content, "dist")
}
