package format

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"strings"

	"github.com/BourgeoisBear/rasterm"
	xdraw "golang.org/x/image/draw"
)

const ansiResetSeq = "\x1b[0m"

// RenderImage renders image data using the best available terminal protocol.
// Returns the rendered string and the number of lines.
func RenderImage(data []byte, maxWidth, maxHeight int) (string, int, error) {
	proto := DetectImageProtocol()

	switch proto {
	case ProtoKitty:
		return renderKitty(data, maxWidth, maxHeight)
	case ProtoIterm:
		return renderIterm(data, maxWidth, maxHeight)
	case ProtoSixel:
		return renderSixel(data, maxWidth, maxHeight)
	default:
		return RenderImageHalfBlock(data, maxWidth, maxHeight)
	}
}

// renderKitty renders using the Kitty graphics protocol.
func renderKitty(data []byte, maxCols, maxRows int) (string, int, error) {
	src, err := decodeImage(data)
	if err != nil {
		return "", 0, err
	}

	cols, rows := fitCellDimensions(src, maxCols, maxRows)

	var buf bytes.Buffer
	err = rasterm.KittyWriteImage(&buf, src, rasterm.KittyImgOpts{
		DstCols: uint32(cols),
		DstRows: uint32(rows),
	})
	if err != nil {
		// Fallback to half-blocks on error
		return RenderImageHalfBlock(data, maxCols, maxRows)
	}

	// The Kitty escape sequence occupies `rows` terminal lines.
	// Pad with newlines so the TUI layout accounts for the space.
	result := buf.String()
	for i := 1; i < rows; i++ {
		result += "\n"
	}

	return result, rows, nil
}

// renderIterm renders using the iTerm2 inline image protocol.
func renderIterm(data []byte, maxCols, maxRows int) (string, int, error) {
	src, err := decodeImage(data)
	if err != nil {
		return "", 0, err
	}

	cols, rows := fitCellDimensions(src, maxCols, maxRows)

	var buf bytes.Buffer
	err = rasterm.ItermWriteImageWithOptions(&buf, src, rasterm.ItermImgOpts{
		Width:         fmt.Sprintf("%d", cols),
		Height:        fmt.Sprintf("%d", rows),
		DisplayInline: true,
	})
	if err != nil {
		return RenderImageHalfBlock(data, maxCols, maxRows)
	}

	result := buf.String()
	for i := 1; i < rows; i++ {
		result += "\n"
	}

	return result, rows, nil
}

// renderSixel renders using the Sixel graphics protocol.
func renderSixel(data []byte, maxCols, maxRows int) (string, int, error) {
	src, err := decodeImage(data)
	if err != nil {
		return "", 0, err
	}

	// Scale image: estimate ~8 pixels per cell width, ~16 per cell height
	pixW := maxCols * 8
	pixH := maxRows * 16
	scaled := scaleImage(src, pixW, pixH)

	// Convert to paletted image (Sixel requires it)
	bounds := scaled.Bounds()
	palImg := image.NewPaletted(bounds, palette.Plan9)
	draw.FloydSteinberg.Draw(palImg, bounds, scaled, bounds.Min)

	var buf bytes.Buffer
	err = rasterm.SixelWriteImage(&buf, palImg)
	if err != nil {
		return RenderImageHalfBlock(data, maxCols, maxRows)
	}

	// Sixel images occupy rows based on pixel height / 6 pixels per sixel row,
	// but in terminal cells it's approximately pixH / cell_height.
	// Use maxRows as the line count since we sized to fit.
	_, rows := fitCellDimensions(src, maxCols, maxRows)

	result := buf.String()
	for i := 1; i < rows; i++ {
		result += "\n"
	}

	return result, rows, nil
}

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
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, xdraw.Over, nil)

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

// Helper functions

func decodeImage(data []byte) (image.Image, error) {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	bounds := src.Bounds()
	if bounds.Dx() == 0 || bounds.Dy() == 0 {
		return nil, fmt.Errorf("empty image")
	}
	return src, nil
}

// fitCellDimensions calculates how many terminal cells (cols x rows) an image
// should occupy, preserving aspect ratio within the given maximums.
func fitCellDimensions(src image.Image, maxCols, maxRows int) (int, int) {
	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	// Approximate: terminal cells are roughly 1:2 aspect (width:height in pixels)
	// So 1 cell = ~1 unit wide, ~2 units tall
	// Scale image aspect ratio accordingly
	aspectRatio := float64(srcW) / float64(srcH)

	cols := maxCols
	rows := int(float64(cols) / aspectRatio / 2.0)

	if rows > maxRows {
		rows = maxRows
		cols = int(float64(rows) * aspectRatio * 2.0)
	}
	if cols > maxCols {
		cols = maxCols
	}

	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}

	return cols, rows
}

// scaleImage scales src to fit within maxW x maxH pixels, preserving aspect ratio.
func scaleImage(src image.Image, maxW, maxH int) *image.RGBA {
	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	dstW := maxW
	dstH := srcH * dstW / srcW
	if dstH > maxH {
		dstH = maxH
		dstW = srcW * dstH / srcH
	}
	if dstW < 1 {
		dstW = 1
	}
	if dstH < 1 {
		dstH = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))

	// Draw white background first (for transparency)
	draw.Draw(dst, dst.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, xdraw.Over, nil)

	return dst
}
