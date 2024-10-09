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
	partitions []uint8
	classes    []floor1Class
	multiplier uint8
}

type floor1Class struct {
	dimension   uint8
	subclassNum uint8
	masterBook  uint8
	subBooks    []int
}

var floor1Multiplier = []int{256, 128, 86, 64}

func readFloorConfig(p *ogg.Packet) ([]floorConfig, error) {
	tmp, err := p.GetUint(6)
	if err != nil {
		return nil, err
	}
	floorLen := tmp + 1
	configs := make([]floorConfig, floorLen)
	for i := range configs {
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
	partCls := make([]uint8, partLen)
	for i := range partCls {
		partCls[i], err = p.GetUint8(4)
		if err != nil {
			return
		}
	}

	clsSize := slices.Max(partCls) + 1
	classes := make([]floor1Class, clsSize)
	for i := range classes {
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
		subBooks := make([]int, subBookLen)
		for j := range subBooks {
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
		partitions: partCls,
		classes:    classes,
		multiplier: mul,
	}
	return floorConfig{
		floorType: 1,
		config1:   &config,
	}, nil
}

func readFloor1Packet(p *ogg.Packet, blockExp int, config floor1Config, codebooks []codebook) ([]int, error) {
	nonZeroFlag, err := p.GetFlag()
	if err != nil {
		return nil, err
	}
	if !nonZeroFlag { // unused floor
		return nil, nil
	}
	yRange := floor1Multiplier[config.multiplier-1]
	yBits := fls(yRange - 1)
	yValues := make([]int, 0)
	yInits, err := p.GetUintSerial(yBits, yBits)
	if err != nil {
		return nil, err
	}
	yValues = append(yValues, int(yInits[0]), int(yInits[1]))

	for _, clsIndex := range config.partitions {
		cls := config.classes[clsIndex]
		cbits := cls.subclassNum
		mask := (1 << cbits) - 1

		var cval int
		if cbits > 0 {
			cval, err = codebooks[cls.masterBook].ReadScalarValue(p)
			if err != nil {
				return nil, err
			}
		}
		for j := 0; j < int(cls.dimension); j++ {
			book := cls.subBooks[cval&mask]
			cval >>= cbits
			if book >= 0 {
				yVal, err := codebooks[book].ReadScalarValue(p)
				if err != nil {
					return nil, err
				}
				yValues = append(yValues, yVal)
			} else {
				yValues = append(yValues, 0)
			}
		}
	}
	xValues := config.xList
	if len(xValues) != len(yValues) {
		return nil, errors.New("floor curve value length mismatch")
	}

	for i := 2; i < len(yValues); i++ {
		lowNeigh := lowNeighbor(xValues, i)
		highNeigh := highNeighbor(xValues, i)
		pred := renderPoint(xValues[lowNeigh], xValues[highNeigh], yValues[lowNeigh], yValues[highNeigh], xValues[i])
		room := 2 * min(pred, yRange-pred)

		if val := yValues[i]; val < room {
			sign := val & 1
			diff := val >> 1
			if sign == 1 {
				diff = -(diff + 1)
			}
			yValues[i] = pred + diff
		} else {
			if yRange > 2*pred {
				yValues[i] = val
			} else {
				yValues[i] = yRange - val - 1
			}
		}
	}
	sortedIndex := make([]int, len(xValues))
	for i := range sortedIndex {
		sortedIndex[i] = i
	}
	slices.SortFunc(sortedIndex, func(a, b int) int { return int(xValues[a]) - int(xValues[b]) })

	n := uint16(1 << blockExp)
	finalY := make([]int, n)

	prevIndex := sortedIndex[0]
	finalY[0] = yValues[0]
	for _, curIndex := range sortedIndex[1:] {
		x0 := xValues[prevIndex]
		x1 := xValues[curIndex]
		if x1 > n { // truncate
			x1 = n
		}
		for x := x0 + 1; x <= x1; x++ {
			finalY[x] = renderPoint(x0, x1, yValues[prevIndex], yValues[curIndex], x)
		}
		prevIndex = curIndex
	}
	if xMax := slices.Max(xValues); xMax < n { // fill
		for x := xMax + 1; x < n; x++ {
			finalY[x] = finalY[xMax]
		}
	}

	return finalY, nil
}
