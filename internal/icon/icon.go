package icon

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
)

// ActiveIconPNG generates a 32x32 PNG icon for active status (green badge)
func ActiveIconPNG() []byte {
	return generateTrayIcon(color.RGBA{R: 34, G: 197, B: 94, A: 255}) // Emerald Green
}

// PausedIconPNG generates a 32x32 PNG icon for paused status (gray badge)
func PausedIconPNG() []byte {
	return generateTrayIcon(color.RGBA{R: 156, G: 163, B: 175, A: 255}) // Cool Gray
}

func generateTrayIcon(badgeColor color.Color) []byte {
	const size = 32
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Draw clean transparent background
	draw.Draw(img, img.Bounds(), image.Transparent, image.Point{}, draw.Src)

	// Draw sleek modern shield base in dark slate / white
	shieldColor := color.RGBA{R: 59, G: 130, B: 246, A: 255} // Blue
	for y := 4; y < 28; y++ {
		for x := 4; x < 28; x++ {
			// Shield geometry equation
			dx := float64(x - 16)
			dy := float64(y - 6)
			if y <= 16 {
				if dx*dx/121.0+dy*dy/100.0 <= 1.0 {
					img.Set(x, y, shieldColor)
				}
			} else {
				// Taper to point at bottom
				widthAtY := 11.0 * (1.0 - float64(y-16)/12.0)
				if widthAtY > 0 && dx >= -widthAtY && dx <= widthAtY {
					img.Set(x, y, shieldColor)
				}
			}
		}
	}

	// Draw lightning bolt ⚡ inside shield (white)
	boltColor := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	boltCoords := [][2]int{
		{16, 8}, {15, 9}, {14, 10}, {13, 11}, {14, 12}, {15, 12}, {16, 12}, {17, 12},
		{15, 13}, {14, 14}, {15, 14}, {16, 14}, {15, 15}, {14, 16}, {13, 17}, {14, 18},
		{15, 19}, {16, 20}, {16, 21}, {16, 22},
	}
	for _, pt := range boltCoords {
		img.Set(pt[0], pt[1], boltColor)
		img.Set(pt[0]+1, pt[1], boltColor)
	}

	// Draw status indicator circle at bottom right (green = active, gray = paused)
	badgeX, badgeY, radius := 24, 24, 5
	for y := badgeY - radius; y <= badgeY + radius; y++ {
		for x := badgeX - radius; x <= badgeX + radius; x++ {
			dx := x - badgeX
			dy := y - badgeY
			if dx*dx+dy*dy <= radius*radius {
				img.Set(x, y, badgeColor)
			}
		}
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
