package vorbis

import (
	"errors"
	"fmt"

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
	for i := 0; i < chNum; i++ {
		var submapIndex uint8
		// case submaps == 1 is not mentioned in spec?
		if len(mapping.submaps) > 1 {
			submapIndex = mapping.mapMux[i]
		}
		floor := vs.floorConfigs[mapping.submaps[submapIndex].floor]
		if floor.floorType == 0 {
			// TODO
		} else if floor.floorType == 1 {
			floorShape, err := readFloor1Packet(p, blockExp, *floor.config1, vs.codebooks)
			if err != nil {
				if errors.Is(err, ogg.ErrEndOfPacket) {
					// treat as unused
					goto overlap
				}
				return nil, err
			}
			if floorShape == nil { // unused
				goto overlap
			}
			fmt.Printf("%v", floorShape)
		}
	}

	// TODO

overlap:
	return nil, nil
}
