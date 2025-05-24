package main

import (
	"image"
	"image/draw"
	"image/png"
	"os"
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

func (ip *ImageProcessor) Deobfuscate(width, height int) *image.RGBA {
	spacingWidth := (width / 32) * 8
	spacingHeight := (height / 32) * 8

	ip.Dst = image.NewRGBA(image.Rect(0, 0, width, height))

	for x := 0; x+spacingWidth <= width; x += spacingWidth {
		for y := (x/spacingWidth)*spacingHeight + spacingHeight; y+spacingHeight <= height; y += spacingHeight {
			oldRect := image.Rect(x, y, x+spacingWidth, y+spacingHeight)
			newPosX := (y / spacingHeight) * spacingWidth
			newPosY := (x / spacingWidth) * spacingHeight
			newRect := image.Rect(newPosX, newPosY, newPosX+spacingWidth, newPosY+spacingHeight)

			draw.Draw(ip.Dst, oldRect, ip.Src, newRect.Min, draw.Src)
			draw.Draw(ip.Dst, newRect, ip.Src, oldRect.Min, draw.Src)
		}
	}

	for i := 0; i < 4; i++ {
		midLineX := i * spacingWidth
		midLineY := i * spacingHeight
		midRect := image.Rect(midLineX, midLineY, midLineX+spacingWidth, midLineY+spacingHeight)
		draw.Draw(ip.Dst, midRect, ip.Src, midRect.Min, draw.Src)
	}

	return ip.Dst
}

func (ip *ImageProcessor) SaveImage(filePath string) error {
	outFile, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer outFile.Close()
	return png.Encode(outFile, ip.Dst)
}

func (ip *ImageProcessor) RestoreRightTransparentStrip(width, height, stripWidth int) {
	if stripWidth <= 0 {
		return
	}
	sourceRect := image.Rect(width-stripWidth, 0, width, height)
	destRect := sourceRect
	draw.Draw(ip.Dst, destRect, ip.Src, sourceRect.Min, draw.Src)
}

func (ip *ImageProcessor) DetectTransparentStripWidth() int {
	bounds := ip.Dst.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	maxTransparentWidth := 0

	for x := width - 1; x >= 0; x-- {
		isTransparentColumn := true
		for y := 0; y < height; y++ {
			_, _, _, alpha := ip.Dst.At(x, y).RGBA()
			if alpha != 0 {
				isTransparentColumn = false
				break
			}
		}
		if isTransparentColumn {
			maxTransparentWidth++
		} else {
			break
		}
	}
	return maxTransparentWidth
}
