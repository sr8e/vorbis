package vorbis

import (
	"math"
)

func toFloat(v uint32) float64 {
	frac := v & 0x1fffff       // 21 bits
	exp := (v>>21)&0x3ff - 788 // 10 bits

	abs := float64(frac) * math.Pow(2, float64(exp))
	if (v>>31)&1 == 0 {
		return abs
	}
	return -abs
}

func lookup1Values(dimension int, entryLen int) int {
	// TODO verify floating point error
	return int(math.Floor(math.Pow(float64(entryLen), 1/float64(dimension))))
}
