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
	configs := make([]residueConfig, residueLen)
	for i := range configs {
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
	cascade := make([]uint8, clsLen)
	for i := range cascade {
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

	residueBooks := make([][8]int, clsLen)
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

func readResiduePacket(p *ogg.Packet, blockExp int, config residueConfig, codebooks []codebook, noDecodeFlags []bool) ([][]float64, error) {
	n := 1 << blockExp
	chNum := len(noDecodeFlags)
	if config.residueType == 2 {
		flag := true
		for _, v := range noDecodeFlags {
			flag = flag && v
		}
		n *= chNum
		noDecodeFlags = []bool{flag}
	}
	decoded, err := decodeCommonResiduePacket(p, n, config, codebooks, noDecodeFlags)

	if err != nil {
		return nil, err
	}
	if config.residueType != 2 {
		return decoded, nil
	}

	// de-interleave
	vec := make([][]float64, chNum)
	for i := range vec {
		vec[i] = make([]float64, n/chNum)
	}
	for i, val := range decoded[0] {
		vec[i%chNum][i/chNum] = val
	}
	return vec, nil
}

func decodeCommonResiduePacket(p *ogg.Packet, n int, config residueConfig, codebooks []codebook, noDecodeFlags []bool) (_ [][]float64, err error) {
	chNum := len(noDecodeFlags)
	resVectors := make([][]float64, chNum)

	begin := min(n, int(config.begin))
	end := min(n, int(config.end))
	readSize := end - begin
	if readSize <= 0 {
		for i := range resVectors {
			resVectors[i] = make([]float64, n)
		}
		return resVectors, nil
	}
	partSize := int(config.partitionSize)
	partNum := readSize / partSize
	cwDim := int(codebooks[config.classBook].vqMap.dimension)

	partClasses := make([][]int, chNum)

	for phase := 0; phase < 8; phase++ {
		partCount := 0
		for partCount < partNum {
			if phase == 0 { // read initial codeword
				for ch, flag := range noDecodeFlags {
					partClasses[ch] = make([]int, partNum) // ?
					resVectors[ch] = make([]float64, n)

					if flag {
						continue
					}
					temp, err := codebooks[config.classBook].ReadScalarValue(p)
					if err != nil {
						return nil, err
					}
					for i := cwDim - 1; i >= 0; i-- {
						partClasses[ch][i+partCount] = temp % int(config.classLen)
						temp /= int(config.classLen)
					}
				}
			}
			for i := 0; i < cwDim; i++ {
				for ch, flag := range noDecodeFlags {
					if flag {
						continue
					}
					pcls := partClasses[ch][partCount]
					vqBookIndex := config.residueBooks[pcls][phase]
					if vqBookIndex == -1 { // unused
						continue
					}
					vqBook := codebooks[vqBookIndex]
					offset := begin + partCount*partSize
					var partVec []float64
					if config.residueType == 0 {
						partVec, err = decodeResidue0(p, vqBook, partSize)
					} else {
						partVec, err = decodeResidue1(p, vqBook, partSize)
					}
					if err != nil {
						return nil, err
					}
					for j, v := range partVec {
						resVectors[ch][offset+j] += v
					}
				}
				partCount++
			}
		}
	}
	return resVectors, nil
}

func decodeResidue0(p *ogg.Packet, vqBook codebook, partSize int) ([]float64, error) {
	v := make([]float64, partSize)
	dim := int(vqBook.vqMap.dimension)
	step := partSize / dim
	for i := 0; i < step; i++ {
		tmp, err := vqBook.ReadVectorValue(p)
		if err != nil {
			return nil, err
		}
		for j := 0; j < dim; j++ {
			v[i+step*j] = tmp[j]
		}
	}
	return v, nil
}

func decodeResidue1(p *ogg.Packet, vqBook codebook, partSize int) ([]float64, error) {
	v := make([]float64, 0, partSize)
	dim := int(vqBook.vqMap.dimension)
	for i := 0; i < partSize; i += dim {
		tmp, err := vqBook.ReadVectorValue(p)
		if err != nil {
			return nil, err
		}
		v = append(v, tmp...)
	}
	return v, nil
}
