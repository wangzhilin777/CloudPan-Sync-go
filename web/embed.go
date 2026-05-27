package webui

import (
	"embed"
	"io/fs"
)

//go:embed static/*
var assets embed.FS

func StaticFS() (fs.FS, error) {
	return fs.Sub(assets, "static")
}

func IndexHTML() ([]byte, error) {
	return assets.ReadFile("static/index.html")
}
