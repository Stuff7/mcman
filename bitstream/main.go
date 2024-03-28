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

func (bs *Bitstream) BitPosition() int {
	return bs.b
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

func (bs *Bitstream) WritePascalString(s string) error {
	if bs.b != 0 {
		bs.b = 0
		bs.i++
	}
	b := bs.currentByte()
	sLen := len(s)
	if sLen > 0xFF {
		return errors.New(fmt.Sprintf("String is too long cannot write as Pascal: %#+v", s))
	}

	*b = byte(sLen)
	bs.buf = append(bs.buf, s...)
	bs.i = len(bs.buf)

	return nil
}

func WriteBits[T int | int64](bs *Bitstream, t T, bits int) {
	for bits > 0 {
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

func (bs *Bitstream) WriteBits(t int, bits int) {
	WriteBits(bs, t, bits)
}

func (bs *Bitstream) WriteBits64(t int64, bits int) {
	WriteBits(bs, t, bits)
}

func ReadBits[T int | int64](bs *Bitstream, bitpos *int, bits int) (T, error) {
	var t uint64
	var readSize int
	for bits > 0 {
		currBPos := *bitpos % 8
		i := *bitpos >> 3
		if i > len(bs.buf) || (i == len(bs.buf) && *bitpos >= bs.b) {
			return T(t), errors.New("EOF")
		}

		b := bs.buf[i]
		readSize = min(8-currBPos, bits)
		if bits > 0 {
			t <<= uint64(readSize)
		}
		m := turnOffRight(turnOffLeft(0xFF, byte(currBPos)), byte(8-readSize-currBPos))
		t |= uint64(b&m) >> (8 - readSize - currBPos)

		*bitpos += readSize
		bits -= readSize
	}

	return T(t), nil
}

func (bs *Bitstream) ReadBits(bitpos *int, bits int) (int, error) {
	return ReadBits[int](bs, bitpos, bits)
}

func (bs *Bitstream) ReadBits64(bitpos *int, bits int) (int64, error) {
	return ReadBits[int64](bs, bitpos, bits)
}

func (bs *Bitstream) ReadPascalString(bitpos *int) (string, error) {
	bp := *bitpos % 8
	i := *bitpos >> 3
	if bp != 0 {
		*bitpos += 8 - bp
		bp = 0
		i++
	}

	if i >= len(bs.buf) {
		return "", errors.New("Tried to read pascal string out of bounds")
	}

	sLen := bs.buf[i]
	i++
	if i+int(sLen) > len(bs.buf) {
		return "", errors.New("Pascal string is too long")
	}

	s := string(bs.buf[i : i+int(sLen)])
	*bitpos += int(sLen)*8 + 8

	return s, nil
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
