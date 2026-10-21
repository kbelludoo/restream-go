package main

import "testing"

func BenchmarkHUDFrame(b *testing.B) {
	for _, res := range []struct {
		name string
		w, h int
	}{{"720p", 1280, 720}, {"1080p", 1920, 1080}} {
		b.Run(res.name, func(b *testing.B) {
			frameW, frameH = res.w, res.h
			renderer := newFrameRenderer()
			drawStaticBackground(renderer.dc, TelemetryRoot{})
			drawDynamicOverlays(renderer.dc, renderer.marquee, 0)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				drawDynamicOverlays(renderer.dc, renderer.marquee, float64(i)/10)
			}
		})
	}
}
