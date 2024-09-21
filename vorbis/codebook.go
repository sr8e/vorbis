package vorbis

import (
	"errors"
	"github.com/sr8e/vorbis/huffman"
	"github.com/sr8e/vorbis/ogg"
)

type codebook struct {
	decisionTree huffman.HuffmanTree
	vqMap        vqLookup
}

type vqLookup struct {
	dimension int
	vectors   [][]float64
}

func readCodebook(p *ogg.Packet) (_ codebook, err error) {
	pattern, err := p.GetUint(24)
	if err != nil {
		return
	}
	if pattern != 0x564342 {
		err = errors.New("cannot capture codebook sync pattern")
		return
	}

	dim, err := p.GetUint(16)
	if err != nil {
		return
	}
	entryLen, err := p.GetUint(24)
	if err != nil {
		return
	}

	entries, err := readCodebookEntries(p, int(entryLen))
	if err != nil {
		return
	}
	vq, err := readVQLookup(p, int(dim), int(entryLen))
	if err != nil {
		return
	}
	tree, err := huffman.GenerateHuffmanTree(entries)
	if err != nil {
		return
	}

	return codebook{
		decisionTree: tree,
		vqMap:        vq,
	}, nil
}

func readCodebookEntries(p *ogg.Packet, entryLen int) ([]int, error) {
	entries := make([]int, entryLen)

	ordered, err := p.GetUint(1)
	if err != nil {
		return nil, err
	}

	if ordered != 0 {
		// TODO
		return nil, errors.New("ordered codebook is not implemented yet :(")
	} else {
		sparse, err := p.GetUint(1)
		if err != nil {
			return nil, err
		}
		for i := 0; i < entryLen; i++ {
			if sparse != 0 {
				flag, err := p.GetUint(1)
				if err != nil {
					return nil, err
				}
				if flag == 0 { // unused entry
					entries[i] = -1
					continue
				}
			}
			cwLen, err := p.GetUint(5)
			if err != nil {
				return nil, err
			}
			entries[i] = int(cwLen) + 1
		}
	}
	return entries, nil
}

func readVQLookup(p *ogg.Packet, dimension int, entryLen int) (_ vqLookup, err error) {
	lookup, err := p.GetUint(4)
	if err != nil {
		return
	}
	if lookup == 0 {
		return
	}
	if lookup > 2 {
		err = errors.New("invalid VQ type")
		return
	}

	values, err := p.GetUintSerial([]int{32, 32, 4, 1})
	if err != nil {
		return
	}
	minimum := toFloat(values[0])
	delta := toFloat(values[1])
	bits := int(values[2]) + 1
	seqFlag := values[3] == 1

	var lookupLen int
	if lookup == 1 {
		lookupLen = lookup1Values(dimension, entryLen)
	} else {
		lookupLen = dimension * entryLen
	}
	muls := make([]uint32, lookupLen)
	for i := 0; i < lookupLen; i++ {
		muls[i], err = p.GetUint(bits)
		if err != nil {
			return
		}
	}
	vectors := make([][]float64, entryLen)
	if lookup == 1 {
		for i := 0; i < entryLen; i++ {
			var last float64
			mulOfs := i
			vectors[i] = make([]float64, dimension)
			for j := 0; j < dimension; j++ {
				vectors[i][j] = float64(muls[mulOfs%lookupLen])*delta + minimum + last
				if seqFlag {
					last = vectors[i][j] // what tf is this for?
				}
				mulOfs /= lookupLen
			}
		}
	} else {
		for i := 0; i < entryLen; i++ {
			var last float64
			vectors[i] = make([]float64, dimension)
			for j := 0; j < dimension; j++ {
				vectors[i][j] = float64(muls[i*dimension+j])*delta + minimum + last
				if seqFlag {
					last = vectors[i][j]
				}
			}
		}
	}

	return vqLookup{
		dimension: dimension,
		vectors:   vectors,
	}, nil
}
