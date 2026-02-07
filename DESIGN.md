# Design

This document describes the architecture and key design decisions behind RGBtoCMYK.

## Overview

RGBtoCMYK is a Go CLI tool that converts RGB JPEG images to CMYK JPEG images for professional printing. The tool uses CGO bindings to two C libraries — LittleCMS 2 (lcms2) for ICC color management and libjpeg-turbo for JPEG decoding and encoding — orchestrated through a three-stage pipeline: **decode**, **transform**, **encode**.

The tool replaces a multi-step ImageMagick workflow and produces output files roughly 50% smaller through channel-aware quantization and optimized Huffman coding.

## Architecture

```
                    ┌─────────────────────────────────────────────┐
                    │              CLI (cobra)                    │
                    │  convert | transform | encode | identify    │
                    └────────────────┬────────────────────────────┘
                                     │
                    ┌────────────────▼────────────────────────────┐
                    │           pipeline.Run()                    │
                    │  Orchestrates the three stages below        │
                    └──┬──────────────┬──────────────────┬────────┘
                       │              │                  │
              ┌────────▼───┐  ┌───────▼────────┐  ┌─────▼──────────┐
              │   Decode   │  │   Transform    │  │    Encode      │
              │ (libjpeg)  │  │   (lcms2)      │  │  (libjpeg)     │
              │            │  │                │  │                │
              │ JPEG→RGB   │  │ RGB→CMYK via   │  │ CMYK→JPEG     │
              │ + ICC      │  │ ICC profiles   │  │ + quant tables │
              │ extraction │  │                │  │ + ICC embed    │
              └────────────┘  └────────────────┘  └────────────────┘
```

### Data flow

1. **Decode**: libjpeg reads the JPEG file, forces RGB output (even for grayscale inputs), and extracts any ICC profile from APP2 markers.

2. **Transform**: lcms2 opens the source ICC profile (from the image, a user override, or the bundled sRGB v4 fallback) and the destination CMYK profile. It creates a `TYPE_RGB_8` → `TYPE_CMYK_8` transform and applies it row by row.

3. **Encode**: libjpeg writes the CMYK pixels as a 4-component JPEG with custom quantization tables, optimized Huffman coding, and the CMYK ICC profile embedded as APP2 marker chunks.

The intermediate representation between stages is `ir.CMYKImage` — a simple struct holding width, height, pixel bytes, and the ICC profile to embed.

## Package layout

```
internal/
  ir/cmykimage.go         Data contract: {Width, Height, Pixels []byte, ICC []byte}
  color/
    transform.go          lcms2 CGO: profile open, transform create/apply, cleanup
    profiles.go           ICC profile parsing, validation, go:embed sRGB fallback
    srgb_v4.icc           Embedded sRGB v4 ICC preference profile
  jpeg/
    decoder.go            libjpeg CGO: JPEG → RGB pixels + ICC extraction
    encoder.go            libjpeg CGO: CMYK pixels → JPEG + ICC embedding
    info.go               libjpeg CGO: read-only JPEG metadata (used by identify)
    icc.go                ICC_PROFILE APP2 marker extraction and reassembly
    quant.go              Quantization table generation with channel-aware scaling
  pipeline/
    pipeline.go           Wires decode → transform → encode
```

## Key design decisions

### CGO over pure Go

We use CGO bindings to lcms2 and libjpeg-turbo rather than pure Go libraries for two reasons:

1. **Correctness**: lcms2 is the reference implementation for ICC color management. Reimplementing ICC profile parsing, gamut mapping, and color space conversion in pure Go would be a large, error-prone undertaking.

2. **Performance**: libjpeg-turbo includes SIMD-optimized DCT and color conversion routines. The JPEG encoding path is the bottleneck for large images.

### Channel-aware quantization

Standard JPEG encoding applies the same quantization to all channels. For CMYK images destined for print, this is wasteful — the K (black) channel carries most of the perceptual detail (text, edges, fine structure), while the CMY channels carry broad color information.

RGBtoCMYK uses two quantization tables:

| Table | Channels | Quality | Rationale |
|-------|----------|---------|-----------|
| 0 | C, M, Y | `quality - cmy_reduction` | Color channels tolerate more compression |
| 1 | K | `quality` | Black channel preserves detail |

The default settings (quality=85, cmy_reduction=15) give K quality 85 and CMY quality 70. Combined with `optimize_coding = TRUE` (image-specific Huffman tables), this produces files roughly 50% smaller than ImageMagick's output with no perceptible quality loss.

