package readln

import (
	"fmt"
	"unicode"
)

type Key int

const (
	NA Key = iota
	Char
	CtrlBackspace
	Enter
	ArrowUp
	ArrowDown
	ArrowRight
	ArrowLeft
	CtrlArrowRight
	CtrlArrowLeft
	Backspace
)

func PushLn(prompt string, history *[]string) (string, error) {
	var localHistory []string
	var newBuf string
	var pos int
	hpos := len(*history)
	var buf = &newBuf
	lastHistoryIdx := len(*history) - 1

	for {
		err := promptLn(prompt, *buf, pos)
		if err != nil {
			return "", err
		}

		key, err := ReadCh(buf, &pos)
		if err != nil {
			return "", err
		}

		switch key {
		case Enter:
			goto BreakLoop
		case ArrowUp:
			hpos = max(0, hpos-1)
		case ArrowDown:
			if hpos < len(*history) {
				hpos++
			}
		default:
			continue
		}

		localPos := lastHistoryIdx - hpos
		var item string
		if localPos >= 0 && localPos < len(localHistory) {
			item = localHistory[localPos]
		} else if hpos >= 0 && hpos < len(*history) {
			item = (*history)[hpos]
			localHistory = append(localHistory, item)
		} else {
			item = newBuf
		}
		buf = &item
		pos = len(item)
	}

BreakLoop:
	fmt.Println()
	if len(*buf) == 0 {
		return "", nil
	}

	*history = append(*history, *buf)
	return (*history)[len(*history)-1], nil
}

func ReadLn(prompt string, buf *string) error {
	var pos = len(*buf)

	for {
		err := promptLn(prompt, string(*buf), pos)
		if err != nil {
			return err
		}

		key, err := ReadCh(buf, &pos)
		if err != nil {
			return err
		}
		if key == Enter {
			break
		}
	}

	fmt.Println()
	return nil
}

func promptLn(prompt string, input string, cursor int) error {
	fmt.Printf("\x1b[2K\r%s%s\r", prompt, input)
	cursor += len(prompt)
	if cursor > 0 {
		fmt.Printf("\x1b[%dC", cursor)
	}
	return nil
}

func ReadCh(buf *string, pos *int) (Key, error) {
	key, ch, err := readKey()
	if err != nil {
		return NA, err
	}
	switch key {
	case Char:
		if *pos == len(*buf) {
			*buf += string(ch)
		} else {
			*buf = (*buf)[:*pos] + string(ch) + (*buf)[*pos:]
		}
		*pos++
	case Backspace:
		if *pos > 0 {
			*buf = (*buf)[:*pos-1] + (*buf)[*pos:]
			*pos--
		}
	case ArrowLeft:
		*pos = max(0, *pos-1)
	case ArrowRight:
		*pos = min(len(*buf), *pos+1)
	case CtrlBackspace:
		idx := *pos
		for idx > 0 && unicode.IsSpace(rune((*buf)[idx-1])) {
			idx--
		}
		for idx > 0 && !unicode.IsSpace(rune((*buf)[idx-1])) {
			idx--
		}
		*buf = (*buf)[:idx] + (*buf)[*pos:]
		*pos = idx
	case CtrlArrowLeft:
		idx := *pos
		for idx > 0 && unicode.IsSpace(rune((*buf)[idx-1])) {
			idx--
		}
		for idx > 0 && !unicode.IsSpace(rune((*buf)[idx-1])) {
			idx--
		}
		*pos = idx
	case CtrlArrowRight:
		idx := *pos
		for idx < len(*buf) && unicode.IsSpace(rune((*buf)[idx])) {
			idx++
		}
		for idx < len(*buf) && !unicode.IsSpace(rune((*buf)[idx])) {
			idx++
		}
		*pos = idx
	}
	return key, nil
}
