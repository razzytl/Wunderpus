package ui

import "embed"

// DistFS contains the compiled frontend assets from the Vite build.
// Build the frontend first: cd ui && npm run build
//
//go:embed all:dist
var DistFS embed.FS
