package vorbis

import (
	"errors"
	"fmt"

	"github.com/sr8e/vorbis/ogg"
)

func readCommonHeader(p *ogg.Packet, headerOrder uint8) error {
	packetType, err := p.GetUint8(8)
	if err != nil {
		return err
	}
	if packetType&1 != 1 || packetType>>1 != headerOrder {
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

	fields, err := p.GetUintSerial(32, 8, 32, 32, 32, 32, 4, 4, 1)
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

func readSetup(p *ogg.Packet, ident Identification) (_ VorbisSetup, err error) {
	err = readCommonHeader(p, 2)
	if err != nil {
		return
	}
	cbLen, err := p.GetUint(8)
	if err != nil {
		return
	}
	codebooks := make([]codebook, cbLen+1)
	for i, _ := range codebooks {
		codebooks[i], err = readCodebook(p)
		if err != nil {
			return
		}
	}

	// placeholder, discard
	tdt, err := p.GetUint(6)
	if err != nil {
		return
	}
	for i := 0; i < int(tdt)+1; i++ {
		var v uint32
		v, err = p.GetUint(16)
		if err != nil {
			return
		}
		if v != 0 {
			err = fmt.Errorf("non-zero value in time domain transform field: %d, %x", tdt, v)
			return
		}
	}

	floorConfigs, err := readFloorConfig(p)
	if err != nil {
		return
	}

	residueConfigs, err := readResidueConfig(p)
	if err != nil {
		return
	}

	mappingConfigs, err := readMappingConfigs(p, ident)
	if err != nil {
		return
	}

	modeConfigs, err := readModeConfigs(p)
	if err != nil {
		return
	}

	framingBit, err := p.GetFlag()
	if err != nil {
		return
	}
	if !framingBit {
		err = errors.New("framing bit not set")
		return
	}

	return VorbisSetup{
		codebooks:      codebooks,
		floorConfigs:   floorConfigs,
		residueConfigs: residueConfigs,
		mappingConfigs: mappingConfigs,
		modeConfigs:    modeConfigs,
	}, nil
}

func readModeConfigs(p *ogg.Packet) ([]modeConfig, error) {
	modeLen, err := p.GetUint(6)
	if err != nil {
		return nil, err
	}
	modeLen += 1

	modes := make([]modeConfig, modeLen, modeLen)
	for i, _ := range modes {
		fields, err := p.GetUintSerial(1, 16, 16, 8)
		if err != nil {
			return nil, err
		}
		if fields[1] != 0 || fields[2] != 0 {
			return nil, errors.New("invalid non-zero value in mode config")
		}
		modes[i] = modeConfig{
			blockFlag: fields[0] == 1,
			mapping:   uint8(fields[3]),
		}
	}
	return modes, nil
}
