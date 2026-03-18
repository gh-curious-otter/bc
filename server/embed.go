package server

import (
	"embed"
	"io/fs"
)

//go:embed web/dist
var webUI embed.FS

// WebDist returns the embedded web/dist filesystem, or nil when only
// placeholder files are present (i.e. the UI has not been built yet).
func WebDist() fs.FS {
	sub, err := fs.Sub(webUI, "web/dist")
	if err != nil {
		return nil
	}
	entries, err := fs.ReadDir(sub, ".")
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if e.Name() != "placeholder.txt" && e.Name() != ".gitkeep" {
			return sub
		}
	}
	return nil
}
