package icon

import (
	_ "embed"
)

//go:embed tray_active.png
var activeIcon []byte

//go:embed tray_paused.png
var pausedIcon []byte

// ActiveIconPNG returns the embedded 32x32 neon shield logo icon
func ActiveIconPNG() []byte {
	return activeIcon
}

// PausedIconPNG returns the embedded 32x32 dimmed grayscale logo icon
func PausedIconPNG() []byte {
	return pausedIcon
}
