package vorbis

import (
	"errors"
	"github.com/sr8e/vorbis/ogg"
	"slices"
)

type floorConfig struct {
	floorType uint16
	config1   *floor1Config
}

type floor1Config struct {
	xList      []uint16
	classes    []floor1Class
	multiplier uint8
}

type floor1Class struct {
	dimension   uint8
	subclassNum uint8
	masterBook  uint8
	subBooks    []int
}

func readFloorConfig(p *ogg.Packet) ([]floorConfig, error) {
	tmp, err := p.GetUint(6)
	if err != nil {
		return nil, err
	}
	floorLen := tmp + 1
	configs := make([]floorConfig, floorLen, floorLen)
	for i, _ := range configs {
		floorType, err := p.GetUint16(16)
		if err != nil {
			return nil, err
		}

		if floorType == 0 {
			// TODO
			return nil, errors.New("floor type 0 is not implemented yet :(")
		} else if floorType == 1 {
			configs[i], err = readFloor1Header(p)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, errors.New("invalid floor Type")
		}
	}
	return configs, nil
}

func readFloor1Header(p *ogg.Packet) (_ floorConfig, err error) {
	partLen, err := p.GetUint(5)
	if err != nil {
		return
	}
	partCls := make([]uint8, partLen, partLen)
	for i, _ := range partCls {
		partCls[i], err = p.GetUint8(4)
		if err != nil {
			return
		}
	}

	clsSize := slices.Max(partCls) + 1
	classes := make([]floor1Class, clsSize, clsSize)
	for i, _ := range classes {
		var dim, subcls, masterBook uint8
		dim, err = p.GetUint8(3)
		if err != nil {
			return
		}
		dim += 1

		subcls, err = p.GetUint8(2)
		if err != nil {
			return
		}

		if subcls != 0 {
			masterBook, err = p.GetUint8(8)
			if err != nil {
				return
			}
		}

		subBookLen := 1 << subcls
		subBooks := make([]int, subBookLen, subBookLen)
		for j, _ := range subBooks {
			var tmp int
			tmp, err = p.GetUintAsInt(8)
			if err != nil {
				return
			}
			subBooks[j] = tmp - 1
		}
		classes[i] = floor1Class{
			dimension:   dim,
			subclassNum: subcls,
			masterBook:  masterBook,
			subBooks:    subBooks,
		}
	}
	mul, err := p.GetUint8(2)
	if err != nil {
		return
	}
	rangeBits, err := p.GetUint(4)
	if err != nil {
		return
	}
	xList := make([]uint16, 2)
	xList[1] = 1 << rangeBits

	for _, part := range partCls {
		for j := 0; j < int(classes[part].dimension); j++ {
			var v uint16
			v, err = p.GetUint16(rangeBits)
			if err != nil {
				return
			}
			xList = append(xList, v)
		}
	}
	config := floor1Config{
		xList:      xList,
		classes:    classes,
		multiplier: mul,
	}
	return floorConfig{
		floorType: 1,
		config1:   &config,
	}, nil
}
