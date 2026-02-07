package jpeg

import (
	"os"
	"testing"
)

func TestDecodeRGB(t *testing.T) {
	data, err := os.ReadFile("/mnt/c/Users/daves/OneDrive/Desktop/rgb_to_cmyk/magick-workflow/candidate-sm.jpg")
	if err != nil {
		t.Skipf("test file not available: %v", err)
	}

	dec, err := DecodeRGB(data)
	if err != nil {
		t.Fatalf("DecodeRGB: %v", err)
	}

	if dec.Width != 500 || dec.Height != 500 {
		t.Errorf("unexpected dimensions: %dx%d", dec.Width, dec.Height)
	}

	expectedPixels := 500 * 500 * 3
	if len(dec.Pixels) != expectedPixels {
		t.Errorf("expected %d pixel bytes, got %d", expectedPixels, len(dec.Pixels))
	}

	if dec.ICC == nil {
		t.Error("expected ICC profile, got nil")
	} else {
		t.Logf("ICC profile: %d bytes", len(dec.ICC))
	}
}
