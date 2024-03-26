package api

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/stuff7/mcman/bitstream"
	"github.com/stuff7/mcman/slc"
)

type Cmd struct {
	tokens []token
	Run    func([]token) error
}

func (c *Cmd) run() error {
	if c.Run == nil {
		return errors.New("Unknown command")
	}
	return c.Run(c.tokens)
}

type commandType int

type command struct {
	typ     commandType
	aliases []string
}

const (
	CmdSet commandType = iota
	CmdAdd
	CmdSearch
	CmdHelp
	CmdDebug
	CmdVersion
	CmdQuit
)

var cmdNames = []command{
	{CmdHelp, []string{"help", "h"}},
	{CmdAdd, []string{"add"}},
	{CmdSet, []string{"set", "global"}},
	{CmdSearch, []string{"search", "find", "fn"}},
	{CmdDebug, []string{"dbg", "debug"}},
	{CmdVersion, []string{"versions"}},
	{CmdQuit, []string{"q", "qa", "quit", "exit"}},
}
var cmdNamesFlat = slc.Flatten(slc.Map(cmdNames, func(c command) []string { return c.aliases }))

func (c *cli) parseCmd(tokens []token) (Cmd, []token) {
	var i int
	var cmdx Cmd
	cmd := nextNonSpaceToken(tokens, &i)
	if cmd == nil || cmd.typ != Unknown {
		return cmdx, tokens
	}

	if ok := nextNonSpaceToken(tokens, &i); ok != nil {
		cmdx.tokens = tokens[i-1:]
	}

	var parseKeywords func([]token) []token
	for _, cmdN := range cmdNames {
		if slices.Contains(cmdN.aliases, cmd.val) {
			cmd.typ = Command

			switch cmdN.typ {
			case CmdSearch:
				cmdx.Run = c.searchCmd
			case CmdAdd:
				parseKeywords = addCmdKwords
				cmdx.Run = c.addCmd
			case CmdSet:
				parseKeywords = c.queryCmdKwords
				cmdx.Run = c.setQueryCmd
			case CmdHelp:
				cmdx.Run = c.helpCmd
			case CmdDebug:
				cmdx.Run = c.debugCmd
			case CmdVersion:
				parseKeywords = versionCmdKwords
				cmdx.Run = c.versionCmd
			case CmdQuit:
				cmdx.Run = c.quitCmd
			}

			if parseKeywords != nil && len(cmdx.tokens) != 0 {
				cmdx.tokens = parseKeywords(cmdx.tokens)
				tokens = slices.Concat(tokens[:i-1], cmdx.tokens)
			}
		} else {
			cmd.keywords = cmdNamesFlat
		}
	}

	return cmdx, tokens
}

func findClosest(in string, aliases []string) *string {
	if len(in) == 0 {
		if a := slc.Get(aliases, 0); a != nil {
			return a
		}
		return nil
	}

	var closest *string
	if slices.ContainsFunc(aliases, func(a string) bool {
		trimmed, found := strings.CutPrefix(a, in)
		if found {
			closest = &trimmed
		}
		return found
	}) {
		return closest
	}
	return nil
}

func (c *cli) addCmd(tokens []token) error {
	if len(tokens) == 0 {
		return errors.New("Usage: add <option> [optionValue]\noptions:\n\tsearch <string>\n\tid <number>")
	}

	var prevT *token
	var i int
	for {
		t := nextNonSpaceToken(tokens, &i)
		if t == nil {
			break
		}

		if prevT != nil && prevT.typ == Keyword {
			switch prevT.val {
			case "search":
				if t.typ != String {
					return errors.New("Invalid search value. Expected a string")
				}

				mods, err := searchMods(t.parseString(), c.query)
				if err != nil {
					return err
				}

				if len(mods) == 0 {
					return errors.New("No mods found")
				}

				mod := &mods[0]
				fmt.Printf("Found: %#+v\n", mod)
				continue
			case "id":
				if t.typ != Number {
					return errors.New("Invalid mod id value. Expected a number")
				}

				mod, err := getModFiles(t.parseNumber(), c.query)
				if err != nil {
					return err
				}

				fmt.Printf("Found: %#+v\n", mod)
				continue
			}
		}

		prevT = t
	}

	return nil
}

func (c *cli) versionCmd(tokens []token) error {
	if len(tokens) == 0 {
		fmt.Printf("%#+v\n", c.versions)
		return nil
	}

	if tokens[0].typ == Keyword {
		versions, err := getVersions()
		if err != nil {
			return err
		}

		if len(versions) == len(c.versions) {
			fmt.Printf("%sUp to date%s\n", clr(46), RESET)
			return nil
		}

		var mapped []string
		var bs bitstream.Bitstream
		major := nextMajor
		minor := 0
		bs.WriteBits(0, 1) // Allocate 1 bitflag to indicate if there's an even number of versions
		for i := 0; i < len(versions); i++ {
			v := &versions[i]
			if v.Version == memVersions[0] {
				break
			}

			mapped = append(mapped, v.Version)
			idx := strings.LastIndex(v.Version, ".")
			if idx < 2 {
				idx = len(v.Version) - 1
			}

			curr, err := strconv.Atoi(v.Version[2:idx])
			fmt.Printf("V: %s %#+v %d\n", v.Version, v.Version[2:idx], idx)
			if err != nil {
				return err
			}

			if major != curr {
				major = curr
				bs.WriteBits(minor, 4)
				minor = 0
				continue
			}
			minor++
		}
		bs.SetBit((major-nextMajor)&1 == 0, 0)
		fmt.Printf("%#+v\n", bs.String())
		bs.SaveToDisk("versions")
		c.versions = append(mapped, c.versions...)
	}

	return nil
}

func (c *cli) helpCmd(_ []token) error {
	println(HELP)
	return nil
}

func (c *cli) quitCmd(_ []token) error {
	c.Running = false
	return nil
}

func (c *cli) searchCmd(tokens []token) error {
	var search string
	if len(tokens) != 0 && tokens[0].typ == String {
		search = tokens[0].parseString()
	} else {
		search = joinTokens(tokens)
	}

	mods, err := searchMods(search, c.query)
	if err != nil {
		return err
	}
	for _, mod := range mods {
		fmt.Printf("[%sID: %s%d%s] %s\n%sDownloads: %s%d\n%s%s\n\n", clr(218), clr(194), mod.ID, RESET, mod.Name, clr(218), clr(194), mod.DownloadCount, RESET, mod.Summary)
	}
	return nil
}

func (c *cli) setQueryCmd(tokens []token) error {
	if len(tokens) == 0 {
		fmt.Println(c.query)
		return nil
	}

	for i := 0; i < len(tokens); i++ {
		k := nextNonSpaceToken(tokens, &i)
		if k.typ == Ident {
			v := nextNonSpaceToken(tokens, &i)
			if v == nil {
				return errors.New("Missing value")
			}

			switch k.val {
			case "gameVersion":
				if v.typ != Keyword {
					return errors.New(fmt.Sprintf("Invalid value %+v", v))
				}
				c.query.GameVersion = v.val
			case "modLoader":
				if v.typ != Keyword {
					return errors.New("Invalid value")
				}
				c.query.ModLoader = slices.Index(modLoaderKeywords, v.val)
			}
		} else {
			return errors.New(fmt.Sprintf("Unknown query key %s", k.val))
		}
	}

	fmt.Println("Query Updated:", c.query)
	return nil
}

func (c *cli) debugCmd([]token) error {
	c.dbg = !c.dbg
	if c.dbg {
		println("Debug enabled")
	} else {
		println("Debug disabled")
	}
	return nil
}
