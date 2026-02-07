package color

/*
#cgo pkg-config: lcms2
#include <lcms2.h>
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"runtime"
	"unsafe"
)

// Lcms2Version returns the encoded CMM version from lcms2.
func Lcms2Version() int {
	return int(C.cmsGetEncodedCMMversion())
}

// Intent constants matching lcms2.
const (
	IntentPerceptual           = 0
	IntentRelativeColorimetric = 1
	IntentSaturation           = 2
	IntentAbsoluteColorimetric = 3
)

// ParseIntent converts a string intent name to an lcms2 intent constant.
func ParseIntent(s string) (int, error) {
	switch s {
	case "perceptual":
		return IntentPerceptual, nil
	case "relative":
		return IntentRelativeColorimetric, nil
	case "saturation":
		return IntentSaturation, nil
	case "absolute":
		return IntentAbsoluteColorimetric, nil
	default:
		return 0, fmt.Errorf("unknown rendering intent: %q", s)
	}
}

// Transform performs ICC color transformations using lcms2.
type Transform struct {
	hSrc       C.cmsHPROFILE
	hDst       C.cmsHPROFILE
	hTransform C.cmsHTRANSFORM
}

// NewTransform creates an RGB→CMYK color transform from raw ICC profile data.
func NewTransform(srcICC, dstICC []byte, intent int) (*Transform, error) {
	hSrc := C.cmsOpenProfileFromMem(unsafe.Pointer(&srcICC[0]), C.cmsUInt32Number(len(srcICC)))
	if hSrc == nil {
		return nil, fmt.Errorf("lcms2: failed to open source profile")
	}

	hDst := C.cmsOpenProfileFromMem(unsafe.Pointer(&dstICC[0]), C.cmsUInt32Number(len(dstICC)))
	if hDst == nil {
		C.cmsCloseProfile(hSrc)
		return nil, fmt.Errorf("lcms2: failed to open destination profile")
	}

	hTransform := C.cmsCreateTransform(
		hSrc, C.TYPE_RGB_8,
		hDst, C.TYPE_CMYK_8,
		C.cmsUInt32Number(intent),
		C.cmsFLAGS_NOCACHE,
	)
	if hTransform == nil {
		C.cmsCloseProfile(hDst)
		C.cmsCloseProfile(hSrc)
		return nil, fmt.Errorf("lcms2: failed to create transform")
	}

	t := &Transform{
		hSrc:       hSrc,
		hDst:       hDst,
		hTransform: hTransform,
	}
	runtime.SetFinalizer(t, (*Transform).Close)
	return t, nil
}

// TransformPixels converts RGB pixels to CMYK in-place row by row.
// src must be width*height*3 bytes (RGB), returns width*height*4 bytes (CMYK).
func (t *Transform) TransformPixels(src []byte, width, height int) ([]byte, error) {
	expectedSrc := width * height * 3
	if len(src) != expectedSrc {
		return nil, fmt.Errorf("expected %d RGB bytes, got %d", expectedSrc, len(src))
	}

	dst := make([]byte, width*height*4)

	for y := 0; y < height; y++ {
		srcOff := y * width * 3
		dstOff := y * width * 4
		C.cmsDoTransform(
			t.hTransform,
			unsafe.Pointer(&src[srcOff]),
			unsafe.Pointer(&dst[dstOff]),
			C.cmsUInt32Number(width),
		)
	}

	return dst, nil
}

// Close releases lcms2 resources.
func (t *Transform) Close() {
	if t.hTransform != nil {
		C.cmsDeleteTransform(t.hTransform)
		t.hTransform = nil
	}
	if t.hDst != nil {
		C.cmsCloseProfile(t.hDst)
		t.hDst = nil
	}
	if t.hSrc != nil {
		C.cmsCloseProfile(t.hSrc)
		t.hSrc = nil
	}
}
