package ogg

import (
	"errors"
)

var mask = [9]byte{
	0b00000000, 0b00000001, 0b00000011, 0b00000111,
	0b00001111, 0b00011111, 0b00111111, 0b01111111,
	0b11111111,
}

type Packet struct {
	continueFlag byte // 1 for following, 2 for followed
	size         int
	data         []byte
	cur          int
}

func (p *Packet) GetUint(n int) (uint32, error) {
	var v uint32
	for i := 0; i < n; {
		bytePos := (p.cur + i) / 8
		bitOfs := (p.cur + i) % 8

		if bytePos >= len(p.data) {
			return 0, errors.New("end-of-packet condition")
		}

		b := p.data[bytePos] >> bitOfs
		maskLen := min(8-bitOfs, n-i)
		v += uint32(b&mask[maskLen]) << i

		i += maskLen
	}
	p.cur += n
	return v, nil
}

func (p *Packet) GetBytes(nByte int) ([]byte, error) {
	arr := make([]byte, nByte)

	for i := 0; i < nByte; i++ {
		b, err := p.GetUint(8)
		if err != nil {
			return nil, err
		}
		arr[i] = byte(b)
	}
	return arr, nil
}

func (p *Packet) GetUintSerial(nList []int) ([]uint32, error) {
	vals := make([]uint32, len(nList))
	for i, n := range nList {
		v, err := p.GetUint(n)
		if err != nil {
			return nil, err
		}
		vals[i] = v
	}
	return vals, nil
}
