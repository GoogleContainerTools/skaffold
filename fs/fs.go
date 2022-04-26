package fs

import (
	"embed"
	"io/fs"
)

var (

	//go:embed assets/*
	Assets embed.FS

	// AssetsFS for testing
	AssetsFS fs.ReadFileFS = Assets
)
