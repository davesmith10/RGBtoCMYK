package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/davesmith10/RGBtoCMYK/internal/color"
	"github.com/davesmith10/RGBtoCMYK/internal/jpeg"
)

const (
	testdataDir = "../../testdata"
	cmykProfile = "/mnt/c/Users/daves/OneDrive/Desktop/rgb_to_cmyk/magick-workflow/PSOcoated_v3.icc"
)

func loadCMYKProfile(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(cmykProfile)
	if err != nil {
		t.Skipf("CMYK profile not available: %v", err)
	}
	return data
}

func loadTestImage(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("test image not available: %v", err)
	}
	return data
}

// verifyOutput runs standard checks on the output of a convert pipeline.
func verifyOutput(t *testing.T, name string, input, output []byte, result *Result) {
	t.Helper()

	// Must be a valid JPEG
	if len(output) < 2 || output[0] != 0xFF || output[1] != 0xD8 {
		t.Fatalf("[%s] output is not a valid JPEG (bad FFD8 magic)", name)
	}

	// Must end with FFD9 (EOI marker)
	if output[len(output)-2] != 0xFF || output[len(output)-1] != 0xD9 {
		t.Errorf("[%s] output does not end with FFD9 (EOI marker)", name)
	}

	info, err := jpeg.GetInfo(output)
	if err != nil {
		t.Fatalf("[%s] GetInfo on output failed: %v", name, err)
	}

	// Must have 4 components (CMYK)
	if info.NumComponents != 4 {
		t.Errorf("[%s] expected 4 components, got %d", name, info.NumComponents)
	}

	// Color space must be CMYK or YCCK
	if info.ColorSpace != "CMYK" && info.ColorSpace != "YCCK" {
		t.Errorf("[%s] expected CMYK or YCCK color space, got %s", name, info.ColorSpace)
	}

	// Dimensions must match source
	if info.Width != result.SrcWidth || info.Height != result.SrcHeight {
		t.Errorf("[%s] dimensions mismatch: src %dx%d, output %dx%d",
			name, result.SrcWidth, result.SrcHeight, info.Width, info.Height)
	}

	// Must have embedded ICC profile
	if info.ICC == nil {
		t.Errorf("[%s] output is missing embedded ICC profile", name)
	} else {
		pi, err := color.ParseProfileInfo(info.ICC)
		if err != nil {
			t.Errorf("[%s] embedded ICC profile is invalid: %v", name, err)
		} else if pi.ColorSpace != "CMYK" {
			t.Errorf("[%s] embedded ICC color space is %s, expected CMYK", name, pi.ColorSpace)
		}
	}

	t.Logf("[%s] %dx%d, %d components, %s, input=%d bytes, output=%d bytes (%.0f%%)",
		name, info.Width, info.Height, info.NumComponents, info.ColorSpace,
		len(input), len(output), float64(len(output))/float64(len(input))*100)
}

// --- Progressive JPEG tests ---

