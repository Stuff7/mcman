package bitstream

import (
	"testing"
	"time"
)

func TestSetBit(t *testing.T) {
	bs := FromBuffer([]byte{0xFF, 0x00})
	bs.SetBit(false, 5)
	ret := bs.buf[0]
	exp := byte(0xF7)
	if ret != exp {
		t.Logf("SetBit Off Failed\nReturned: %08b\nExpected: %08b", ret, exp)
	}

	bs.SetBit(true, 13)
	ret = bs.buf[1]
	exp = byte(0x08)
	if ret != exp {
		t.Logf("SetBit On Failed\nReturned: %08b\nExpected: %08b", ret, exp)
	}
}

func TestConstLen(t *testing.T) {
	const bits int = 4
	nums := [...]int{13, 3, 7, 6, 9, 8, 0, 0, 8, 5, 4, 2, 0}
	var bs Bitstream
	for _, n := range nums {
		bs.WriteBits(n, bits)
	}
	bsexp := "11010011 01110110 10011000 00000000 10000101 01000010 00000000"
	bsret := bs.String()
	if bsret != bsexp {
		t.Errorf("Write Failed\nReturned: %#+v\nExpected: %#+v", bsret, bsexp)
	}

	var b int
	for int(b) < bits*len(nums) {
		i := b / bits
		n, err := bs.ReadBits(&b, bits)
		if err != nil {
			t.Errorf("Read Error\nerr: %s", err)
		}
		if n != nums[i] {
			t.Errorf("Read failed\nnums[%d]\nReturned: %d\t%08b\nExpected: %d\t%08b", i, n, n, nums[i], nums[i])
		}
	}
}

func TestVariableLen(t *testing.T) {
	var bs Bitstream
	const bits int = 16
	var nums []int
	for i := 1; i < bits; i++ {
		nums = append(nums, i)
		bs.WriteBits(i, i)
	}
	lightningDate := time.Date(1955, time.November, 12, 22, 4, 0, 0, time.UTC)
	expDate := lightningDate.Unix()
	bs.WriteBits64(expDate, 64)
	expStr := "November 12, 1955, also marks the exact time when lightning strikes the Hill Valley clock tower, at exactly 10:04 p.m"
	bs.WritePascalString(expStr)
	bs.WriteBits(5, 3)

	bsexp := "11001101 00001010 00110000 01110000 10000000 01001000 00010100 00000010 " +
		"11000000 00110000 00000001 10100000 00000111 00000000 00001111 11111111 " +
		"11111111 11111111 11111111 11100101 01101001 00110100 01010000 01110101 " +
		"01001110 01101111 01110110 01100101 01101101 01100010 01100101 01110010 " +
		"00100000 00110001 00110010 00101100 00100000 00110001 00111001 00110101 " +
		"00110101 00101100 00100000 01100001 01101100 01110011 01101111 00100000 " +
		"01101101 01100001 01110010 01101011 01110011 00100000 01110100 01101000 " +
		"01100101 00100000 01100101 01111000 01100001 01100011 01110100 00100000 " +
		"01110100 01101001 01101101 01100101 00100000 01110111 01101000 01100101 " +
		"01101110 00100000 01101100 01101001 01100111 01101000 01110100 01101110 " +
		"01101001 01101110 01100111 00100000 01110011 01110100 01110010 01101001 " +
		"01101011 01100101 01110011 00100000 01110100 01101000 01100101 00100000 " +
		"01001000 01101001 01101100 01101100 00100000 01010110 01100001 01101100 " +
		"01101100 01100101 01111001 00100000 01100011 01101100 01101111 01100011 " +
		"01101011 00100000 01110100 01101111 01110111 01100101 01110010 00101100 " +
		"00100000 01100001 01110100 00100000 01100101 01111000 01100001 01100011 " +
		"01110100 01101100 01111001 00100000 00110001 00110000 00111010 00110000 " +
		"00110100 00100000 01110000 00101110 01101101 10100000"
	bsret := bs.String()

	if bsexp != bsret {
		t.Errorf("Write Failed\nReturned: %#+v\nExpected: %#+v", bs.String(), bsexp)
	}

	var b int
	for i := 0; i < len(nums); i++ {
		n, err := bs.ReadBits(&b, i+1)

		if err != nil {
			t.Errorf("Read Error\nerr:%s", err)
		}

		if n != nums[i] {
			t.Errorf("Read failed\nnums[%d]\nReturned: %d\t%08b\nExpected: %d\t%08b", i, n, n, nums[i], nums[i])
		}
	}

	ret, err := bs.ReadBits64(&b, 64)
	if err != nil {
		t.Errorf("ReadBits64 Error\nerr:%s", err)
	}

	if retDate := time.Unix(ret, 0).UTC(); retDate != lightningDate {
		t.Errorf(
			"Wrong lightning date\nReturned: %s (%d)\t%08b\nExpected: %s (%d)\t%08b",
			retDate.Format(time.RFC822), ret, ret,
			lightningDate.Format(time.RFC822), expDate, expDate,
		)
	}

	retStr, err := bs.ReadPascalString(&b)
	if err != nil {
		t.Errorf("ReadPascalString Error\nerr:%s", err)
	}

	if retStr != expStr {
		t.Errorf("ReadPascalString Failed\nReturned: %#+v\nExpected: %#+v", retStr, expStr)
	}

	if n, err := bs.ReadBits(&b, 3); err != nil || n != 5 {
		t.Errorf("Failed to read bits after pascal string\nReturned: %d\nExpected: 5\nerr: %s", n, err)
	}
}
