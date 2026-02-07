package ir

// CMYKImage is the intermediate representation passed between the color
// transform and the JPEG encoder. Pixels are stored as interleaved C,M,Y,K
// bytes (4 bytes per pixel, row-major order).
type CMYKImage struct {
	Width  int
	Height int
	Pixels []byte // len = Width * Height * 4
	ICC    []byte // CMYK ICC profile to embed in output
}
