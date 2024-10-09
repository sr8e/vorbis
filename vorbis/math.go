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

func lookup1Values(dimension uint16, entryLen uint32) int {
	// TODO verify floating point error
	return int(math.Floor(math.Pow(float64(entryLen), 1/float64(dimension))))
}

func inverseDecibels(index int) float64 {
	if index < 0 || index > 0xff {
		return 0
	}
	deciBel := -7 * float64(0xff-index) / float64(0x100)
	return math.Pow(10, deciBel)
}

// fls finds last set bit of integer.
func fls(x int) (i uint32) {
	if x <= 0 {
		return 0
	}
	for i = 0; x > 0; i++ {
		x >>= 1
	}
	return i
}

// lowNeighbor returns argmax { v[i] | i < x and v[i] < v[x] }.
func lowNeighbor(v []uint16, x int) int {
	var maxVal uint16 = 0
	maxIdx := -1
	for i := 0; i < x; i++ {
		if maxVal <= v[i] && v[i] < v[x] {
			maxVal = v[i]
			maxIdx = i
		}
	}
	return maxIdx
}

// highNeighbor returns argmin { v[i] | i < x and v[i] > v[x] }.
func highNeighbor(v []uint16, x int) int {
	var minVal uint16 = 1<<16 - 1
	minIdx := -1
	for i := 0; i < x; i++ {
		if v[x] < v[i] && v[i] <= minVal {
			minVal = v[i]
			minIdx = i
		}
	}
	return minIdx
}

func renderPoint(x0, x1 uint16, y0, y1 int, x uint16) int {
	return y0 + (y1-y0)*int(x-x0)/int(x1-x0)
}
