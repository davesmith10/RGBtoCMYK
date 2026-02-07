package pipeline

import (
	"fmt"

	"github.com/davesmith10/RGBtoCMYK/internal/color"
	"github.com/davesmith10/RGBtoCMYK/internal/jpeg"
)

// Options controls the full RGB→CMYK conversion pipeline.
type Options struct {
	SrcProfileOverride []byte // optional: override source RGB ICC profile
	DstProfile         []byte // required: destination CMYK ICC profile
	Quality            int    // JPEG quality (1-100)
	CMYReduction       int    // quality reduction for CMY channels
	Intent             int    // lcms2 rendering intent
}

// Result holds the output of a pipeline run.
type Result struct {
	Data      []byte // encoded CMYK JPEG
	SrcWidth  int
	SrcHeight int
}

// Run executes the full RGB→CMYK pipeline: decode → color transform → encode.
func Run(jpegData []byte, opts Options) (*Result, error) {
	// 1. Decode RGB JPEG
	decoded, err := jpeg.DecodeRGB(jpegData)
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	// 2. Determine source ICC profile
	// If the embedded profile is grayscale, discard it — libjpeg already
	// converted the pixels to RGB, so we need an RGB source profile.
	srcICC := opts.SrcProfileOverride
	if srcICC == nil {
		srcICC = decoded.ICC
	}
	if srcICC != nil {
		if pi, err := color.ParseProfileInfo(srcICC); err == nil && pi.ColorSpace == "GRAY" {
			srcICC = nil // fall through to sRGB fallback
		}
	}
	if srcICC == nil {
		srcICC = color.EmbeddedSRGB
	}

	// 3. Color transform RGB → CMYK
	xform, err := color.NewTransform(srcICC, opts.DstProfile, opts.Intent)
	if err != nil {
		return nil, fmt.Errorf("color transform setup: %w", err)
	}
	defer xform.Close()

	cmykPixels, err := xform.TransformPixels(decoded.Pixels, decoded.Width, decoded.Height)
	if err != nil {
		return nil, fmt.Errorf("color transform: %w", err)
	}

	// 4. Encode CMYK JPEG
	encoded, err := jpeg.EncodeCMYK(cmykPixels, decoded.Width, decoded.Height, opts.DstProfile, jpeg.EncoderOptions{
		Quality:      opts.Quality,
		CMYReduction: opts.CMYReduction,
	})
	if err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}

	return &Result{
		Data:      encoded,
		SrcWidth:  decoded.Width,
		SrcHeight: decoded.Height,
	}, nil
}
