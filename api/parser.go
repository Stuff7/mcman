package api

import (
	"slices"
	"strconv"
	"strings"

	"github.com/stuff7/mcman/readln"
)

type tokenType byte

const (
	Unknown tokenType = iota
	Command
	Ident
	String
	Number
	Assign
	Space
)

type token struct {
	typ tokenType
	val string
}

func (t token) parseString() string {
	if t.typ != String {
		return t.val
	}

	var val strings.Builder
	var isEsc bool
	var i int
	skipWhile(&i, t.val, func(b byte) bool {
		if !isEsc {
			switch b {
			case '\\':
				isEsc = true
			case '"':
				return false
			default:
				val.WriteByte(b)
			}
			return true
		}

		isEsc = false
		switch b {
		case 'b':
			val.WriteByte('\b')
		case 't':
			val.WriteByte('\t')
		case 'n':
			val.WriteByte('\n')
		case 'f':
			val.WriteByte('\f')
		case 'r':
			val.WriteByte('\r')
		case 'u', 'U':
			start := i + 1
			byteLen := 4
			if b == 'U' {
				byteLen = 8
			}

			i += byteLen
			if i > len(t.val) {
				return false
			}

			dec, err := strconv.ParseUint(t.val[start:i+1], 16, byteLen*8)
			if err != nil {
				return true
			}

			val.WriteRune(rune(dec))
		default:
			val.WriteByte(b)
		}

		return true
	})

	return val.String()
}

func (t token) parseNumber() int {
	if t.typ != Number {
		return 0
	}
	val, _ := strconv.Atoi(t.val)
	return val
}

func TestParser(prompt string) error {
	println("Press q to quit")
	var history []string
	var tokens []token

	for {
		in, err := readln.PushLn(prompt, &history, func(_ readln.Key, s *string, _ *int) string {
			tokens = tokenize(*s)
			return renderCmd(tokens)
		})

		if err != nil {
			return err
		}

		if in == "q" {
			break
		}
		for i, t := range tokens {
			println(i, t.parseString())
		}
	}

	println("Quit")
	return nil
}

func tokenize(in string) []token {
	var tokens []token
	for i := 0; i < len(in); {
		b := in[i]
		switch {
		case isDigit(b):
			tokens = append(tokens, token{typ: Number, val: in[skipWhile(&i, in, isDigit):i]})
		case isAlpha(b):
			val := in[skipWhile(&i, in, isAlphanumeric):i]
			if slices.Contains(cmdNamesFlat, val) {
				tokens = append(tokens, token{typ: Command, val: val})
			} else {
				tokens = append(tokens, token{typ: Ident, val: val})

			}
		case readln.IsSpace(b):
			val := in[skipWhile(&i, in, readln.IsSpace):i]
			tokens = append(tokens, token{typ: Space, val: val})
		case b == '=':
			start := i
			i++
			tokens = append(tokens, token{typ: Assign, val: in[start:i]})
		case b == '"':
			isEsc := false
			start := skipWhile(&i, in, func(b byte) bool {
				if !isEsc {
					switch b {
					case '\\':
						isEsc = true
					case '"':
						return false
					}
				}

				return true
			})

			i++
			end := len(in)
			if i < end {
				end = i
			}

			tokens = append(tokens, token{typ: String, val: in[start:end]})
		default:
			i++
		}
	}

	return tokens
}

func renderCmd(tokens []token) string {
	var b strings.Builder
	for _, t := range tokens {
		switch t.typ {
		case Command:
			b.WriteString(clr(226))
		case Ident:
			b.WriteString(clr(117))
		case String:
			b.WriteString(clr(214))
		case Number:
			b.WriteString(clr(194))
		case Assign:
			b.WriteString(clr(254))
		case Space:
			b.WriteString(RESET)
		}
		b.WriteString(t.val)
		b.WriteString(RESET)
	}

	return b.String()
}

func skipWhile(i *int, in string, cond func(b byte) bool) int {
	start := *i
	for *i++; *i < len(in) && cond(in[*i]); *i++ {
	}
	return start
}

func isDigit(b byte) bool {
	return b <= '9' && b >= '0'
}

func isAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_'
}

func isAlphanumeric(b byte) bool {
	return isDigit(b) || isAlpha(b)
}
