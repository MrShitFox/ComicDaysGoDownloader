package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
)

// Comic Days scrambles images by splitting the picture into a
// divideNum x divideNum grid of equally sized cells and transposing that grid
// (the cell at row r, column c is swapped with the cell at row c, column r).
// Cell dimensions are rounded down to a multiple of `multiple` pixels, so the
// right and bottom edges of the image may contain a leftover margin that is
// never scrambled and must be copied over untouched.
const (
	divideNum = 4
	multiple  = 8
)

type ImageProcessor struct {
	Src image.Image
	Dst *image.RGBA
}

func NewImageContext(src image.Image) *ImageProcessor {
	return &ImageProcessor{
		Src: src,
		Dst: nil,
	}
}

// Deobfuscate reverses the Comic Days scrambling and stores the result in Dst.
func (ip *ImageProcessor) Deobfuscate(width, height int) *image.RGBA {
	if ip.Src == nil || width <= 0 || height <= 0 {
		ip.Dst = nil
		return nil
	}

	cellWidth := (width / (divideNum * multiple)) * multiple
	cellHeight := (height / (divideNum * multiple)) * multiple

	ip.Dst = image.NewRGBA(image.Rect(0, 0, width, height))

	// Copy the whole source first. This preserves any unscrambled pixels,
	// including the right and bottom margins that fall outside the cell grid.
	offset := ip.Src.Bounds().Min
	draw.Draw(ip.Dst, ip.Dst.Bounds(), ip.Src, offset, draw.Src)

	// If the image is too small to contain a single cell there is nothing to
	// unscramble; the copy above already produced the correct output.
	if cellWidth == 0 || cellHeight == 0 {
		return ip.Dst
	}

	// Transpose the grid back into place. Transposition is its own inverse, so
	// applying the same operation the server used restores the original layout.
	for row := 0; row < divideNum; row++ {
		for col := 0; col < divideNum; col++ {
			dstRect := image.Rect(
				col*cellWidth, row*cellHeight,
				col*cellWidth+cellWidth, row*cellHeight+cellHeight,
			)
			srcMin := image.Point{
				X: offset.X + row*cellWidth,
				Y: offset.Y + col*cellHeight,
			}
			draw.Draw(ip.Dst, dstRect, ip.Src, srcMin, draw.Src)
		}
	}

	return ip.Dst
}

func (ip *ImageProcessor) SaveImage(filePath string) error {
	if ip.Dst == nil {
		return fmt.Errorf("image has not been deobfuscated")
	}

	outFile, err := os.CreateTemp(filepath.Dir(filePath), ".tmp-*.png")
	if err != nil {
		return err
	}
	tmpName := outFile.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tmpName)
		}
	}()

	if err := png.Encode(outFile, ip.Dst); err != nil {
		_ = outFile.Close()
		return err
	}
	if err := outFile.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, filePath); err != nil {
		return err
	}
	removeTemp = false
	return nil
}
