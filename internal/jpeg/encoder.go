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
} encode_err_mgr;

static void encode_error_exit(j_common_ptr cinfo) {
    encode_err_mgr *e = (encode_err_mgr *)cinfo->err;
    (*(cinfo->err->format_message))(cinfo, e->msg);
    longjmp(e->jmpbuf, 1);
}

typedef struct {
    unsigned char *buf;
    unsigned long  size;
    int            has_error;
    char           error_msg[256];
} encode_result;

#define ICC_TAG "ICC_PROFILE"
#define ICC_TAG_LEN 12
#define ICC_HEADER_LEN 14
#define MAX_CHUNK_DATA (65535 - 2 - ICC_HEADER_LEN)

// write_icc_markers chunks and writes an ICC profile as APP2 markers.
static void write_icc_markers(j_compress_ptr cinfo, const unsigned char *icc, unsigned long icc_len) {
    int num_chunks = (icc_len + MAX_CHUNK_DATA - 1) / MAX_CHUNK_DATA;
    if (num_chunks > 255) num_chunks = 255;

    for (int i = 0; i < num_chunks; i++) {
        unsigned long offset = (unsigned long)i * MAX_CHUNK_DATA;
        unsigned long chunk_data_len = icc_len - offset;
        if (chunk_data_len > MAX_CHUNK_DATA) chunk_data_len = MAX_CHUNK_DATA;

        unsigned long marker_len = ICC_HEADER_LEN + chunk_data_len;
        unsigned char *marker = (unsigned char *)malloc(marker_len);
        if (marker == NULL) return;

        // "ICC_PROFILE\0" = 12 bytes (string literal includes null terminator)
        memcpy(marker, ICC_TAG "\0", ICC_TAG_LEN);
        marker[12] = (unsigned char)(i + 1);         // sequence number (1-based)
        marker[13] = (unsigned char)num_chunks;      // total count
        memcpy(marker + ICC_HEADER_LEN, icc + offset, chunk_data_len);

        jpeg_write_marker(cinfo, JPEG_APP0 + 2, marker, (unsigned int)marker_len);
        free(marker);
    }
}

// encode_cmyk_jpeg encodes CMYK pixels to JPEG with custom quantization tables.
static encode_result encode_cmyk_jpeg(
    const unsigned char *pixels, int width, int height,
    const unsigned int *cmy_qtable, const unsigned int *k_qtable,
    const unsigned char *icc, unsigned long icc_len
) {
    encode_result res;
    memset(&res, 0, sizeof(res));

    struct jpeg_compress_struct cinfo;
    encode_err_mgr jerr;

    cinfo.err = jpeg_std_error(&jerr.pub);
    jerr.pub.error_exit = encode_error_exit;

    if (setjmp(jerr.jmpbuf)) {
        strncpy(res.error_msg, jerr.msg, sizeof(res.error_msg)-1);
        res.has_error = 1;
        jpeg_destroy_compress(&cinfo);
        return res;
    }

    jpeg_create_compress(&cinfo);
    jpeg_mem_dest(&cinfo, &res.buf, &res.size);

    cinfo.image_width = width;
    cinfo.image_height = height;
    cinfo.input_components = 4;
    cinfo.in_color_space = JCS_CMYK;

    jpeg_set_defaults(&cinfo);
    cinfo.optimize_coding = TRUE;

    // Set all sampling factors to 1x1 (no subsampling for CMYK)
    for (int i = 0; i < 4; i++) {
        cinfo.comp_info[i].h_samp_factor = 1;
        cinfo.comp_info[i].v_samp_factor = 1;
    }

    // Set quantization tables directly (pre-scaled values).
    if (cinfo.quant_tbl_ptrs[0] == NULL)
        cinfo.quant_tbl_ptrs[0] = jpeg_alloc_quant_table((j_common_ptr)&cinfo);
    if (cinfo.quant_tbl_ptrs[1] == NULL)
        cinfo.quant_tbl_ptrs[1] = jpeg_alloc_quant_table((j_common_ptr)&cinfo);

    for (int i = 0; i < 64; i++) {
        cinfo.quant_tbl_ptrs[0]->quantval[i] = (UINT16)cmy_qtable[i];
        cinfo.quant_tbl_ptrs[1]->quantval[i] = (UINT16)k_qtable[i];
    }

    cinfo.comp_info[0].quant_tbl_no = 0;
    cinfo.comp_info[1].quant_tbl_no = 0;
    cinfo.comp_info[2].quant_tbl_no = 0;
    cinfo.comp_info[3].quant_tbl_no = 1;

    jpeg_start_compress(&cinfo, TRUE);

    // Write ICC profile as APP2 marker chunks
    if (icc != NULL && icc_len > 0) {
        write_icc_markers(&cinfo, icc, icc_len);
    }

    // Write scanlines
    int row_stride = width * 4;
    while (cinfo.next_scanline < cinfo.image_height) {
        const unsigned char *row = pixels + cinfo.next_scanline * row_stride;
        jpeg_write_scanlines(&cinfo, (JSAMPARRAY)&row, 1);
    }

    jpeg_finish_compress(&cinfo);
    jpeg_destroy_compress(&cinfo);
    return res;
}

static void free_encode_buf(unsigned char *buf) {
    free(buf);
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// EncoderOptions controls CMYK JPEG encoding.
type EncoderOptions struct {
	Quality      int // 1-100, default 85
	CMYReduction int // quality reduction for CMY vs K, default 15
}

// EncodeCMYK encodes CMYK pixel data to JPEG format with channel-aware quantization.
// pixels must be width*height*4 bytes (CMYK interleaved).
// iccProfile is the ICC profile to embed (can be nil).
func EncodeCMYK(pixels []byte, width, height int, iccProfile []byte, opts EncoderOptions) ([]byte, error) {
	expectedSize := width * height * 4
	if len(pixels) != expectedSize {
		return nil, fmt.Errorf("expected %d CMYK bytes, got %d", expectedSize, len(pixels))
	}

	if opts.Quality == 0 {
		opts.Quality = 85
	}
	if opts.CMYReduction == 0 {
		opts.CMYReduction = 15
	}

	cmyTable, kTable := GenerateQuantTables(opts.Quality, opts.CMYReduction)

	var cmyQtableC [64]C.uint
	var kQtableC [64]C.uint
	for i := 0; i < 64; i++ {
		cmyQtableC[i] = C.uint(cmyTable[i])
		kQtableC[i] = C.uint(kTable[i])
	}

	var iccPtr *C.uchar
	var iccLen C.ulong
	if len(iccProfile) > 0 {
		iccPtr = (*C.uchar)(unsafe.Pointer(&iccProfile[0]))
		iccLen = C.ulong(len(iccProfile))
	}

	res := C.encode_cmyk_jpeg(
		(*C.uchar)(unsafe.Pointer(&pixels[0])),
		C.int(width), C.int(height),
		&cmyQtableC[0], &kQtableC[0],
		iccPtr, iccLen,
	)

	if res.has_error != 0 {
		return nil, fmt.Errorf("libjpeg encode: %s", C.GoString(&res.error_msg[0]))
	}

	defer C.free_encode_buf(res.buf)

	output := C.GoBytes(unsafe.Pointer(res.buf), C.int(res.size))
	return output, nil
}
