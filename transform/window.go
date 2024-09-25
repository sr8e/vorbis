package transform

import (
	"math"
)

func VorbisWindow(i, sampleBits int) float64 {
	N := 1 << sampleBits
	return math.Sin(math.Pi / 2 * math.Pow(math.Sin(math.Pi/float64(2*N)*float64(2*i+1)), 2))
}

func RectWindow(_, _ int) float64 {
	return math.Pow(2, -1/float64(2))
}
