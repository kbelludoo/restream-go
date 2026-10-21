package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Exercise the production encoder arguments against three local FLV outputs.
// No stream keys or public ingest endpoints are used.
func TestFFmpegFanout(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg unavailable")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	dir := t.TempDir()
	audio := filepath.Join(dir, "radio.mp3")
	if out, err := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100", "-t", "4", audio).CombinedOutput(); err != nil {
		t.Fatalf("audio fixture: %v\n%s", err, out)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, audio)
	}))
	defer server.Close()

	for _, res := range []struct {
		name string
		w, h int
	}{{"720p", 1280, 720}, {"1080p", 1920, 1080}} {
		t.Run(res.name, func(t *testing.T) {
			setFrameSize(t, res.w, res.h)
			var targets []string
			for i := 0; i < 3; i++ {
				targets = append(targets, fmt.Sprintf("[f=flv:flvflags=no_duration_filesize]%s-%d.flv", res.name, i))
			}
			command := newFFmpegCommand(10, strings.Join(targets, "|"), server.URL)
			args := command.Args[1:]
			args = append(args[:len(args)-1], "-t", "2", args[len(args)-1])
			cmd := exec.CommandContext(ctx, command.Path, args...)
			cmd.Dir = dir
			var logs bytes.Buffer
			cmd.Stderr = &logs
			stdin, err := cmd.StdinPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				_ = stdin.Close()
				t.Fatal(err)
			}
			renderer := newFrameRenderer()
			drawStaticBackground(renderer.dc, sampleTelemetry(t))
			var writeErr error
			for i := 0; i < 20; i++ {
				drawDynamicOverlays(renderer.dc, renderer.marquee, float64(i)/10)
				if _, writeErr = stdin.Write(renderer.dc.Image().(*image.RGBA).Pix); writeErr != nil {
					break
				}
			}
			_ = stdin.Close()
			if err := cmd.Wait(); err != nil {
				t.Fatalf("ffmpeg: %v (write: %v)\n%s", err, writeErr, &logs)
			}
			for i := 0; i < 3; i++ {
				out, err := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-count_frames",
					"-show_entries", "stream=codec_name,width,height,nb_read_frames", "-of", "json",
					filepath.Join(dir, fmt.Sprintf("%s-%d.flv", res.name, i))).Output()
				if err != nil {
					t.Fatal(err)
				}
				var result struct {
					Streams []struct {
						Codec  string `json:"codec_name"`
						Width  int    `json:"width"`
						Height int    `json:"height"`
						Frames string `json:"nb_read_frames"`
					} `json:"streams"`
				}
				if err := json.Unmarshal(out, &result); err != nil {
					t.Fatal(err)
				}
				if len(result.Streams) != 2 || result.Streams[0].Codec != "h264" ||
					result.Streams[0].Width != frameW || result.Streams[0].Height != frameH ||
					result.Streams[0].Frames != "20" || result.Streams[1].Codec != "aac" {
					t.Fatalf("incorrect output %d: %s", i, out)
				}
			}
		})
	}
}
