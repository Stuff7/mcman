package api

import (
	"slices"
	"strconv"
	"strings"

	"github.com/stuff7/mcman/readln"
	"github.com/stuff7/mcman/slc"
)

type tokenType byte

const (
	Unknown tokenType = iota
	Command
	Ident
	String
	Number
	Symbol
	Keyword
	Space
)

type token struct {
	typ      tokenType
	val      string
	lst      int
	keywords []string
}

func newToken(typ tokenType, in string, i *int, cond func(byte) bool) token {
	fst := skipWhile(i, in, cond)
	return token{typ: typ, val: in[fst:*i], lst: *i}
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

func (c *cli) parseVersion(tokens []token, i int) []token {
	j := i
	var b strings.Builder
	typ := Number
	for j-i < 5 && j < len(tokens) {
		t := &tokens[j]
		j++
		if t.typ != typ || (t.typ == Symbol && t.val != ".") {
			break
		}

		if typ == Number {
			typ = Symbol
		} else {
			typ = Number
		}

		b.WriteString(t.val)
	}

	t := token{val: b.String(), lst: tokens[j-1].lst}
	if slices.Contains(c.versions, t.val) {
		t.typ = Keyword
	} else {
		t.typ = Symbol
		t.keywords = c.versions
	}

	return slices.Replace(tokens, i, j, t)
}

func remCmdKwords(tokens []token) []token {
	var i int
	for {
		t := nextNonSpaceToken(tokens, &i)
		if t == nil || t.typ != Unknown {
			break
		}

		t.autocomplete(Keyword, []string{"search", "id", "index"})
	}

	return tokens
}

func addCmdKwords(tokens []token) []token {
	var i int
	for {
		t := nextNonSpaceToken(tokens, &i)
		if t == nil || t.typ != Unknown {
			break
		}

		t.autocomplete(Keyword, []string{"search", "id"})
	}

	return tokens
}

func (c *cli) queryCmdKwords(tokens []token) []token {
	var t, prevT *token
	var i int
	for {
		t = nextNonSpaceToken(tokens, &i)
		if t == nil {
			break
		}

		if prevT != nil && prevT.typ == Ident {
			switch prevT.val {
			case "modLoader":
				t.autocomplete(Keyword, modLoaderKeywords)
			case "gameVersion":
				i--
				tokens = c.parseVersion(tokens, i)
			}
		} else if t.typ == Unknown {
			t.autocomplete(Ident, queryFields)
		}

		prevT = t
	}

	return tokens
}

func versionCmdKwords(tokens []token) []token {
	var t *token
	var i int

	for {
		t = nextNonSpaceToken(tokens, &i)
		if t == nil || t.typ != Unknown {
			break
		}
		t.autocomplete(Keyword, []string{"update"})
	}

	return tokens
}

func (t *token) autocomplete(to tokenType, keywords []string) {
	if slices.Contains(keywords, t.val) {
		t.typ = to
	} else {
		t.keywords = keywords
	}
}

func nextNonSpaceToken(tokens []token, i *int) *token {
	for *i < len(tokens) && tokens[*i].typ == Space {
		*i++
	}

	if len(tokens) == *i {
		return nil
	}
	t := &tokens[*i]
	*i++

	return t
}

func joinTokens(tokens []token) string {
	var b strings.Builder
	for i := 0; i < len(tokens); i++ {
		b.WriteString(tokens[i].val)
	}
	return b.String()
}

func tokenize(in string) []token {
	var tokens []token
	for i := 0; i < len(in); {
		b := in[i]
		switch {
		case isDigit(b):
			tokens = append(tokens, newToken(Number, in, &i, isDigit))
		case isAlpha(b):
			tokens = append(tokens, newToken(Unknown, in, &i, isAlphanumeric))
		case readln.IsSpace(b):
			tokens = append(tokens, newToken(Space, in, &i, readln.IsSpace))
		case b == '"':
			isEsc := false
			tokens = append(tokens, newToken(String, in, &i, func(b byte) bool {
				if !isEsc {
					switch b {
					case '\\':
						isEsc = true
					case '"':
						i++
						if i >= len(in) {
							i = len(in)
						}

						return false
					}
				}
				return true
			}))
		default:
			tokens = append(tokens, newToken(Symbol, in, &i, func(byte) bool { return false }))
		}
	}

	l := slc.Last(tokens)
	if l == nil {
		tokens = append(tokens, token{Unknown, "", 0, nil})
	} else if l.typ == Space {
		tokens = append(tokens, token{Unknown, "", l.lst, nil})
	}

	return tokens
}

func renderTokens(tokens []token, k readln.Key, s *string, p *int) string {
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
		case Keyword:
			b.WriteString(clr(85))
		case Symbol:
			b.WriteString(clr(213))
		default:
			b.WriteString(RESET)
		}

		b.WriteString(t.val)
		if t.lst == *p && t.keywords != nil {
			closest := findClosest(t.val, t.keywords)
			if closest == nil {
				break
			}

			if k == readln.Tab {
				*s = (*s)[:*p] + *closest + (*s)[*p:]
				*p += len(*closest)
				break
			}

			b.WriteString(clr(248) + *closest + RESET)
		}
	}

	b.WriteString(RESET)
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
