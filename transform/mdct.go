package transform

import (
	"slices"
)

func mdctKernel(data []float64, sampleBits int) []float64 {
	N := 1 << sampleBits

	// fold input around boundary condition
	// mdct(a, b, c, d) -> dct4(-c_rev - d, a - b_rev)
	dctData := make([]float64, N/2, N/2)
	for i, _ := range dctData {
		if i < N / 4 {
			dctData[i] = - data[i + N*3/4] - data[N*3/4 - 1 - i]
		} else {
			dctData[i] = data[i - N/4] - data[N*3/4 - 1 - i]
		}
	}

	return DCT4(dctData, sampleBits - 1)
}

func MDCT(data []float64, sampleBits int, windowFunc func(int, int) float64) []float64 {
	N := 1 << sampleBits
	if len(data) != N {
		return nil
	}

	if windowFunc == nil {
		windowFunc = RectWindow
	}

	windowed := make([]float64, N, N)
	for i, v := range data {
		windowed[i] = v * windowFunc(i, sampleBits)
	}

	return mdctKernel(windowed, sampleBits)
}

func IMDCT(data []float64, sampleBits int, windowFunc func(int, int) float64) []float64 {
	N := 1 << sampleBits
	if len(data) != N / 2 { // coefficients are half the length of samples
		return nil
	}

	if windowFunc == nil {
		windowFunc = RectWindow
	}

	res := IDCT4(data, sampleBits - 1)
	
	// wrap result around boundary condition
	// res=(A, B) -> return (B, -B_rev, -A_rev, -A)
	cat := make([]float64, N, N)
	copy(cat, res[N/4:])
	for i, v := range res {
		res[i] = -v
	}
	copy(cat[N/4:], res)
	copy(cat[3*N/4:], res[:N/4])
	slices.Reverse(cat[N/4:3*N/4])

	for i, _ := range cat {
		cat[i] *= windowFunc(i, sampleBits)
	}
	return cat
}
