package transform

import (
	"math"
	"math/cmplx"
)

func rotationFactor(bits int, inverse bool) []complex128 {
	N := 1 << bits
	W := make([]complex128, N, N)

	c := -2.0
	if inverse {
		c = 2.0
	}

	for i := 0; i < N; i++ {
		W[i] = cmplx.Rect(1, c*float64(i)*math.Pi/float64(N))
	}
	return W
}

func bitReverse(bits int) []int {
	N := 1 << bits
	seq := make([]int, N, N)
	for i := 0; i < bits; i++ {
		for j := 0; j < (1 << i); j++ {
			seq[j+(1<<i)] = seq[j] + (1 << (bits - i - 1))
		}
	}
	return seq
}

func fftKernel(data []complex128, bits int, inverse bool) []complex128 {
	N := 1 << bits

	if len(data) != N {
		return nil
	}

	rev := bitReverse(bits)
	W := rotationFactor(bits, inverse)

	prev := make([]complex128, N, N)
	for i := 0; i < N; i++ {
		prev[i] = data[rev[i]]
	}

	for i := 0; i < bits; i++ {
		next := make([]complex128, N, N)
		for j := 0; j < N; j++ {
			ofs := 1 << i
			if (j>>i)%2 == 0 {
				next[j] += prev[j]
				next[j+ofs] += prev[j]
			} else {
				rotIndex := (j << (bits - i - 1)) % N
				next[j-ofs] += prev[j] * W[rotIndex-N/2]
				next[j] += prev[j] * W[rotIndex]
			}
		}
		prev = next
	}
	return prev
}

func FFT(data []complex128, bits int) []complex128 {
	return fftKernel(data, bits, false)
}

func IFFT(data []complex128, bits int) []complex128 {
	f := fftKernel(data, bits, true)
	N := 1 << bits
	for i, _ := range f {
		f[i] /= complex(float64(N), 0)
	}
	return f
}
