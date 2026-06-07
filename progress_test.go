package go_ffmpeg

import (
	"errors"
	"testing"
	"time"
)

func TestNewProgress(t *testing.T) {
	startTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	duration := 2 * time.Hour
	line := "frame=  123 fps= 45 q=28.0 size=    1024KiB time=00:01:23.45 bitrate= 512.0kbits/s speed=1.5x"

	progress, err := newProgress(line, duration, startTime, "/input.mkv", "/output.mp4")
	if err != nil {
		t.Fatalf("newProgress() error = %v", err)
	}

	if progress.Frame != 123 {
		t.Errorf("Frame = %d, want 123", progress.Frame)
	}
	if progress.FPS != 45 {
		t.Errorf("FPS = %v, want 45", progress.FPS)
	}
	if progress.Q != 28.0 {
		t.Errorf("Q = %v, want 28", progress.Q)
	}
	if progress.Size != 1024 {
		t.Errorf("Size = %v, want 1024", progress.Size)
	}
	if progress.Time != 83*time.Second+450*time.Millisecond {
		t.Errorf("Time = %v, want 1m23.45s", progress.Time)
	}
	if progress.Bitrate != 512.0 {
		t.Errorf("Bitrate = %v, want 512", progress.Bitrate)
	}
	if progress.Speed != 1.5 {
		t.Errorf("Speed = %v, want 1.5", progress.Speed)
	}
	if progress.InputFile != "/input.mkv" {
		t.Errorf("InputFile = %q, want /input.mkv", progress.InputFile)
	}
	if progress.OutputFile != "/output.mp4" {
		t.Errorf("OutputFile = %q, want /output.mp4", progress.OutputFile)
	}
}

func TestNewProgressNoProgressInformation(t *testing.T) {
	_, err := newProgress("ffmpeg version 6.0", time.Minute, time.Now(), "", "")
	if !errors.Is(err, ErrNoProgressInformation) {
		t.Fatalf("newProgress() error = %v, want ErrNoProgressInformation", err)
	}
}

func TestNewProgressZeroDuration(t *testing.T) {
	line := "frame=1 fps=1 q=0.0 size=1KiB time=00:00:01.00 bitrate=1.0kbits/s speed=1x"

	progress, err := newProgress(line, 0, time.Now(), "", "")
	if err != nil {
		t.Fatalf("newProgress() error = %v", err)
	}

	if progress.PercentComplete != 0.01 {
		t.Errorf("PercentComplete = %v, want 0.01", progress.PercentComplete)
	}
}