The quantization tables are set by writing directly to `quant_tbl_ptrs[n]->quantval[i]` rather than using `jpeg_add_quant_table()`, because the latter applies its own scaling. Our tables are pre-scaled using the standard IJG formula and injected as-is.

### ICC profile handling

ICC profiles are embedded in JPEG files as APP2 marker segments, each prefixed with the tag `ICC_PROFILE\0` followed by a sequence number and total count. The maximum payload per marker is 65,533 bytes (65,535 minus the 2-byte length field), so large profiles like PSOcoated_v3.icc (2.1 MB) require ~34 chunks.

The chunking is performed in C rather than Go to comply with Go 1.21+'s CGO pointer rules. Passing a Go slice of pointers (each pointing into Go-allocated chunk buffers) through CGO triggers a panic. Moving the chunking loop into C means only a single flat `[]byte` crosses the Go/C boundary.

Profile extraction during decode works in reverse: APP2 markers are collected, filtered for the ICC tag, sorted by sequence number, and concatenated.

### Grayscale input handling

Grayscale JPEG inputs have 1 component and a grayscale ICC profile. The pipeline handles this transparently:

1. libjpeg converts grayscale to RGB during decoding (via `out_color_space = JCS_RGB`)
2. The pipeline detects the grayscale ICC profile by checking the color space field in the ICC header
3. The grayscale ICC is discarded and the bundled sRGB v4 profile is used instead

This avoids the `lcms2: failed to create transform` error that would occur if a grayscale profile were used with `TYPE_RGB_8`.

### Error handling in libjpeg

libjpeg's default error handler calls `exit()`, which would kill the entire Go process. Both the decoder and encoder install a custom error manager that uses `setjmp`/`longjmp` to recover from errors, captures the error message, and returns it to Go as a normal error value.

### Adobe APP14 marker

CMYK JPEGs require an Adobe APP14 marker to signal the color space to readers. libjpeg writes this marker automatically when `in_color_space = JCS_CMYK`, with transform code 0 ("as-is"). No manual pixel inversion is needed.

### No subsampling

All four CMYK components use 1x1 sampling factors (no chroma subsampling). CMYK data doesn't have the luminance/chrominance separation that makes 4:2:0 subsampling effective in YCbCr, and subsampling would introduce visible artifacts in the color channels.

### Memory management

Pixel buffers are Go-allocated `[]byte` slices. References are kept on the Go stack during CGO calls to prevent garbage collection. The only C-owned resources are lcms2 profile and transform handles, which are released by `Transform.Close()`. A `runtime.SetFinalizer` provides a safety net if `Close()` is not called explicitly.

The JPEG encoder uses `jpeg_mem_dest` to write to a C-allocated buffer, which is copied to Go memory immediately after encoding and then freed.

## File size comparison

Using `candidate-0.jpg` (7158x5250, 8.2 MB RGB with embedded sRGB) converted to CMYK with PSOcoated_v3.icc:

| Tool | Output size | Ratio vs ImageMagick |
|------|-------------|---------------------|
| ImageMagick 7 | 25.5 MB | 1.0x |
| RGBtoCMYK (q=85, cmy_reduction=15) | 12.8 MB | 0.50x |

The 2.1 MB PSOcoated_v3 ICC profile is embedded in both outputs. Subtracting the profile, the actual image data is ~10.7 MB (ours) vs ~23.4 MB (ImageMagick).

## Rendering intents

The `--intent` flag maps directly to lcms2 rendering intent constants:

| Flag value | lcms2 constant | Use case |
|------------|----------------|----------|
| `perceptual` | `INTENT_PERCEPTUAL` | Photos (default) — compresses gamut to preserve relationships |
| `relative` | `INTENT_RELATIVE_COLORIMETRIC` | Proofing — maps white point, clips out-of-gamut |
| `saturation` | `INTENT_SATURATION` | Graphics — maximizes color vividness |
| `absolute` | `INTENT_ABSOLUTE_COLORIMETRIC` | Proofing — preserves absolute colors including white point |

## Testing strategy

The test suite uses real-world JPEG files covering the input variations a production tool must handle:

- **Encoding variants**: Baseline, progressive, progressive with optimized Huffman tables
- **Color spaces**: sRGB v4, AdobeRGB 1998, Display P3, grayscale
- **ICC presence**: Embedded ICC, no ICC (sRGB fallback)
- **Quality range**: 1 through 100
- **Rendering intents**: All four ICC intents
- **Image types**: Photos and vector art rasterizations
- **Sizes**: Small (500x500) through large (10650x13426)

Each test verifies structural correctness of the output: valid JPEG framing (FFD8/FFD9), 4-component CMYK color space, dimension preservation, and a valid embedded CMYK ICC profile.
