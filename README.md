# RGBtoCMYK

A Go CLI tool that converts RGB JPEG images to CMYK JPEG images suitable for professional printing. It replaces multi-step ImageMagick workflows with a single command and produces significantly smaller output files through channel-aware quantization and optimized Huffman coding.

## Motivation

The standard ImageMagick approach to RGB-to-CMYK conversion:

```bash
magick input.jpg +profile icm \
  -profile sRGB_v4_ICC_preference.icc \
  -profile PSOcoated_v3.icc \
  output.jpg
```

produces unnecessarily large files. A 8.2 MB RGB input yields a 25 MB CMYK output. RGBtoCMYK produces a 12.8 MB output from the same input — roughly 50% smaller — by applying channel-aware quantization that allocates more bits to the K (black) channel where the human eye is most sensitive to detail, and fewer bits to the CMY channels.

## Requirements

- Go 1.21+ (built and tested with Go 1.25.5)
- [LittleCMS 2](https://www.littlecms.com/) (lcms2) development headers
- [libjpeg-turbo](https://libjpeg-turbo.org/) development headers
- pkg-config

### Installing dependencies

**Fedora / RHEL / CentOS:**
```bash
sudo dnf install lcms2-devel libjpeg-turbo-devel
```

**Debian / Ubuntu:**
```bash
sudo apt install liblcms2-dev libjpeg-turbo8-dev
```

**macOS (Homebrew):**
```bash
brew install little-cms2 jpeg-turbo
```

Verify with:
```bash
pkg-config --exists lcms2 libjpeg && echo "OK"
```

## Building

```bash
make build
```

This produces `bin/rgbtocmyk`.

## Usage

### convert — Full RGB-to-CMYK pipeline

```bash
rgbtocmyk convert \
  -i input.jpg \
  -o output.jpg \
  --profile PSOcoated_v3.icc \
  --quality 85
```

| Flag | Default | Description |
|------|---------|-------------|
| `-i, --input` | (required) | Input RGB JPEG file |
| `-o, --output` | (required) | Output CMYK JPEG file |
| `--profile` | (required) | Destination CMYK ICC profile |
| `--src-profile` | (auto) | Override source RGB ICC profile |
| `--quality` | 85 | JPEG quality (1-100) |
| `--cmy-reduction` | 15 | Quality reduction for CMY channels relative to K |
| `--intent` | perceptual | Rendering intent: `perceptual`, `relative`, `saturation`, `absolute` |

The source RGB profile is determined automatically: the tool uses the ICC profile embedded in the input JPEG if present, otherwise falls back to the bundled sRGB v4 profile. The `--src-profile` flag overrides this.

Grayscale JPEG inputs are handled transparently — libjpeg converts to RGB during decoding and the pipeline uses sRGB for the color transform.

### identify — Inspect image metadata

```bash
rgbtocmyk identify image.jpg
```

Prints dimensions, component count, color space, file size, and ICC profile details.

Example output:
```
File:       candidate-0.jpg
Dimensions: 7158 x 5250
Components: 3
Color space: YCbCr
File size:  8565760 bytes (8.2 MB)
ICC profile: 456 bytes
  Version:     4.3.0
  Color space: RGB
  PCS:         CIEXYZ
  Class:       Display
```

### transform — Color transform only (raw output)

```bash
rgbtocmyk transform \
  -i input.jpg \
  -o output.raw \
  --profile PSOcoated_v3.icc
```

Writes raw interleaved CMYK bytes (4 bytes per pixel, row-major) and a JSON sidecar with width, height, and format metadata.

### encode — Encode raw CMYK to JPEG

```bash
rgbtocmyk encode \
  -i input.raw \
  -o output.jpg \
  --width 1440 --height 2160 \
  --icc PSOcoated_v3.icc
```

Encodes raw CMYK pixel data (from `transform` or other sources) to a CMYK JPEG with optional ICC profile embedding.

## Testing

```bash
make test
```

The test suite covers:

- **Progressive JPEGs** — progressive scan, progressive with optimized Huffman, regular baseline
- **Grayscale inputs** — grayscale ICC profiles are detected and handled via sRGB fallback
- **Multiple RGB color spaces** — sRGB v4, AdobeRGB 1998, Display P3
- **No embedded ICC** — falls back to bundled sRGB v4
- **All four rendering intents** — perceptual, relative colorimetric, saturation, absolute colorimetric
- **Quality extremes** — quality 1 through 100
- **Source profile override** — explicit `--src-profile` flag

Every test verifies: valid JPEG structure (FFD8/FFD9 markers), 4 CMYK components, correct color space, dimension preservation, and embedded CMYK ICC profile.

## Project structure

```
RGBtoCMYK/
  cmd/rgbtocmyk/          CLI entry point and subcommands
  internal/
    ir/                   CMYKImage intermediate representation
    color/                lcms2 CGO bindings, ICC profile handling
    jpeg/                 libjpeg-turbo CGO bindings (decode, encode, ICC chunking)
    pipeline/             Orchestrates decode -> transform -> encode
  testdata/               Test images (progressive, various color spaces)
```

See [DESIGN.md](DESIGN.md) for architecture details.

## License

Apache License 2.0. See [LICENSE](LICENSE).
