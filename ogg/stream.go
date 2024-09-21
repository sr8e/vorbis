package ogg

import (
	"errors"
)

type Stream struct {
	serial uint32
	pages  []*Page
}

func (s *Stream) GetPackets() ([]Packet, error) {
	if len(s.pages) == 0 {
		return nil, errors.New("no pages in stream")
	}
	if s.pages[0].streamFlag&1 == 0 {
		return nil, errors.New("invalid stream beginning")
	}
	if s.pages[len(s.pages)-1].streamFlag&0b10 == 0 {
		return nil, errors.New("invalid stream end")
	}
	packetList := make([]Packet, 0)
	var tmp Packet
	for i, page := range s.pages {
		if page.seq != uint32(i) {
			return nil, errors.New("invalid page sequence")
		}
		for _, packet := range page.packets {
			pre := tmp.continueFlag&0b10 != 0
			suf := packet.continueFlag&1 != 0

			if pre && suf {
				tmp.data = append(tmp.data, packet.data...)
				tmp.continueFlag = packet.continueFlag
				tmp.size += packet.size
			} else if !pre && !suf {
				tmp = packet
			} else {
				return nil, errors.New("packet continuation mismatch")
			}
			if tmp.continueFlag&0b10 == 0 {
				packetList = append(packetList, tmp)
			}
		}
	}
	if tmp.continueFlag&0b10 != 0 {
		return nil, errors.New("unfinished packet at the end")
	}
	return packetList, nil
}
