package go_ffmpeg

import (
	"path/filepath"
	"testing"
)

func TestDefaultOutputFile(t *testing.T) {
	input := filepath.Join(string(filepath.Separator), "Conversions", "movie.mkv")
	expected := filepath.Join(string(filepath.Separator), "Conversions", "Converted", "movie.mp4")

	if got := defaultOutputFile(input); got != expected {
		t.Errorf("defaultOutputFile(%q) = %q, want %q", input, got, expected)
	}
}
