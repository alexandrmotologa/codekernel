package web

import "embed"

// DistFS contains the embedded React 19 single-page application build output.
//
//go:embed all:dist
var DistFS embed.FS
