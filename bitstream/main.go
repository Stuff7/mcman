package bitstream

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type Bitstream struct {
	b   int
	i   int
	buf []byte
}

func FromBuffer(buf []byte) *Bitstream {
	return &Bitstream{buf: buf, b: len(buf) % 8, i: len(buf) >> 3}
}

func (bs *Bitstream) SetBit(state bool, pos int) error {
	b := byte(pos % 8)
	i := pos >> 3
	if i >= len(bs.buf) {
		return errors.New(fmt.Sprintf("Bit position %d is out of bounds for bitstream of %d bytes", pos, len(bs.buf)))
	}

	cb := &bs.buf[i]
	m := turnOffRight(turnOffLeft(0xFF, b-1), 8-b)
	if state {
		*cb |= m
	} else {
		*cb &= ^m
	}

	return nil
}

func (bs *Bitstream) WriteBits(t int, bits int) {
	for bits > 0 {
		t &= 0xFF_FF_FF_FF >> (32 - bits)
		b := bs.currentByte()
		size := min(bits, 8-bs.b)
		mask := byte(int(t)>>max(bits-size, 0)) << (8 - bs.b - size)
		*b |= mask

		written := bs.b + size
		bs.b = written % 8
		bs.i += written >> 3
		bits -= size
	}
}

func (bs *Bitstream) ReadBits(bitpos *int, bits int) (int, error) {
	var t int
	var readSize int
	for ; bits > 0; t <<= bits {
		currBPos := *bitpos % 8
		i := *bitpos >> 3
		if i > len(bs.buf) || (i == len(bs.buf) && *bitpos >= bs.b) {
			return t, errors.New("EOF")
		}

		b := bs.buf[i]
		readSize = min(8-currBPos, bits)
		m := turnOffRight(turnOffLeft(0xFF, byte(currBPos)), byte(8-readSize-currBPos))
		t |= int(b&m) >> (8 - readSize - currBPos)

		*bitpos += readSize
		bits -= readSize
	}

	return t, nil
}

func turnOffLeft(n byte, c byte) byte {
	return n & ((byte(0xFF) << c) >> c)
}

func turnOffRight(n byte, c byte) byte {
	return n & ((byte(0xFF) >> c) << c)
}

func (bs *Bitstream) SaveToDisk(name string) error {
	return os.WriteFile(name, bs.buf, 0666)
}

func (bs Bitstream) String() string {
	var bd strings.Builder
	for _, b := range bs.buf {
		bd.WriteString(fmt.Sprintf("%08b ", b))
	}
	ret := bd.String()
	return ret[:len(ret)-1]
}

func (b *Bitstream) currentByte() *byte {
	if b.i >= len(b.buf) {
		b.buf = append(b.buf, 0)
	}
	return &b.buf[b.i]
}
