//go:build unix
// +build unix

package readln

import (
	"os"
	"syscall"
	"unsafe"
)

func readKey() (Key, byte, error) {
	ch, err := getch()
	if err != nil {
		return NA, 0, err
	}

	key := NA
	switch ch {
	case 8, 23:
		key = CtrlBackspace
	case 9:
		key = Tab
	case 10:
		key = Enter
	case 27:
		key, err = parseEscSeq()
	case 127:
		key = Backspace
	default:
		if ch > 31 {
			return Char, ch, err
		}
	}

	return key, 0, nil
}

const ESC_SEQ_LEN = 6

var ESC_SEQ_LIST = [ESC_SEQ_LEN]struct {
	key Key
	seq []byte
}{
	{ArrowUp, []byte("[A")},
	{ArrowDown, []byte("[B")},
	{ArrowRight, []byte("[C")},
	{ArrowLeft, []byte("[D")},
	{CtrlArrowRight, []byte("[1;5C")},
	{CtrlArrowLeft, []byte("[1;5D")},
}

func parseEscSeq() (Key, error) {
	var ch byte
	pos := 0
	for pos < ESC_SEQ_LEN+1 {
		var err error
		ch, err = getch()
		if err != nil {
			return 0, err
		}

		for _, item := range ESC_SEQ_LIST {
			seq := item.seq
			if pos >= len(seq) {
				continue
			}
			if seq[pos] == ch && len(seq)-1 == pos {
				return item.key, nil
			}
		}
		pos++
	}
	return 0, nil
}

func getch() (byte, error) {
	var buf [1]byte
	var old syscall.Termios
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdin), uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(&old)))
	if err != 0 {
		return 0, err
	}

	old.Lflag &^= syscall.ICANON | syscall.ECHO
	old.Cc[syscall.VMIN] = 1
	old.Cc[syscall.VTIME] = 0

	_, _, err = syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdin), uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(&old)))
	if err != 0 {
		return 0, err
	}

	_, err1 := os.Stdin.Read(buf[:])
	if err1 != nil {
		return 0, err
	}

	old.Lflag |= syscall.ICANON | syscall.ECHO
	_, _, err = syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdin), uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(&old)))
	if err != 0 {
		return 0, err
	}

	return buf[0], nil
}
