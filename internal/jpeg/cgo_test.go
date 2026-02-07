package jpeg

import "testing"

func TestLibjpegLinkage(t *testing.T) {
	ver := LibjpegVersion()
	if ver == 0 {
		t.Fatal("libjpeg version returned 0")
	}
	t.Logf("libjpeg version: %d", ver)
}
