package api

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

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
	typ         commandType
	aliases     []string
	description string
}

func newCommand(typ commandType, desc string, aliases ...string) command {
	return command{typ, aliases, desc}
}

const (
	CmdSet commandType = iota
	CmdAdd
	CmdList
	CmdSearch
	CmdHelp
	CmdDebug
	CmdVersion
	CmdQuit
)

var commands = []command{
	newCommand(CmdHelp, "Print this table", "help", "h"),
	newCommand(CmdAdd, "Add a new mod", "add"),
	newCommand(CmdList, "List all the mods", "list", "ls"),
	newCommand(CmdSet, "Set global query parameters", "set", "global"),
	newCommand(CmdSearch, "Search mods", "search", "find", "fn"),
	newCommand(CmdDebug, "Enable/Disable debug logs", "debug", "dbg"),
	newCommand(CmdVersion, "Update saved versions", "versions"),
	newCommand(CmdQuit, "Quit", "quit", "qa", "q", "exit"),
}
var cmdNames = slc.Flatten(slc.Map(commands, func(c command) []string { return c.aliases }))

func (c *cli) parseCmd(tokens []token) (Cmd, []token) {
	var i int
	var cmd Cmd
	t := nextNonSpaceToken(tokens, &i)
	if t == nil || t.typ != Unknown {
		return cmd, tokens
	}

	if ok := nextNonSpaceToken(tokens, &i); ok != nil {
		cmd.tokens = tokens[i-1:]
	}

	var parseKeywords func([]token) []token
	for _, cmdN := range commands {
		if slices.Contains(cmdN.aliases, t.val) {
			t.typ = Command

			switch cmdN.typ {
			case CmdSearch:
				cmd.Run = c.searchCmd
			case CmdAdd:
				parseKeywords = addCmdKwords
				cmd.Run = c.addCmd
			case CmdList:
				cmd.Run = c.listCmd
			case CmdSet:
				parseKeywords = c.queryCmdKwords
				cmd.Run = c.setQueryCmd
			case CmdHelp:
				cmd.Run = c.helpCmd
			case CmdDebug:
				cmd.Run = c.debugCmd
			case CmdVersion:
				parseKeywords = versionCmdKwords
				cmd.Run = c.versionCmd
			case CmdQuit:
				cmd.Run = c.quitCmd
			}

			if parseKeywords != nil && len(cmd.tokens) != 0 {
				cmd.tokens = parseKeywords(cmd.tokens)
				tokens = slices.Concat(tokens[:i-1], cmd.tokens)
			}
		}
	}

	if t.typ != Command && cmd.Run == nil {
		t.keywords = cmdNames
	}

	return cmd, tokens
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

func (c *cli) listCmd([]token) error {
	fmt.Printf("Found %s%d%s mods\n", clr(49), len(c.mods), RESET)

	var sb strings.Builder
	for i, m := range c.mods {
		sb.WriteString(fmt.Sprintf("\n[idx: %s%d%s] %s%s%s", clr(157), i, RESET, clr(214)+BOLD, m.name, RESET))
		sb.WriteString(fmt.Sprintf(" (%s%s %s%s%s)\n", clr(228)+BOLD, modLoaderKeywords[m.modLoader], clr(231), m.gameVersion, RESET))
		sb.WriteString(fmt.Sprintf("Download: %s%s%s\n", clr(123)+BOLD, m.downloadUrl, RESET))
		sb.WriteString(fmt.Sprintf("Uploaded: %s%s%s\n", clr(219)+BOLD, m.uploaded.Format(time.RFC822), RESET))
	}

	println(sb.String())
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

				if err := c.addMod(t.parseString(), false); err != nil {
					return err
				}
				continue
			case "id":
				if t.typ != Number {
					return errors.New("Invalid mod id value. Expected a number")
				}

				if err := c.addMod(t.parseNumber(), false); err != nil {
					return err
				}
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

var helpTable string

func (c *cli) helpCmd(tokens []token) error {
	if len(helpTable) != 0 {
		println(helpTable)
		return nil
	}

	rows := make([][3]string, len(commands)+2)
	rows[0] = [3]string{"Command", "Description", " "}
	rows[1][2] = "-"
	maxLn := []int{len(rows[0][0]), len(rows[0][1])}
	for j, cmd := range commands {
		rows[j+2] = [3]string{cmd.aliases[0], cmd.description, " "}
		for i, n := range rows[j+2][:2] {
			l := &maxLn[i]
			if n := len(n); n > *l {
				*l = n
			}
		}
	}

	var sb strings.Builder
	for _, row := range rows {
		for i, s := range row[:2] {
			l := &maxLn[i]
			sb.WriteString(fmt.Sprintf("|%s%s%s", row[2], s, strings.Repeat(row[2], *l-len(s)+1)))
		}
		sb.WriteString("|\n")
	}

	helpTable = sb.String()
	return c.helpCmd(tokens)
}

func (c *cli) quitCmd([]token) error {
	c.Running = false
	return c.saveMods()
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
