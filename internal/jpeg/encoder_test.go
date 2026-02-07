package jpeg

import (
	"testing"
)

func TestEncodeCMYKSynthetic(t *testing.T) {
	// Create a 2x2 synthetic CMYK image (all mid-gray)
	width, height := 2, 2
	pixels := make([]byte, width*height*4)
	for i := 0; i < len(pixels); i += 4 {
		pixels[i] = 50   // C
		pixels[i+1] = 50 // M
		pixels[i+2] = 50 // Y
		pixels[i+3] = 50 // K
	}

	data, err := EncodeCMYK(pixels, width, height, nil, EncoderOptions{Quality: 85, CMYReduction: 15})
	if err != nil {
		t.Fatalf("EncodeCMYK: %v", err)
	}

	// Check JPEG magic
	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
		t.Fatal("output is not a valid JPEG (bad magic)")
	}

	// Verify we can read it back with GetInfo
	info, err := GetInfo(data)
	if err != nil {
		t.Fatalf("GetInfo on encoded output: %v", err)
	}

	if info.NumComponents != 4 {
		t.Errorf("expected 4 components, got %d", info.NumComponents)
	}
	if info.ColorSpace != "CMYK" && info.ColorSpace != "YCCK" {
		t.Errorf("expected CMYK or YCCK color space, got %s", info.ColorSpace)
	}

	t.Logf("Encoded %dx%d CMYK JPEG: %d bytes, %d components, %s",
		info.Width, info.Height, len(data), info.NumComponents, info.ColorSpace)
}
