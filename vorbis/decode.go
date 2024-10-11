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

func (vd *VorbisDecoder) DecodeAll() ([][]float64, error) {
	if !vd.isReady {
		err := vd.ReadHeaders()
		if err != nil {
			return nil, err
		}
	}

	samples := make([][]float64, vd.Identification.Channels)
	buf := make([][]float64, vd.Identification.Channels)
	for ch := range samples {
		samples[ch] = make([]float64, 0)
	}

	for _, packet := range vd.Packets[3:] {
		content, err := readAudioPacket(&packet, vd.Identification, vd.setup)
		if err != nil {
			return nil, err
		}

		for ch, v := range content {
			curN := len(v) / 2
			preN := len(buf[ch])
			if len(buf) == 0 { // first packet
				buf[ch] = v[curN:]
				continue
			}

			left := (preN - curN) / 2
			right := (preN + curN) / 2
			overlap := make([]float64, right)
			for i := 0; i < right; i++ {
				if i < left {
					overlap[i] = buf[ch][i]
				} else if i < preN {
					overlap[i] = buf[ch][i] + v[i-left]
				} else {
					overlap[i] = v[i-left]
				}
			}
			// TODO it seems we need to clip the value into [-1, 1]
			samples[ch] = append(samples[ch], overlap...)
			buf[ch] = v[curN:]
		}
	}

	return samples, nil
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
