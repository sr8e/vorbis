package ogg

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/sr8e/vorbis/crc"
)

type OggLoader struct {
	file    *os.File
	buf     []byte
	cur     int // cursor position of buf going to be read.
	bufLen  int // length of buffer.  bufLen - cur bytes can be read.
	Streams map[uint32]Stream
}

type Page struct {
	stream     uint32
	streamFlag byte // 1 for beginning, 2 for end (3 for both?)
	granule    uint64
	seq        uint32
	packets    []Packet
}

func (ol *OggLoader) Open(path string) error {
	if ol.file != nil {
		return errors.New("file is already opened")
	}
	fp, err := os.Open(path)
	if err != nil {
		return err
	}
	ol.file = fp
	ol.buf = make([]byte, 256)

	return nil
}

func (ol *OggLoader) Close() error {
	return ol.file.Close()
}

func (ol *OggLoader) ReadAll() error {
	ol.Streams = map[uint32]Stream{}
	for {
		p, err := ol.readPage()
		if err != nil {
			return err
		}
		if p == nil {
			break
		}
		if s, ok := ol.Streams[p.stream]; !ok {
			ol.Streams[p.stream] = Stream{serial: p.stream, pages: []*Page{p}}
		} else {
			s.pages = append(s.pages, p)
			ol.Streams[p.stream] = s
		}
	}
	return nil
}

func (ol *OggLoader) readPage() (*Page, error) {
	p := &Page{}
	pageBytes := make([]byte, 0)

	pattern, err := ol.getBytes(4)
	if err != nil {
		if errors.Is(err, io.EOF) {
			// would be proper end of file
			return nil, nil
		}
		return nil, err
	}
	if string(pattern) != "OggS" {
		return nil, errors.New("cannot capture page header")
	}
	pageBytes = append(pageBytes, pattern...)

	fields, err := ol.getBytes(23)
	if err != nil {
		return nil, err
	}
	pageBytes = append(pageBytes, fields...)

	typeFlag := fields[1]
	continued := typeFlag&1 == 1
	p.streamFlag = typeFlag >> 1 & 0b11

	p.granule = binary.LittleEndian.Uint64(fields[2:10])
	p.stream = binary.LittleEndian.Uint32(fields[10:14])
	p.seq = binary.LittleEndian.Uint32(fields[14:18])

	checksum := binary.LittleEndian.Uint32(fields[18:22])
	// fill 0 instead of checksum to verify
	copy(pageBytes[22:26], []byte{0, 0, 0, 0})

	segListLen := int(fields[22])

	p.packets = make([]Packet, 0)
	initPacket := Packet{}
	if continued {
		initPacket.continueFlag |= 1
	}
	p.packets = append(p.packets, initPacket)

	segLens, err := ol.getBytes(segListLen)
	if err != nil {
		return nil, err
	}
	pageBytes = append(pageBytes, segLens...)

	packetIndex := 0
	for i, sl := range segLens {
		p.packets[packetIndex].size += int(sl)

		if sl != 0xff && i < segListLen-1 {
			// next packet exists
			packetIndex++
			p.packets = append(p.packets, Packet{})
		} else if sl == 0xff && i == segListLen-1 {
			// this packet continues to next page
			p.packets[packetIndex].continueFlag |= 0b10
		}
	}

	for i, packet := range p.packets {
		data, err := ol.getBytes(packet.size)
		if err != nil {
			return nil, err
		}
		p.packets[i].data = data
		pageBytes = append(pageBytes, data...)
	}

	// end of page, calculate checksum
	calcsum := crc.CRC32(append(pageBytes, 0x0, 0x0, 0x0, 0x0), 0x0, 0x0)
	if checksum != calcsum {
		return nil, fmt.Errorf("checksum does not match, read: %x <-> calc: %x", checksum, calcsum)
	}

	return p, nil
}

func (ol *OggLoader) getBytes(n int) ([]byte, error) {
	if n <= ol.bufLen-ol.cur {
		b := make([]byte, 0, n)
		b = append(b, ol.buf[ol.cur:ol.cur+n]...)
		ol.cur = ol.cur + n
		return b, nil
	}

	resLen := n - (ol.bufLen - ol.cur)
	b := make([]byte, 0, n)
	b = append(b, ol.buf[ol.cur:ol.bufLen]...)

	readBytesLen, err := ol.file.Read(ol.buf)
	ol.bufLen = readBytesLen
	ol.cur = 0

	if err != nil {
		if ol.bufLen == 0 && errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("encountered EOF while reading: %w", err)
		}

		return nil, err
	}

	res, err := ol.getBytes(resLen)
	if err != nil {
		return nil, err
	}

	b = append(b, res...)
	return b, nil

}
