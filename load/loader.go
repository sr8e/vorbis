package load

import (
	"errors"
	"fmt"
	"io"
	"os"
)

type BinaryLoader struct {
	file   *os.File
	buf    []byte
	cur    int // cursor position of buf going to be read.
	bufLen int // length of buffer.  bufLen - cur bytes can be read.
}

func (bl *BinaryLoader) Open(path string) error {
	if bl.file != nil {
		return errors.New("file is already opened")
	}
	fp, err := os.Open(path)
	if err != nil {
		return err
	}
	bl.file = fp
	bl.buf = make([]byte, 4096)

	return nil
}

func (bl *BinaryLoader) Close() error {
	return bl.file.Close()
}

func (bl *BinaryLoader) GetBytes(n int) ([]byte, error) {
	if n == 0 {
		return []byte{}, nil
	}
	if n <= bl.bufLen-bl.cur {
		b := make([]byte, 0, n)
		b = append(b, bl.buf[bl.cur:bl.cur+n]...)
		bl.cur = bl.cur + n
		return b, nil
	}

	resLen := n - (bl.bufLen - bl.cur)
	b := make([]byte, 0, n)
	b = append(b, bl.buf[bl.cur:bl.bufLen]...)
	bl.cur = 0

	for resLen > 0 {
		readSize, err := bl.file.Read(bl.buf)
		bl.bufLen = readSize
		if err != nil {
			if readSize == 0 && errors.Is(err, io.EOF) {
				return nil, fmt.Errorf("encountered EOF while reading: %w", err)
			}
			return nil, err
		}
		size := min(readSize, resLen)
		b = append(b, bl.buf[0:size]...)
		resLen -= size
		bl.cur = size
	}
	return b, nil
}
