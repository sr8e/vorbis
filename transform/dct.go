package transform

import (
	"math"
	"math/cmplx"
)

func DCT4(data []float64, bits int) []float64 {
	N := 1 << bits
	if len(data) != N {
		return nil
	}

	// pack the data into length of N/2 complex array
	cmpData := make([]complex128, N / 2, N / 2)
	for i, _ := range cmpData {
		// pre-rotation
		cmpData[i] = complex(data[2 * i], data[N - 1 - 2 * i]) * cmplx.Rect(1, - math.Pi * float64(i) / float64(N))
	}

	cmpCoef := FFT(cmpData, bits - 1)

	// unpack the coefficient
	res := make([]float64, N, N)
	for i, v := range cmpCoef {
		// post-rotation
		post := v * cmplx.Rect(1, -math.Pi * float64(4 * i + 1) / float64(4 * N))
		res[2 * i] = real(post)
		res[N - 1 - 2 * i] = -imag(post)
	}

	return res
}

func IDCT4(data []float64, bits int) []float64 {
	f := DCT4(data, bits)
	N := 1 << bits
	for i, _ := range f {
		f[i] /= float64(N) / 2
	}
	return f
}
