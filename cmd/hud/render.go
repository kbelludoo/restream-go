package main

import (
	"image"

	"github.com/fogleman/gg"
)

type frameRenderer struct {
	dc      *gg.Context
	marquee *gg.Context
}

func newFrameRenderer() *frameRenderer {
	dc := gg.NewContext(frameW, frameH)
	startX, height := 164, 52
	if frameW <= 1280 {
		startX, height = 143, 36
	}
	bounds := image.Rect(startX, frameH-height+2, frameW-10, frameH)
	frame := dc.Image().(*image.RGBA)
	// Rebase a view onto the footer to (0,0), keeping the full frame's stride.
	// gg requires zero-based bounds; image.RGBA.SubImage alone keeps its origin.
	footer := &image.RGBA{
		Pix:    frame.Pix[frame.PixOffset(bounds.Min.X, bounds.Min.Y):],
		Stride: frame.Stride,
		Rect:   image.Rect(0, 0, bounds.Dx(), bounds.Dy()),
	}
	marquee := gg.NewContextForRGBA(footer)
	marquee.Translate(-float64(bounds.Min.X), -float64(bounds.Min.Y))
	return &frameRenderer{dc: dc, marquee: marquee}
}
