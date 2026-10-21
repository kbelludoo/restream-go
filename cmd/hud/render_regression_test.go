package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"image"
	"os"
	"testing"
)

func setFrameSize(t testing.TB, w, h int) {
	t.Helper()
	oldW, oldH := frameW, frameH
	frameW, frameH = w, h
	t.Cleanup(func() { frameW, frameH = oldW, oldH })
}

func sampleTelemetry(t testing.TB) TelemetryRoot {
	t.Helper()
	data, err := os.ReadFile("status_sample.json")
	if err != nil {
		t.Fatal(err)
	}
	var telemetry TelemetryRoot
	if err := json.Unmarshal(data, &telemetry); err != nil {
		t.Fatal(err)
	}
	return telemetry
}

func TestFrameReuse(t *testing.T) {
	for _, res := range []struct {
		name string
		w, h int
	}{{"720p", 1280, 720}, {"1080p", 1920, 1080}} {
		t.Run(res.name, func(t *testing.T) {
			setFrameSize(t, res.w, res.h)
			reused := newFrameRenderer()
			drawStaticBackground(reused.dc, sampleTelemetry(t))
			pix := reused.dc.Image().(*image.RGBA).Pix
			background := bytes.Clone(pix)
			fresh := newFrameRenderer()
			freshPix := fresh.dc.Image().(*image.RGBA).Pix
			clock := image.Rect(frameW-85, 20, frameW, 61)
			if frameW <= 1280 {
				clock = image.Rect(frameW-76, 6, frameW, 42)
			}
			for _, pulse := range []float64{0, 0.1, 1.7, 20, 90, 200} {
				copy(freshPix, background)
				drawDynamicOverlays(reused.dc, reused.marquee, pulse)
				drawDynamicOverlays(fresh.dc, fresh.marquee, pulse)
				for y := 0; y < frameH; y++ {
					for x := 0; x < frameW; x++ {
						// The wall clock can tick between the two renders.
						if image.Pt(x, y).In(clock) {
							continue
						}
						i := (y*frameW + x) * 4
						if !bytes.Equal(pix[i:i+4], freshPix[i:i+4]) {
							t.Fatalf("frame left a trail at (%d,%d), pulse=%v", x, y, pulse)
						}
					}
				}
			}
		})
	}
}

type frameWriterFunc func([]byte) (int, error)

func (f frameWriterFunc) Write(p []byte) (int, error) { return f(p) }

func TestStreamFramesWriterFailure(t *testing.T) {
	setFrameSize(t, 1280, 720)
	wantErr := errors.New("output closed")
	err := streamFrames(frameWriterFunc(func(p []byte) (int, error) {
		if len(p) != frameW*frameH*4 {
			t.Fatalf("incomplete RGBA frame: %d bytes", len(p))
		}
		return 0, wantErr
	}), 120, "", "status_sample.json")
	if !errors.Is(err, wantErr) {
		t.Fatalf("got %v, want %v", err, wantErr)
	}
}
