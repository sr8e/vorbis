package vorbis

import (
	"github.com/sr8e/vorbis/ogg"
)

type VorbisDecoder struct {
	Packets        []ogg.Packet
	Identification Identification
	setup          VorbisSetup
	isReady        bool
}

type Identification struct {
	Channels   byte
	SampleRate uint32
	BitRate    [3]int32
	BlockExp   [2]uint8
}

type VorbisSetup struct {
	codebooks      []codebook
	floorConfigs   []floorConfig
	residueConfigs []residueConfig
	mappingConfigs []mappingConfig
	modeConfigs    []modeConfig
}

type modeConfig struct {
	blockFlag bool
	mapping   uint8
}

func (vd *VorbisDecoder) ReadHeaders() error {
	ident, err := readIdentification(&vd.Packets[0])
	if err != nil {
		return err
	}
	vd.Identification = ident

	// TODO read comment header

	vs, err := readSetup(&vd.Packets[2], ident)
	if err != nil {
		return err
	}
	vd.setup = vs
	vd.isReady = true

	return nil
}
