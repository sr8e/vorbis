package crc

var p uint32 = 0x04c11db7

var table [256]uint32

func init() {
	createTable()
}

func createTable() {
	for i := 0; i < 256; i++ {
		c := uint32(i) << 24
		for j := 0; j < 8; j++ {
			flag := (c >> 31) & 0b1
			c = c << 1
			if flag == 1 {
				c ^= p
			}
		}
		table[i] = c
	}
}

func CRC32(value []byte, initMask uint32, finalMask uint32) uint32 {
	c := initMask
	for _, b := range value {
		c = (c<<8 + uint32(b)) ^ table[c>>24]
	}
	return c ^ finalMask
}
