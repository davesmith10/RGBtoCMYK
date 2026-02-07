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
} err_mgr;

static void error_exit_handler(j_common_ptr cinfo) {
    err_mgr *e = (err_mgr *)cinfo->err;
    (*(cinfo->err->format_message))(cinfo, e->msg);
    longjmp(e->jmpbuf, 1);
}

typedef struct {
    int width;
    int height;
    int num_components;
    int color_space;    // J_COLOR_SPACE enum value
    int num_markers;
    int has_error;
    char error_msg[256];
} jpeg_info_result;

// info_marker holds extracted APP2 marker data
typedef struct {
    unsigned char *data;
    unsigned int  len;
} info_marker;

static jpeg_info_result get_jpeg_info(const unsigned char *buf, unsigned long buf_size,
                                       info_marker *markers, int max_markers, int *marker_count) {
    jpeg_info_result res;
    memset(&res, 0, sizeof(res));
    *marker_count = 0;

    struct jpeg_decompress_struct cinfo;
    err_mgr jerr;

    cinfo.err = jpeg_std_error(&jerr.pub);
    jerr.pub.error_exit = error_exit_handler;

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

    res.width = cinfo.image_width;
    res.height = cinfo.image_height;
    res.num_components = cinfo.num_components;
    res.color_space = cinfo.jpeg_color_space;

    // extract APP2 markers
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

    jpeg_destroy_decompress(&cinfo);
    return res;
}

static void free_info_markers(info_marker *markers, int count) {
    for (int i = 0; i < count; i++) {
        free(markers[i].data);
    }
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// ColorSpaceName returns a string for libjpeg's J_COLOR_SPACE.
func colorSpaceName(cs int) string {
	switch cs {
	case 0:
		return "Unknown"
	case 1:
		return "Grayscale"
	case 2:
		return "RGB"
	case 3:
		return "YCbCr"
	case 4:
		return "CMYK"
	case 5:
		return "YCCK"
	default:
		return fmt.Sprintf("J_COLOR_SPACE(%d)", cs)
	}
}

// ImageInfo contains metadata about a JPEG file.
type ImageInfo struct {
	Width         int
	Height        int
	NumComponents int
	ColorSpace    string
	ICC           []byte // extracted ICC profile, nil if absent
}

// GetInfo reads JPEG metadata and extracts any ICC profile without fully decoding the image.
func GetInfo(data []byte) (*ImageInfo, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("data too short for JPEG")
	}

	const maxMarkers = 256
	var cMarkers [maxMarkers]C.info_marker
	var markerCount C.int

	res := C.get_jpeg_info(
		(*C.uchar)(unsafe.Pointer(&data[0])),
		C.ulong(len(data)),
		&cMarkers[0],
		C.int(maxMarkers),
		&markerCount,
	)

	defer C.free_info_markers(&cMarkers[0], markerCount)

	if res.has_error != 0 {
		return nil, fmt.Errorf("libjpeg: %s", C.GoString(&res.error_msg[0]))
	}

	// Collect APP2 marker data into Go slices
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

	return &ImageInfo{
		Width:         int(res.width),
		Height:        int(res.height),
		NumComponents: int(res.num_components),
		ColorSpace:    colorSpaceName(int(res.color_space)),
		ICC:           icc,
	}, nil
}
