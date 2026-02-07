package pipeline

import (
	"os"
	"testing"

	"github.com/davesmith10/RGBtoCMYK/internal/color"
	"github.com/davesmith10/RGBtoCMYK/internal/jpeg"
)

func TestFullPipeline(t *testing.T) {
	inputData, err := os.ReadFile("/mnt/c/Users/daves/OneDrive/Desktop/rgb_to_cmyk/magick-workflow/candidate-sm.jpg")
	if err != nil {
		t.Skipf("test input not available: %v", err)
	}
	dstProfile, err := os.ReadFile("/mnt/c/Users/daves/OneDrive/Desktop/rgb_to_cmyk/magick-workflow/PSOcoated_v3.icc")
	if err != nil {
		t.Skipf("CMYK profile not available: %v", err)
	}

	result, err := Run(inputData, Options{
		DstProfile:   dstProfile,
		Quality:      85,
		CMYReduction: 15,
		Intent:       color.IntentPerceptual,
	})
	if err != nil {
		t.Fatalf("Pipeline: %v", err)
	}

	// Verify output is valid JPEG
	if len(result.Data) < 2 || result.Data[0] != 0xFF || result.Data[1] != 0xD8 {
		t.Fatal("output is not a valid JPEG")
	}

	// Verify output metadata
	info, err := jpeg.GetInfo(result.Data)
	if err != nil {
		t.Fatalf("GetInfo on output: %v", err)
	}

	if info.NumComponents != 4 {
		t.Errorf("expected 4 components, got %d", info.NumComponents)
	}
	if info.Width != 500 || info.Height != 500 {
		t.Errorf("unexpected dimensions: %dx%d", info.Width, info.Height)
	}
	if info.ICC == nil {
		t.Error("expected embedded ICC profile in output")
	}

	t.Logf("Pipeline output: %dx%d, %d components, %s, %d bytes (%.1f KB), ICC=%d bytes",
		info.Width, info.Height, info.NumComponents, info.ColorSpace,
		len(result.Data), float64(len(result.Data))/1024, len(info.ICC))
	t.Logf("Input size: %d bytes, Output size: %d bytes, Ratio: %.1f%%",
		len(inputData), len(result.Data), float64(len(result.Data))/float64(len(inputData))*100)
}
