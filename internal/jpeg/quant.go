package jpeg

// Standard JPEG luminance quantization table (from JPEG spec, Annex K).
var stdLuminanceQuant = [64]int{
	16, 11, 10, 16, 24, 40, 51, 61,
	12, 12, 14, 19, 26, 58, 60, 55,
	14, 13, 16, 24, 40, 57, 69, 56,
	14, 17, 22, 29, 51, 87, 80, 62,
	18, 22, 37, 56, 68, 109, 103, 77,
	24, 35, 55, 64, 81, 104, 113, 92,
	49, 64, 78, 87, 103, 121, 120, 101,
	72, 92, 95, 98, 112, 100, 103, 99,
}

// ScaleQuantTable scales a base quantization table by a quality factor (1-100)
// using the standard IJG formula.
func ScaleQuantTable(base [64]int, quality int) [64]uint16 {
	if quality <= 0 {
		quality = 1
	}
	if quality > 100 {
		quality = 100
	}

	var scale int
	if quality < 50 {
		scale = 5000 / quality
	} else {
		scale = 200 - quality*2
	}

	var table [64]uint16
	for i := 0; i < 64; i++ {
		val := (base[i]*scale + 50) / 100
		if val < 1 {
			val = 1
		}
		if val > 255 {
			val = 255
		}
		table[i] = uint16(val)
	}
	return table
}

// GenerateQuantTables returns two quantization tables:
// table[0] = CMY channels (scaled at quality - cmyReduction)
// table[1] = K channel (scaled at quality)
func GenerateQuantTables(quality, cmyReduction int) (cmy [64]uint16, k [64]uint16) {
	cmyQuality := quality - cmyReduction
	if cmyQuality < 1 {
		cmyQuality = 1
	}
	cmy = ScaleQuantTable(stdLuminanceQuant, cmyQuality)
	k = ScaleQuantTable(stdLuminanceQuant, quality)
	return
}
