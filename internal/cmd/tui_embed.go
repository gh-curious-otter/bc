package cmd

import (
	_ "embed"
)

// tuiBundleJS is the precompiled TUI bundle (single JS file produced by
// `bun build --minify`). At runtime the TUI is extracted to a temp directory
// and executed via `bun run`.
//
// The bundle is built by `make build-local-tui-bundle` which writes to
// internal/cmd/tui-bundle/index.js. In dev checkouts without the bundle,
// this will be a tiny stub file and the code falls back to tui/dist/index.js.
//
//go:embed tui-bundle/index.js
var tuiBundleJS []byte

// hasEmbeddedTUI reports whether a real TUI bundle is embedded (vs the stub).
func hasEmbeddedTUI() bool {
	return len(tuiBundleJS) > 10_000 // stub is ~100 bytes, real bundle is MB
}
