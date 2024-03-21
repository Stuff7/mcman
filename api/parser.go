package api

import (
	"slices"
	"strconv"
	"strings"

	"github.com/stuff7/mcman/readln"
)

type tokenType int

const (
	Unknown tokenType = iota
	Command
	Ident
	String
	Number
	Assign
	Whitespace
)

type token struct {
	typ tokenType
	val string
}

func TestParser(prompt string) error {
	println("Press q to quit")
	var history []string

	for {
		in, err := readln.PushLn(prompt, &history, func(_ readln.Key, s *string, _ *int) string {
			return renderCmd(tokenize(*s))
		})

		if err != nil {
			return err
		}

		if in == "q" {
			break
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
			typ := Ident
			if slices.Contains(cmdNamesFlat, val) {
				typ = Command
			}
			tokens = append(tokens, token{typ: typ, val: val})
		case isSpace(b):
			tokens = append(tokens, token{typ: Whitespace, val: in[skipWhile(&i, in, isSpace):i]})
		case b == '=':
			start := i
			i++
			tokens = append(tokens, token{typ: Assign, val: in[start:i]})
		case b == '"':
			isEsc := false
			var val strings.Builder
			skipWhile(&i, in, func(b byte) bool {
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
					if i > len(in) {
						return false
					}

					dec, err := strconv.ParseUint(in[start:i+1], 16, byteLen*8)
					if err != nil {
						return true
					}

					val.WriteRune(rune(dec))
				default:
					val.WriteByte(b)
				}

				return true
			})
			tokens = append(tokens, token{typ: String, val: val.String()})
			i++
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
		case Whitespace:
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

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}
