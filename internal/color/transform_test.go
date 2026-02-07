package color

import (
	"os"
	"testing"
)

func TestTransformKnownPixels(t *testing.T) {
	cmykICC, err := os.ReadFile("/mnt/c/Users/daves/OneDrive/Desktop/rgb_to_cmyk/magick-workflow/PSOcoated_v3.icc")
	if err != nil {
		t.Skipf("CMYK profile not available: %v", err)
	}

	xform, err := NewTransform(EmbeddedSRGB, cmykICC, IntentPerceptual)
	if err != nil {
		t.Fatalf("NewTransform: %v", err)
	}
	defer xform.Close()

	// Test: pure white (255,255,255) should have low CMYK values
	// Test: pure black (0,0,0) should have high K
	// Test: pure red (255,0,0) should have high M and Y
	pixels := []byte{
		255, 255, 255, // white
		0, 0, 0, // black
		255, 0, 0, // red
	}

	cmyk, err := xform.TransformPixels(pixels, 3, 1)
	if err != nil {
		t.Fatalf("TransformPixels: %v", err)
	}
	if len(cmyk) != 12 {
		t.Fatalf("expected 12 CMYK bytes, got %d", len(cmyk))
	}

	// White: C,M,Y should be near 0, K should be near 0
	t.Logf("White  → C=%d M=%d Y=%d K=%d", cmyk[0], cmyk[1], cmyk[2], cmyk[3])
	if cmyk[3] > 10 {
		t.Errorf("white K=%d, expected near 0", cmyk[3])
	}

	// Black: K should be high (>200)
	t.Logf("Black  → C=%d M=%d Y=%d K=%d", cmyk[4], cmyk[5], cmyk[6], cmyk[7])
	if cmyk[7] < 200 {
		t.Errorf("black K=%d, expected >200", cmyk[7])
	}

	// Red: should have significant M and Y
	t.Logf("Red    → C=%d M=%d Y=%d K=%d", cmyk[8], cmyk[9], cmyk[10], cmyk[11])
	if cmyk[9] < 100 {
		t.Errorf("red M=%d, expected >100", cmyk[9])
	}
}
