package color

import "testing"

func TestLcms2Linkage(t *testing.T) {
	ver := Lcms2Version()
	if ver == 0 {
		t.Fatal("lcms2 version returned 0")
	}
	t.Logf("lcms2 encoded CMM version: %d", ver)
}
