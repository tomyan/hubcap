package docs

import "embed"

//go:embed commands/*.md
var Commands embed.FS
