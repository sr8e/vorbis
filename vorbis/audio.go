package vorbis

import (
	"errors"

	"github.com/sr8e/vorbis/ogg"
	"github.com/sr8e/vorbis/transform"
)

func readAudioPacket(p *ogg.Packet, ident Identification, vs VorbisSetup) ([][]float64, error) {
	packetType, err := p.GetFlag()
	if err != nil {
		return nil, err
	}
	if packetType {
		return nil, errors.New("invalid packet type flag")
	}
	modeNum, err := p.GetUint(fls(len(vs.modeConfigs) - 1))
	if err != nil {
		return nil, err
	}
	mode := vs.modeConfigs[modeNum]

	var blockExp int
	var windowFunc func(int, int) float64

	if mode.blockFlag { // long window
		blockExp = int(ident.BlockExp[1])
		windowFlags, err := p.GetUint(2)
		if err != nil {
			return nil, err
		}
		leftExp := int(ident.BlockExp[windowFlags&1])
		rightExp := int(ident.BlockExp[(windowFlags>>1)&1])
		windowFunc = transform.VorbisWindowVarWidth(leftExp, rightExp)
	} else {
		blockExp = int(ident.BlockExp[0])
		windowFunc = transform.VorbisWindowVarWidth(blockExp, blockExp)
	}

	mapping := vs.mappingConfigs[mode.mapping]
	chNum := int(ident.Channels)

	// floor decode
	floors := make([][]int, chNum)
	noResidueFlags := make([]bool, chNum)
	for i := 0; i < chNum; i++ {
		floor := vs.floorConfigs[mapping.submaps[mapping.mapMux[i]].floor]

		floorShape, err := readFloorPacket(p, blockExp-1, floor, vs.codebooks)
		if err != nil && !errors.Is(err, ogg.ErrEndOfPacket) {
			return nil, err
		}
		if floorShape == nil { // unused
			noResidueFlags[i] = true
		}
		floors[i] = floorShape
	}
	// nonzero propagate
	for _, v := range mapping.polarMap {
		if noResidueFlags[v[0]] != noResidueFlags[v[1]] {
			noResidueFlags[v[0]] = false
			noResidueFlags[v[1]] = false
		}
	}

	// residue decode
	for i, submap := range mapping.submaps {
		noDecodeFlags := make([]bool, 0, chNum)
		for ch, submapIndex := range mapping.mapMux {
			if int(submapIndex) == i {
				noDecodeFlags = append(noDecodeFlags, noResidueFlags[ch])
			}
		}
		residue := vs.residueConfigs[submap.residue]

		resVectors, err := readResiduePacket(p, blockExp-1, residue, vs.codebooks, noDecodeFlags)
	}

	// TODO

	return nil, nil
}
