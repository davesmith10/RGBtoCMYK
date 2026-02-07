package color

import (
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
)

//go:embed srgb_v4.icc
var EmbeddedSRGB []byte

const (
	maxProfileSize = 4 * 1024 * 1024 // 4 MB
	acspMagic      = 0x61637370      // 'acsp'
)

// ProfileInfo contains metadata parsed from an ICC profile header.
type ProfileInfo struct {
	Size       uint32
	Version    string
	ColorSpace string // "RGB ", "CMYK", etc.
	PCS        string // "XYZ ", "Lab "
	Class      string // "mntr", "prtr", "scnr", etc.
}

// ParseProfileInfo reads ICC header metadata from raw profile bytes.
func ParseProfileInfo(data []byte) (*ProfileInfo, error) {
	if len(data) < 128 {
		return nil, errors.New("ICC profile too short (< 128 bytes)")
	}
	if uint32(len(data)) > maxProfileSize {
		return nil, fmt.Errorf("ICC profile too large (%d bytes, max %d)", len(data), maxProfileSize)
	}
	sig := binary.BigEndian.Uint32(data[36:40])
	if sig != acspMagic {
		return nil, fmt.Errorf("invalid ICC signature: 0x%08x (expected 0x%08x)", sig, acspMagic)
	}
	size := binary.BigEndian.Uint32(data[0:4])
	major := data[8]
	minor := data[9] >> 4
	bugfix := data[9] & 0x0f

	info := &ProfileInfo{
		Size:       size,
		Version:    fmt.Sprintf("%d.%d.%d", major, minor, bugfix),
		ColorSpace: string(data[16:20]),
		PCS:        string(data[20:24]),
		Class:      string(data[12:16]),
	}
	return info, nil
}

// LoadProfile reads an ICC profile from disk and validates it.
func LoadProfile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading ICC profile: %w", err)
	}
	if _, err := ParseProfileInfo(data); err != nil {
		return nil, fmt.Errorf("validating ICC profile %s: %w", path, err)
	}
	return data, nil
}

// ColorSpaceName returns a human-readable name for an ICC color space signature.
func ColorSpaceName(sig string) string {
	switch sig {
	case "RGB ":
		return "RGB"
	case "CMYK":
		return "CMYK"
	case "GRAY":
		return "Grayscale"
	case "Lab ":
		return "CIELAB"
	case "XYZ ":
		return "CIEXYZ"
	default:
		return sig
	}
}

// ProfileClassName returns a human-readable name for an ICC profile class.
func ProfileClassName(sig string) string {
	switch sig {
	case "mntr":
		return "Display"
	case "prtr":
		return "Output"
	case "scnr":
		return "Input"
	case "link":
		return "DeviceLink"
	case "spac":
		return "ColorSpace"
	case "abst":
		return "Abstract"
	case "nmcl":
		return "NamedColor"
	default:
		return sig
	}
}