func TestConvert_ProgressiveWithICC(t *testing.T) {
	profile := loadCMYKProfile(t)
	input := loadTestImage(t, filepath.Join(testdataDir, "progressive", "test-face-progressive.jpg"))

	result, err := Run(input, Options{
		DstProfile:   profile,
		Quality:      85,
		CMYReduction: 15,
		Intent:       color.IntentPerceptual,
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}
	verifyOutput(t, "progressive-with-icc", input, result.Data, result)
}

func TestConvert_ProgressiveOptimizedNoICC(t *testing.T) {
	profile := loadCMYKProfile(t)
	input := loadTestImage(t, filepath.Join(testdataDir, "progressive", "test-face-progressive-optimized.jpg"))

	result, err := Run(input, Options{
		DstProfile:   profile,
		Quality:      85,
		CMYReduction: 15,
		Intent:       color.IntentPerceptual,
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}
	verifyOutput(t, "progressive-optimized-no-icc", input, result.Data, result)
}

func TestConvert_RegularBaselineWithICC(t *testing.T) {
	profile := loadCMYKProfile(t)
	input := loadTestImage(t, filepath.Join(testdataDir, "progressive", "test-face-regular.jpg"))

	result, err := Run(input, Options{
		DstProfile:   profile,
		Quality:      85,
		CMYReduction: 15,
		Intent:       color.IntentPerceptual,
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}
	verifyOutput(t, "regular-baseline-with-icc", input, result.Data, result)
}

// --- Grayscale input tests (libjpeg converts gray→RGB, pipeline discards gray ICC) ---

func TestConvert_GrayscaleLandscape(t *testing.T) {
	profile := loadCMYKProfile(t)
	input := loadTestImage(t, filepath.Join(testdataDir, "openprint", "4x6-landscape-photo-sgray.jpg"))

	result, err := Run(input, Options{
		DstProfile:   profile,
		Quality:      85,
		CMYReduction: 15,
		Intent:       color.IntentPerceptual,
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}
	verifyOutput(t, "grayscale-landscape-4x6", input, result.Data, result)
}

func TestConvert_GrayscalePortrait(t *testing.T) {
	profile := loadCMYKProfile(t)
	input := loadTestImage(t, filepath.Join(testdataDir, "openprint", "4x6-portrait-photo-sgray.jpg"))

	result, err := Run(input, Options{
		DstProfile:   profile,
		Quality:      85,
		CMYReduction: 15,
		Intent:       color.IntentPerceptual,
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}
	verifyOutput(t, "grayscale-portrait-4x6", input, result.Data, result)
}

func TestConvert_GrayscaleLarge(t *testing.T) {
	profile := loadCMYKProfile(t)
	input := loadTestImage(t, filepath.Join(testdataDir, "openprint", "8x10-landscape-photo-sgray.jpg"))

	result, err := Run(input, Options{
		DstProfile:   profile,
		Quality:      85,
		CMYReduction: 15,
		Intent:       color.IntentPerceptual,
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}
	verifyOutput(t, "grayscale-landscape-8x10", input, result.Data, result)
}

// --- sRGB input tests ---

func TestConvert_SRGBLandscape(t *testing.T) {
	profile := loadCMYKProfile(t)
	input := loadTestImage(t, filepath.Join(testdataDir, "openprint", "4x6-landscape-photo-srgb.jpg"))

	result, err := Run(input, Options{
		DstProfile:   profile,
		Quality:      85,
		CMYReduction: 15,
		Intent:       color.IntentPerceptual,
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}
	verifyOutput(t, "srgb-landscape-4x6", input, result.Data, result)
}

func TestConvert_SRGBPortrait(t *testing.T) {
	profile := loadCMYKProfile(t)
	input := loadTestImage(t, filepath.Join(testdataDir, "openprint", "4x6-portrait-photo-srgb.jpg"))

	result, err := Run(input, Options{
		DstProfile:   profile,
		Quality:      85,
		CMYReduction: 15,
		Intent:       color.IntentPerceptual,
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}
	verifyOutput(t, "srgb-portrait-4x6", input, result.Data, result)
}

func TestConvert_SRGBLargePhoto(t *testing.T) {
	profile := loadCMYKProfile(t)
	input := loadTestImage(t, filepath.Join(testdataDir, "openprint", "8x10-landscape-photo-color.jpg"))

	result, err := Run(input, Options{
		DstProfile:   profile,
		Quality:      85,
		CMYReduction: 15,
		Intent:       color.IntentPerceptual,
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}
	verifyOutput(t, "srgb-landscape-8x10", input, result.Data, result)
}

// --- AdobeRGB input tests ---

func TestConvert_AdobeRGB_4x6(t *testing.T) {
	profile := loadCMYKProfile(t)
	input := loadTestImage(t, filepath.Join(testdataDir, "openprint", "4x6-portrait-photo-adobergb.jpg"))

	result, err := Run(input, Options{
		DstProfile:   profile,
		Quality:      85,
		CMYReduction: 15,
		Intent:       color.IntentPerceptual,
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}
	verifyOutput(t, "adobergb-portrait-4x6", input, result.Data, result)
}

func TestConvert_AdobeRGB_A6(t *testing.T) {
	profile := loadCMYKProfile(t)
	input := loadTestImage(t, filepath.Join(testdataDir, "openprint", "a6-portrait-photo-adobergb.jpg"))

	result, err := Run(input, Options{
		DstProfile:   profile,
		Quality:      85,
		CMYReduction: 15,
		Intent:       color.IntentPerceptual,
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}
	verifyOutput(t, "adobergb-portrait-a6", input, result.Data, result)
}

// --- Display P3 input tests ---

func TestConvert_DisplayP3_4x6(t *testing.T) {
	profile := loadCMYKProfile(t)
	input := loadTestImage(t, filepath.Join(testdataDir, "openprint", "4x6-portrait-photo-displayp3.jpg"))

	result, err := Run(input, Options{
		DstProfile:   profile,
		Quality:      85,
		CMYReduction: 15,
		Intent:       color.IntentPerceptual,
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}
	verifyOutput(t, "displayp3-portrait-4x6", input, result.Data, result)
}

func TestConvert_DisplayP3_A6(t *testing.T) {
	profile := loadCMYKProfile(t)
	input := loadTestImage(t, filepath.Join(testdataDir, "openprint", "a6-portrait-photo-displayp3.jpg"))

	result, err := Run(input, Options{
		DstProfile:   profile,
		Quality:      85,
		CMYReduction: 15,
		Intent:       color.IntentPerceptual,
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}
	verifyOutput(t, "displayp3-portrait-a6", input, result.Data, result)
}

// --- No ICC profile (sRGB fallback) tests ---

func TestConvert_NoICC_Vector(t *testing.T) {
	profile := loadCMYKProfile(t)
	input := loadTestImage(t, filepath.Join(testdataDir, "openprint", "4x6-portrait-vector-srgb.jpg"))

	result, err := Run(input, Options{
		DstProfile:   profile,
		Quality:      85,
		CMYReduction: 15,
		Intent:       color.IntentPerceptual,
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}
	verifyOutput(t, "no-icc-vector-4x6", input, result.Data, result)
}

func TestConvert_NoICC_A3Vector(t *testing.T) {
	profile := loadCMYKProfile(t)
	input := loadTestImage(t, filepath.Join(testdataDir, "openprint", "a3-portrait-vector-srgb.jpg"))

	result, err := Run(input, Options{
		DstProfile:   profile,
		Quality:      85,
		CMYReduction: 15,
		Intent:       color.IntentPerceptual,
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}
	verifyOutput(t, "no-icc-vector-a3", input, result.Data, result)
}

func TestConvert_NoICC_LegalVector(t *testing.T) {
	profile := loadCMYKProfile(t)
	input := loadTestImage(t, filepath.Join(testdataDir, "openprint", "legal-portrait-vector-srgb.jpg"))

	result, err := Run(input, Options{
		DstProfile:   profile,
		Quality:      85,
		CMYReduction: 15,
		Intent:       color.IntentPerceptual,
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}
	verifyOutput(t, "no-icc-vector-legal", input, result.Data, result)
}

func TestConvert_NoICC_TabloidVector(t *testing.T) {
	profile := loadCMYKProfile(t)
	input := loadTestImage(t, filepath.Join(testdataDir, "openprint", "tabloid-portrait-vector-srgb.jpg"))

	result, err := Run(input, Options{
		DstProfile:   profile,
		Quality:      85,
		CMYReduction: 15,
		Intent:       color.IntentPerceptual,
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}
	verifyOutput(t, "no-icc-vector-tabloid", input, result.Data, result)
}

// --- Rendering intent variations ---

func TestConvert_IntentRelative(t *testing.T) {
	profile := loadCMYKProfile(t)
	input := loadTestImage(t, filepath.Join(testdataDir, "openprint", "4x6-portrait-photo-srgb.jpg"))

	result, err := Run(input, Options{
		DstProfile:   profile,
		Quality:      85,
		CMYReduction: 15,
		Intent:       color.IntentRelativeColorimetric,
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}
	verifyOutput(t, "intent-relative", input, result.Data, result)
}

func TestConvert_IntentSaturation(t *testing.T) {
	profile := loadCMYKProfile(t)
	input := loadTestImage(t, filepath.Join(testdataDir, "openprint", "4x6-portrait-photo-srgb.jpg"))

	result, err := Run(input, Options{
		DstProfile:   profile,
		Quality:      85,
		CMYReduction: 15,
		Intent:       color.IntentSaturation,
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}
	verifyOutput(t, "intent-saturation", input, result.Data, result)
}

func TestConvert_IntentAbsolute(t *testing.T) {
	profile := loadCMYKProfile(t)
	input := loadTestImage(t, filepath.Join(testdataDir, "openprint", "4x6-portrait-photo-srgb.jpg"))

	result, err := Run(input, Options{
		DstProfile:   profile,
		Quality:      85,
		CMYReduction: 15,
		Intent:       color.IntentAbsoluteColorimetric,
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}
	verifyOutput(t, "intent-absolute", input, result.Data, result)
}

// --- Quality variation tests ---

func TestConvert_QualityLow(t *testing.T) {
	profile := loadCMYKProfile(t)
	input := loadTestImage(t, filepath.Join(testdataDir, "openprint", "a6-landscape-photo-srgb.jpg"))

	result, err := Run(input, Options{
		DstProfile:   profile,
		Quality:      50,
		CMYReduction: 15,
		Intent:       color.IntentPerceptual,
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}
	verifyOutput(t, "quality-50", input, result.Data, result)
}

func TestConvert_QualityMax(t *testing.T) {
	profile := loadCMYKProfile(t)
	input := loadTestImage(t, filepath.Join(testdataDir, "openprint", "a6-landscape-photo-srgb.jpg"))

	result, err := Run(input, Options{
		DstProfile:   profile,
		Quality:      100,
		CMYReduction: 0,
		Intent:       color.IntentPerceptual,
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}
	verifyOutput(t, "quality-100", input, result.Data, result)
}

func TestConvert_QualityMinimal(t *testing.T) {
	profile := loadCMYKProfile(t)
	input := loadTestImage(t, filepath.Join(testdataDir, "openprint", "a6-portrait-vector-srgb.jpg"))

	result, err := Run(input, Options{
		DstProfile:   profile,
		Quality:      1,
		CMYReduction: 0,
		Intent:       color.IntentPerceptual,
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}
	verifyOutput(t, "quality-1", input, result.Data, result)
}

// --- Source profile override test ---

func TestConvert_SrcProfileOverride(t *testing.T) {
	profile := loadCMYKProfile(t)
	input := loadTestImage(t, filepath.Join(testdataDir, "openprint", "4x6-portrait-photo-srgb.jpg"))

	// Override with the embedded sRGB — result should still be valid
	result, err := Run(input, Options{
		SrcProfileOverride: color.EmbeddedSRGB,
		DstProfile:         profile,
		Quality:             85,
		CMYReduction:        15,
		Intent:              color.IntentPerceptual,
	})
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}
	verifyOutput(t, "src-profile-override", input, result.Data, result)
}
