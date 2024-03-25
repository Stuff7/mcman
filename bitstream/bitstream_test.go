package bitstream

import "testing"

func TestSetBit(t *testing.T) {
	bs := FromBuffer([]byte{0xFF, 0x00})
	bs.SetBit(false, 5)
	ret := bs.buf[0]
	exp := byte(0xF7)
	if ret != exp {
		t.Logf("SetBit On Failed\nReturned: %08b\nExpected: %08b", ret, exp)
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
			t.Errorf("Read failed\nnums[%d]\nReturned: %d\nExpected: %d", i, n, nums[i])
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
	bsexp := "11001101 00001010 00110000 01110000 10000000 01001000 00010100 00000010 11000000 00110000 00000001 10100000 00000111 00000000 00001111"
	bsret := bs.String()
	if bsret != bsexp {
		t.Errorf("Write Failed\nReturned: %#+v\nExpected: %#+v", bsret, bsexp)
	}

	var b int
	for i := 0; i < len(nums); i++ {
		n, err := bs.ReadBits(&b, i+1)
		if err != nil {
			t.Errorf("Read Error\nerr:%s", err)
		}
		if n != nums[i] {
			t.Errorf("Read failed\nnums[%d]\nReturned: %d\nExpected: %d", i, n, nums[i])
		}
	}
}
