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
	dimension uint16
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

	dim, err := p.GetUint16(16)
	if err != nil {
		return
	}
	entryLen, err := p.GetUint(24)
	if err != nil {
		return
	}

	entries, err := readCodebookEntries(p, entryLen)
	if err != nil {
		return
	}
	vq, err := readVQLookup(p, dim, entryLen)
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

func readCodebookEntries(p *ogg.Packet, entryLen uint32) ([]int, error) {
	entries := make([]int, entryLen)

	ordered, err := p.GetFlag()
	if err != nil {
		return nil, err
	}

	if ordered {
		curLen, err := p.GetUintAsInt(5)
		if err != nil {
			return nil, err
		}
		curLen += 1
		for i := uint32(0); i < entryLen; {
			num, err := p.GetUint(fls(int(entryLen - i)))
			if err != nil {
				return nil, err
			}
			for j := i; j < i+num; j++ {
				entries[j] = curLen
			}
			i += num
			curLen++
		}
	} else {
		sparse, err := p.GetFlag()
		if err != nil {
			return nil, err
		}
		for i := range entries {
			if sparse {
				used, err := p.GetFlag()
				if err != nil {
					return nil, err
				}
				if !used {
					entries[i] = -1
					continue
				}
			}
			cwLen, err := p.GetUintAsInt(5)
			if err != nil {
				return nil, err
			}
			entries[i] = cwLen + 1
		}
	}
	return entries, nil
}

func readVQLookup(p *ogg.Packet, dimension uint16, entryLen uint32) (_ vqLookup, err error) {
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

	values, err := p.GetUintSerial(32, 32, 4, 1)
	if err != nil {
		return
	}
	minimum := toFloat(values[0])
	delta := toFloat(values[1])
	bits := values[2] + 1
	seqFlag := values[3] == 1

	var lookupLen int
	if lookup == 1 {
		lookupLen = lookup1Values(dimension, entryLen)
	} else {
		lookupLen = int(dimension) * int(entryLen)
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
		for i := range vectors {
			var last float64
			mulOfs := i
			vectors[i] = make([]float64, dimension)
			for j := range vectors[i] {
				vectors[i][j] = float64(muls[mulOfs%lookupLen])*delta + minimum + last
				if seqFlag {
					last = vectors[i][j] // what tf is this for?
				}
				mulOfs /= lookupLen
			}
		}
	} else {
		for i := range vectors {
			var last float64
			vectors[i] = make([]float64, dimension)
			for j := range vectors[i] {
				vectors[i][j] = float64(muls[i*int(dimension)+j])*delta + minimum + last
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

func (cb *codebook) readValue(p *ogg.Packet) (int, error) {
	tree := cb.decisionTree
	tree.Reset()

	for tree.GetValue() == -1 {
		dir, err := p.GetFlag()
		if err != nil {
			return 0, err
		}
		tree.Descend(dir)
	}
	return tree.GetValue(), nil
}

// ReadScalarValue reads bits from packet until it encounters leaf node in decision tree and returns scalar value.
func (cb *codebook) ReadScalarValue(p *ogg.Packet) (int, error) {
	if cb.vqMap.dimension != 0 {
		return 0, errors.New("cannot read scalar value from vector context")
	}
	return cb.readValue(p)
}

// ReadVectorValue reads bits from packet until it encounters leaf node in decision tree and returns vector value from VQ lookup table.
func (cb *codebook) ReadVectorValue(p *ogg.Packet) ([]float64, error) {
	if cb.vqMap.dimension == 0 {
		return nil, errors.New("cannot read vector value from scalar context")
	}
	vqIndex, err := cb.readValue(p)
	if err != nil {
		return nil, err
	}
	return cb.vqMap.vectors[vqIndex], nil
}
