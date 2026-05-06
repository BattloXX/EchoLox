// Package webembed embeds the web UI static files.
package webembed

import "embed"

//go:embed web
var Files embed.FS
