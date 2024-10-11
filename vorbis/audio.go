package vorbis

import (
	"errors"
	"slices"

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
	floors := make([][]float64, chNum)
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
	residues := make([][]float64, chNum)
	for i, submap := range mapping.submaps {
		noDecodeFlags := make([]bool, 0, chNum)
		chMap := make([]int, 0, chNum)
		for ch, submapIndex := range mapping.mapMux {
			if int(submapIndex) == i {
				noDecodeFlags = append(noDecodeFlags, noResidueFlags[ch])
				chMap = append(chMap, ch)
			}
		}
		residue := vs.residueConfigs[submap.residue]

		resVectors, err := readResiduePacket(p, blockExp-1, residue, vs.codebooks, noDecodeFlags)
		if err != nil {
			return nil, err
		}
		for j, v := range resVectors {
			residues[chMap[j]] = v
		}
	}

	// residue decoupling
	for _, v := range slices.Backward(mapping.polarMap) {
		mag := residues[v[0]]
		amp := residues[v[1]]

		for i := range mag {
			m := mag[i]
			a := amp[i]
			if m > 0 {
				if a > 0 {
					amp[i] = m - a
				} else {
					mag[i] = m + a
					amp[i] = m
				}
			} else {
				if a > 0 {
					amp[i] = m + a
				} else {
					mag[i] = m - a
					amp[i] = m
				}
			}
		}
	}

	// dot product
	for ch, fvec := range floors {
		for j := range fvec {
			floors[ch][j] *= residues[ch][j]
		}
	}

	// inverse MDCT
	trans := make([][]float64, chNum)
	for i, v := range floors {
		trans[i] = transform.IMDCT(v, blockExp, windowFunc)
	}
	return trans, nil

}
