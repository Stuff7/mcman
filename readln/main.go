package readln

import (
	"fmt"
	"slices"
	"unicode"
)

type Key int

const (
	NA Key = iota
	Char
	Tab
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

func PushLn(prompt string, history *[]string, promptHl func(Key, *string, *int) string) (string, error) {
	var localHistory []string
	var newBuf string
	var pos int
	hpos := len(*history)
	var buf = &newBuf
	lastHistoryIdx := len(*history) - 1

	for {
		promptLn(prompt, promptHl(NA, buf, &pos), pos)
		key, err := ReadCh(buf, &pos)
		if err != nil {
			return "", err
		}

		promptLn(prompt, promptHl(key, buf, &pos), pos)
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
		var item *string
		if localPos >= 0 && localPos < len(localHistory) {
			item = &localHistory[localPos]
		} else if hpos >= 0 && hpos < len(*history) {
			localHistory = append(localHistory, (*history)[hpos])
			item = &localHistory[localPos]
		} else {
			item = &newBuf
		}
		buf = item
		pos = len(*item)
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
		promptLn(prompt, string(*buf), pos)

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

func promptLn(prompt string, input string, cursor int) {
	fmt.Printf("\x1b[2K\r%s%s", prompt, input)
	cursor += len(prompt)
	if cursor > 0 {
		fmt.Printf("\r\x1b[%dC", cursor)
	}
}

func ReadCh(s *string, pos *int) (Key, error) {
	buf := []byte(*s)
	key, ch, err := readKey()
	if err != nil {
		return NA, err
	}

	switch key {
	case Char:
		if *pos == len(buf) {
			buf = append(buf, ch)
		} else {
			buf = slices.Insert(buf, 1, ch)
		}
		*pos++
	case Backspace:
		if *pos > 0 {
			buf = slices.Delete(buf, *pos-1, *pos)
			*pos--
		}
	case ArrowLeft:
		*pos = max(0, *pos-1)
	case ArrowRight:
		*pos = min(len(buf), *pos+1)
	case CtrlBackspace:
		idx := *pos
		for idx > 0 && unicode.IsSpace(rune((buf)[idx-1])) {
			idx--
		}
		for idx > 0 && !unicode.IsSpace(rune((buf)[idx-1])) {
			idx--
		}
		buf = slices.Delete(buf, idx, *pos)
		*pos = idx
	case CtrlArrowLeft:
		idx := *pos
		for idx > 0 && unicode.IsSpace(rune((buf)[idx-1])) {
			idx--
		}
		for idx > 0 && !unicode.IsSpace(rune((buf)[idx-1])) {
			idx--
		}
		*pos = idx
	case CtrlArrowRight:
		idx := *pos
		for idx < len(buf) && unicode.IsSpace(rune((buf)[idx])) {
			idx++
		}
		for idx < len(buf) && !unicode.IsSpace(rune((buf)[idx])) {
			idx++
		}
		*pos = idx
	}
	*s = string(buf)
	return key, nil
}
