//go:build windows

package divert

import "embed"

//go:embed bin/*
var embeddedBin embed.FS
