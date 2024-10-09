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

func VorbisWindowVarWidth(leftBits, rightBits int) func(int, int) float64 {
	lq := 1 << (leftBits - 2)
	rq := 1 << (rightBits - 2)
	return func(i, windowBits int) float64 {
		Nq := 1 << (windowBits - 2)
		leftStart := Nq - lq
		leftEnd := Nq + lq
		rightStart := Nq*3 - rq
		rightEnd := Nq*3 + rq

		if leftStart <= i && i < leftEnd {
			return VorbisWindow(i-leftStart, leftBits)
		} else if leftEnd <= i && i < rightStart {
			return 1
		} else if rightStart <= i && i < rightEnd {
			return VorbisWindow(i-rightStart+rq*2, rightBits)
		}
		return 0
	}
}
