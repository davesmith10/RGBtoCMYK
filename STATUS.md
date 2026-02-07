# RGBtoCMYK — Project Status

## Current Phase: COMPLETE (all 8 phases done)

## Results
- **ImageMagick output**: 25.5 MB (from 8.2 MB input)
- **Our output**: 12.8 MB (~50% smaller than ImageMagick)
- Valid CMYK JPEG: 4 components, embedded PSOcoated_v3 ICC, CMYK color space

## Completed Phases
- **Phase 0**: System deps (lcms2 2.12, libjpeg 62)
- **Phase 1**: Skeleton (go.mod, Makefile, cobra CLI, CMYKImage IR, CGO linkage)
- **Phase 2**: ICC profile loading/validation, APP2 extraction/chunking, identify subcommand
- **Phase 3**: lcms2 color transform (RGB→CMYK, row-by-row, verified with known pixels)
- **Phase 4**: libjpeg RGB JPEG decoder with ICC extraction
- **Phase 5**: CMYK JPEG encoder with channel-aware quantization (CMY vs K tables)
- **Phase 6**: Pipeline orchestration (decode→transform→encode, sRGB fallback)
- **Phase 7**: CLI subcommands (convert, transform, encode, identify)
- **Phase 8**: Testing & verification (all tests pass, E2E validated)

## CLI Usage
```bash
# Full conversion
./bin/rgbtocmyk convert -i input.jpg -o output.jpg --profile PSOcoated_v3.icc [--quality 85] [--cmy-reduction 15]

# Inspect image
./bin/rgbtocmyk identify image.jpg

# Color transform only (raw output)
./bin/rgbtocmyk transform -i input.jpg -o output.raw --profile PSOcoated_v3.icc

# Encode raw CMYK to JPEG
./bin/rgbtocmyk encode -i input.raw -o output.jpg --width W --height H [--icc profile.icc]
```
