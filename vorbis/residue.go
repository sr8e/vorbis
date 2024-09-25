package vorbis

import (
	"fmt"
	"github.com/sr8e/vorbis/ogg"
)

type residueConfig struct {
	residueType   uint16
	begin         uint32
	end           uint32
	partitionSize uint32
	classLen      uint8
	classBook     uint8
	residueBooks  [][8]int
}

func readResidueConfig(p *ogg.Packet) ([]residueConfig, error) {
	tmp, err := p.GetUint(6)
	if err != nil {
		return nil, err
	}
	residueLen := tmp + 1
	configs := make([]residueConfig, residueLen, residueLen)
	for i, _ := range configs {
		residueType, err := p.GetUint16(16)
		if err != nil {
			return nil, err
		}

		if residueType < 3 {
			cfg, err := readResidueHeader(p)
			if err != nil {
				return nil, err
			}
			cfg.residueType = residueType
			configs[i] = cfg
		} else {
			return nil, fmt.Errorf("invalid residue type %d", residueType)
		}
	}
	return configs, nil
}

func readResidueHeader(p *ogg.Packet) (_ residueConfig, err error) {
	fields, err := p.GetUintSerial(24, 24, 24, 6, 8)
	if err != nil {
		return
	}
	clsLen := fields[3] + 1
	cascade := make([]uint8, clsLen, clsLen)
	for i, _ := range cascade {
		var high, low uint8
		var flag bool
		low, err = p.GetUint8(3)
		if err != nil {
			return
		}
		flag, err = p.GetFlag()
		if err != nil {
			return
		}
		if flag {
			high, err = p.GetUint8(5)
			if err != nil {
				return
			}
		}
		cascade[i] = high<<3 + low
	}

	residueBooks := make([][8]int, clsLen, clsLen)
	for i, v := range cascade {
		for j := 0; j < 8; j++ {
			if (v>>j)&1 == 1 {
				residueBooks[i][j], err = p.GetUintAsInt(8)
				if err != nil {
					return
				}
			} else {
				// unused
				residueBooks[i][j] = -1
			}
		}
	}

	return residueConfig{
		begin:         fields[0],
		end:           fields[1],
		partitionSize: fields[2] + 1,
		classLen:      uint8(clsLen),
		classBook:     uint8(fields[4]),
		residueBooks:  residueBooks,
	}, nil
}
