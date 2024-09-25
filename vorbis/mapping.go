package vorbis

import (
	"errors"
	"github.com/sr8e/vorbis/ogg"
)

type mappingConfig struct {
	polarMap [][]uint32
	mapMux   []uint8
	submaps  []mappingSubmap
}

type mappingSubmap struct {
	floor   uint32
	residue uint32
}

func readMappingConfigs(p *ogg.Packet, ident Identification) ([]mappingConfig, error) {
	mapLen, err := p.GetUint(6)
	if err != nil {
		return nil, err
	}
	mapLen += 1

	maps := make([]mappingConfig, mapLen, mapLen)
	for i, _ := range maps {
		mapType, err := p.GetUint(16)
		if err != nil {
			return nil, err
		}
		if mapType != 0 {
			return nil, errors.New("invalid maptype")
		}
		submapFlag, err := p.GetFlag()
		if err != nil {
			return nil, err
		}
		var submapLen uint8 = 1
		if submapFlag {
			submapLen, err = p.GetUint8(4)
			if err != nil {
				return nil, err
			}
		}
		couplingFlag, err := p.GetFlag()
		if err != nil {
			return nil, err
		}
		var polarMap [][]uint32
		if couplingFlag {
			couplingStep, err := p.GetUint(8)
			if err != nil {
				return nil, err
			}
			couplingStep += 1

			polarMap = make([][]uint32, couplingStep, couplingStep)
			b := fls(int(ident.Channels - 1))
			for j, _ := range polarMap {
				polarMap[j], err = p.GetUintSerial(b, b)
				if err != nil {
					return nil, err
				}
			}
		}
		rsv, err := p.GetUint(2)
		if err != nil {
			return nil, err
		}
		if rsv != 0 {
			return nil, errors.New("non-zero reserved field in mapping setup")
		}

		var mapMux []uint8
		if submapLen > 1 {
			mapMux = make([]uint8, ident.Channels, ident.Channels)
			for j, _ := range mapMux {
				mapMux[j], err = p.GetUint8(4)
				if err != nil {
					return nil, err
				}
				if mapMux[j] > submapLen {
					return nil, errors.New("invalid submap mux value")
				}
			}
		}
		submaps := make([]mappingSubmap, submapLen, submapLen)
		for j, _ := range submaps {
			fields, err := p.GetUintSerial(8, 8, 8)
			if err != nil {
				return nil, err
			}
			submaps[j] = mappingSubmap{
				floor:   fields[1],
				residue: fields[2],
			}
		}

		maps[i] = mappingConfig{
			polarMap: polarMap,
			mapMux:   mapMux,
			submaps:  submaps,
		}
	}
	return maps, nil
}
