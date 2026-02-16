package format

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"strings"

	"golang.org/x/image/draw"
)

const ansiResetSeq = "\x1b[0m"

// RenderImageHalfBlock renders image data as half-block characters for terminal display.
// Uses U+2584 (Lower Half Block): top pixel = background color, bottom pixel = foreground color.
// Returns the rendered string and the number of lines.
func RenderImageHalfBlock(data []byte, maxWidth, maxHeight int) (string, int, error) {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", 0, err
	}

	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	if srcW == 0 || srcH == 0 {
		return "", 0, fmt.Errorf("empty image")
	}

	// Each terminal cell represents 2 vertical pixels, so max pixel height = maxHeight * 2
	maxPixelH := maxHeight * 2

	// Scale preserving aspect ratio
	dstW := maxWidth
	dstH := srcH * dstW / srcW
	if dstH > maxPixelH {
		dstH = maxPixelH
		dstW = srcW * dstH / srcH
	}
	if dstW > maxWidth {
		dstW = maxWidth
	}
	if dstW < 1 {
		dstW = 1
	}
	if dstH < 1 {
		dstH = 1
	}

	// Make height even for half-block pairing
	if dstH%2 != 0 {
		dstH++
	}

	// Scale image
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)

	// Render using half-block characters
	var sb strings.Builder
	lineCount := dstH / 2

	for row := 0; row < dstH; row += 2 {
		if row > 0 {
			sb.WriteString("\n")
		}
		for col := 0; col < dstW; col++ {
			// Top pixel → background color
			tr, tg, tb, _ := dst.At(col, row).RGBA()
			// Bottom pixel → foreground color
			br, bg, bb, _ := dst.At(col, row+1).RGBA()

			// RGBA returns 16-bit values, convert to 8-bit
			sb.WriteString(fmt.Sprintf("\x1b[48;2;%d;%d;%d;38;2;%d;%d;%dm▄",
				tr>>8, tg>>8, tb>>8,
				br>>8, bg>>8, bb>>8))
		}
		sb.WriteString(ansiResetSeq)
	}

	return sb.String(), lineCount, nil
}
