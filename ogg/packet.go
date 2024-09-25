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
	size         uint32
	data         []byte
	cur          uint32
}

func (p *Packet) GetUint(n uint32) (uint32, error) {
	var i, v uint32
	for i = 0; i < n; {
		bytePos := (p.cur + i) / 8
		bitOfs := (p.cur + i) % 8

		if bytePos >= p.size {
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

func (p *Packet) GetFlag() (bool, error) {
	v, err := p.GetUint(1)
	return v == 1, err
}

func (p *Packet) GetUint8(n uint32) (uint8, error) {
	if n > 8 {
		return 0, errors.New("parameter n is too large")
	}
	v, err := p.GetUint(n)
	return uint8(v), err
}

func (p *Packet) GetUint16(n uint32) (uint16, error) {
	if n > 16 {
		return 0, errors.New("parameter n is too large")
	}
	v, err := p.GetUint(n)
	return uint16(v), err
}

func (p *Packet) GetUintAsInt(n uint32) (int, error) {
	if n >= 32 {
		return 0, errors.New("parameter n is too large")
	}
	v, err := p.GetUint(n)
	return int(v), err
}

func (p *Packet) GetBytes(nByte uint32) ([]byte, error) {
	arr := make([]byte, nByte, nByte)

	for i, _ := range arr {
		b, err := p.GetUint(8)
		if err != nil {
			return nil, err
		}
		arr[i] = byte(b)
	}
	return arr, nil
}

func (p *Packet) GetUintSerial(nList ...uint32) ([]uint32, error) {
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
