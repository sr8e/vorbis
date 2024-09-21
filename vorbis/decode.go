package vorbis

import (
	"errors"
	"fmt"
	"github.com/sr8e/vorbis/ogg"
)

type Identification struct {
	Channels   byte
	SampleRate uint32
	BitRate    [3]int32
	BlockExp   [2]byte
}

type VorbisSetup struct {
	codebooks []codebook
}

type VorbisDecoder struct {
	Packets        []ogg.Packet
	Identification Identification
	setup          VorbisSetup
}

func (vd *VorbisDecoder) ReadHeaders() error {
	ident, err := readIdentification(&vd.Packets[0])
	if err != nil {
		return err
	}
	vd.Identification = ident

	vs, err := readSetup(&vd.Packets[2])
	if err != nil {
		return err
	}
	vd.setup = vs
	return nil
}

func readCommonHeader(p *ogg.Packet, headerOrder int) error {
	packetType, err := p.GetUint(8)
	if err != nil {
		return err
	}
	if packetType&1 != 1 || packetType>>1 != uint32(headerOrder) {
		return fmt.Errorf("invalid header type %x at packet %d", packetType, headerOrder)
	}
	pattern, err := p.GetBytes(6)
	if err != nil {
		return err
	}
	if string(pattern) != "vorbis" {
		return errors.New("invalid header packet")
	}
	return nil
}

func readIdentification(p *ogg.Packet) (_ Identification, err error) {
	err = readCommonHeader(p, 0)
	if err != nil {
		return
	}

	fields, err := p.GetUintSerial([]int{32, 8, 32, 32, 32, 32, 4, 4, 1})
	if err != nil {
		return
	}

	if fields[0] != 0 {
		err = errors.New("incompatible vorbis version")
		return
	}
	var bitRate [3]int32
	for i, v := range fields[3:6] {
		bitRate[i] = int32(v)
	}
	var blockExp [2]byte
	for i, v := range fields[6:8] {
		if v < 6 || 12 < v {
			err = fmt.Errorf("invalid block size: %d", v)
			return
		}
		blockExp[i] = byte(v)
	}
	if fields[8] != 1 {
		err = errors.New("framing bit not set")
		return
	}

	return Identification{
		Channels:   byte(fields[1]),
		SampleRate: fields[2],
		BitRate:    bitRate,
		BlockExp:   blockExp,
	}, nil
}

func readSetup(p *ogg.Packet) (_ VorbisSetup, err error) {
	err = readCommonHeader(p, 2)
	if err != nil {
		return
	}
	cbLen, err := p.GetUint(8)
	if err != nil {
		return
	}
	codebooks := make([]codebook, cbLen)
	for i := 0; i < int(cbLen); i++ {
		codebooks[i], err = readCodebook(p)
		if err != nil {
			return
		}
	}

	// TODO other field

	return VorbisSetup{
		codebooks: codebooks,
	}, nil
}
