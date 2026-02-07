package jpeg

/*
#cgo pkg-config: libjpeg
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <jpeglib.h>
#include <setjmp.h>

typedef struct {
    struct jpeg_error_mgr pub;
    jmp_buf               jmpbuf;
    char                  msg[JMSG_LENGTH_MAX];
} decode_err_mgr;

static void decode_error_exit(j_common_ptr cinfo) {
    decode_err_mgr *e = (decode_err_mgr *)cinfo->err;
    (*(cinfo->err->format_message))(cinfo, e->msg);
    longjmp(e->jmpbuf, 1);
}

typedef struct {
    unsigned char *data;
    unsigned int   len;
} decode_marker;

typedef struct {
    int            width;
    int            height;
    int            num_components;
    unsigned char *pixels;       // RGB output
    unsigned long  pixels_size;
    int            num_markers;
    int            has_error;
    char           error_msg[256];
} decode_result;

static decode_result decode_rgb_jpeg(const unsigned char *buf, unsigned long buf_size,
                                      decode_marker *markers, int max_markers, int *marker_count) {
    decode_result res;
    memset(&res, 0, sizeof(res));
    *marker_count = 0;

    struct jpeg_decompress_struct cinfo;
    decode_err_mgr jerr;

    cinfo.err = jpeg_std_error(&jerr.pub);
    jerr.pub.error_exit = decode_error_exit;

    if (setjmp(jerr.jmpbuf)) {
        strncpy(res.error_msg, jerr.msg, sizeof(res.error_msg)-1);
        res.has_error = 1;
        jpeg_destroy_decompress(&cinfo);
        return res;
    }

    jpeg_create_decompress(&cinfo);
    jpeg_save_markers(&cinfo, JPEG_APP0+2, 0xFFFF); // APP2 for ICC
    jpeg_mem_src(&cinfo, (unsigned char *)buf, buf_size);
    jpeg_read_header(&cinfo, TRUE);

    // Force RGB output
    cinfo.out_color_space = JCS_RGB;

    jpeg_start_decompress(&cinfo);

    res.width = cinfo.output_width;
    res.height = cinfo.output_height;
    res.num_components = cinfo.output_components; // should be 3 for RGB

    res.pixels_size = (unsigned long)res.width * res.height * res.num_components;
    res.pixels = (unsigned char *)malloc(res.pixels_size);
    if (res.pixels == NULL) {
        strncpy(res.error_msg, "malloc failed for pixel buffer", sizeof(res.error_msg)-1);
        res.has_error = 1;
        jpeg_destroy_decompress(&cinfo);
        return res;
    }

    int row_stride = res.width * res.num_components;
    while (cinfo.output_scanline < cinfo.output_height) {
        unsigned char *row = res.pixels + cinfo.output_scanline * row_stride;
        jpeg_read_scanlines(&cinfo, &row, 1);
    }

    // Extract APP2 markers
    jpeg_saved_marker_ptr m = cinfo.marker_list;
    int count = 0;
    while (m != NULL && count < max_markers) {
        if (m->marker == (JPEG_APP0+2) && m->data_length > 0) {
            markers[count].data = (unsigned char *)malloc(m->data_length);
            if (markers[count].data != NULL) {
                memcpy(markers[count].data, m->data, m->data_length);
                markers[count].len = m->data_length;
                count++;
            }
        }
        m = m->next;
    }
    *marker_count = count;

    jpeg_finish_decompress(&cinfo);
    jpeg_destroy_decompress(&cinfo);
    return res;
}

static void free_decode_markers(decode_marker *markers, int count) {
    for (int i = 0; i < count; i++) {
        free(markers[i].data);
    }
}

static void free_decode_pixels(unsigned char *p) {
    free(p);
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// LibjpegVersion returns the JPEG library version.
func LibjpegVersion() int {
	return int(C.JPEG_LIB_VERSION)
}

// DecodedRGB holds the result of decoding an RGB JPEG.
type DecodedRGB struct {
	Width  int
	Height int
	Pixels []byte // RGB interleaved, len = Width * Height * 3
	ICC    []byte // extracted ICC profile, nil if absent
}

// DecodeRGB decodes a JPEG file from memory, outputting RGB pixels.
func DecodeRGB(data []byte) (*DecodedRGB, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("data too short for JPEG")
	}

	const maxMarkers = 256
	var cMarkers [maxMarkers]C.decode_marker
	var markerCount C.int

	res := C.decode_rgb_jpeg(
		(*C.uchar)(unsafe.Pointer(&data[0])),
		C.ulong(len(data)),
		&cMarkers[0],
		C.int(maxMarkers),
		&markerCount,
	)

	defer C.free_decode_markers(&cMarkers[0], markerCount)

	if res.has_error != 0 {
		return nil, fmt.Errorf("libjpeg decode: %s", C.GoString(&res.error_msg[0]))
	}

	defer C.free_decode_pixels(res.pixels)

	// Copy pixel data to Go-managed memory
	pixelSize := int(res.pixels_size)
	pixels := make([]byte, pixelSize)
	copy(pixels, unsafe.Slice((*byte)(unsafe.Pointer(res.pixels)), pixelSize))

	// Extract ICC
	var app2Markers [][]byte
	for i := 0; i < int(markerCount); i++ {
		m := cMarkers[i]
		goData := C.GoBytes(unsafe.Pointer(m.data), C.int(m.len))
		app2Markers = append(app2Markers, goData)
	}

	icc, err := ExtractICC(app2Markers)
	if err != nil {
		return nil, fmt.Errorf("extracting ICC: %w", err)
	}

	return &DecodedRGB{
		Width:  int(res.width),
		Height: int(res.height),
		Pixels: pixels,
		ICC:    icc,
	}, nil
}
