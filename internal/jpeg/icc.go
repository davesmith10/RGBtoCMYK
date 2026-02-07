package jpeg

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
)

const (
	iccMarkerTag     = "ICC_PROFILE\x00"
	maxChunkDataSize = 65519 // max APP2 payload minus 2-byte length = 65535 - 2 - 14 (tag + seq + count)
)

// ExtractICC reassembles an ICC profile from APP2 marker segments.
// markers is a slice of raw APP2 marker payloads (excluding the APP2 marker bytes themselves).
func ExtractICC(markers [][]byte) ([]byte, error) {
	type chunk struct {
		seq  int
		data []byte
	}
	var chunks []chunk
	expectedCount := 0

	for _, m := range markers {
		if len(m) < 14 {
			continue
		}
		if string(m[:12]) != iccMarkerTag {
			continue
		}
		seq := int(m[12])
		count := int(m[13])
		if seq == 0 || seq > count {
			return nil, fmt.Errorf("invalid ICC chunk sequence %d/%d", seq, count)
		}
		if expectedCount == 0 {
			expectedCount = count
		} else if count != expectedCount {
			return nil, fmt.Errorf("inconsistent ICC chunk count: %d vs %d", count, expectedCount)
		}
		chunks = append(chunks, chunk{seq: seq, data: m[14:]})
	}

	if len(chunks) == 0 {
		return nil, nil // no ICC profile present
	}
	if len(chunks) != expectedCount {
		return nil, fmt.Errorf("expected %d ICC chunks, found %d", expectedCount, len(chunks))
	}

	sort.Slice(chunks, func(i, j int) bool { return chunks[i].seq < chunks[j].seq })

	var buf bytes.Buffer
	for _, c := range chunks {
		buf.Write(c.data)
	}
	return buf.Bytes(), nil
}

// ChunkICC splits an ICC profile into APP2-ready marker payloads.
// Each returned []byte is a complete APP2 marker payload (tag + seq/count + profile data).
func ChunkICC(profile []byte) ([][]byte, error) {
	if len(profile) == 0 {
		return nil, errors.New("empty ICC profile")
	}

	numChunks := (len(profile) + maxChunkDataSize - 1) / maxChunkDataSize
	if numChunks > 255 {
		return nil, fmt.Errorf("ICC profile too large: needs %d chunks (max 255)", numChunks)
	}

	chunks := make([][]byte, 0, numChunks)
	for i := 0; i < numChunks; i++ {
		start := i * maxChunkDataSize
		end := start + maxChunkDataSize
		if end > len(profile) {
			end = len(profile)
		}
		// header: "ICC_PROFILE\0" + seq (1-based) + count
		header := make([]byte, 14)
		copy(header, iccMarkerTag)
		header[12] = byte(i + 1)
		header[13] = byte(numChunks)

		chunk := make([]byte, 0, 14+end-start)
		chunk = append(chunk, header...)
		chunk = append(chunk, profile[start:end]...)
		chunks = append(chunks, chunk)
	}
	return chunks, nil
}
