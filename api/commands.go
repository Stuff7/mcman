package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

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
	CmdRem
	CmdImport
	CmdExport
	CmdClear
	CmdDownload
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
	newCommand(CmdRem, "Remove a mod", "remove", "rm", "rem", "del"),
	newCommand(CmdImport, "Import mods from json file { id: string }[]", "import"),
	newCommand(CmdExport, "Export mods to json file", "export"),
	newCommand(CmdClear, "Clear the terminal", "clear"),
	newCommand(CmdDownload, "Download all mods", "download", "dwn"),
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
			case CmdRem:
				parseKeywords = remCmdKwords
				cmd.Run = c.remCmd
			case CmdImport:
				cmd.Run = c.importCmd
			case CmdExport:
				cmd.Run = c.exportCmd
			case CmdClear:
				cmd.Run = c.clearCmd
			case CmdDownload:
				cmd.Run = c.downloadCmd
			case CmdList:
				parseKeywords = addCmdKwords
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

func (c *cli) clearCmd([]token) error {
	fmt.Printf("\x1b[2J\x1b[1;1H%s", LOGO)
	return nil
}

func (c *cli) exportCmd(tokens []token) error {
	var out string
	if len(tokens) == 0 {
		out = "mods.json"
	} else {
		var i int
		t := nextNonSpaceToken(tokens, &i)
		if t != nil && t.typ == String {
			out = t.parseString()
		}
	}

	data, err := json.Marshal(c.mods)
	if err != nil {
		return err
	}

	if err := os.WriteFile(out, data, 0666); err != nil {
		return err
	}

	fmt.Printf("Exported %d mods to %+v\n", len(c.mods), out)

	return nil
}

func (c *cli) importCmd(tokens []token) error {
	if len(tokens) == 0 {
		return errors.New("Usage: import <file.json>")
	}

	var i int
	t := nextNonSpaceToken(tokens, &i)
	if t == nil {
		return errors.New("Missing import file path")
	}

	if t.typ != String {
		return errors.New("Invalid argument. Expected a string")
	}

	if err := c.importMods(t.parseString()); err != nil {
		return err
	}

	return nil
}

func (c *cli) listCmd(tokens []token) error {
	if len(tokens) == 0 {
		listMods(c.mods, nil)
		return nil
	}

	var i int
	t := nextNonSpaceToken(tokens, &i)
	v := nextNonSpaceToken(tokens, &i)

	if t == nil || v == nil || t.typ != Keyword {
		return errors.New("Missing argument")
	}

	var search string
	switch t.val {
	case "search":
		if v.typ != String {
			return errors.New("Invalid argument type")
		}
		search = v.parseString()
		listMods(c.mods, func(m modEntry) bool {
			return strings.Contains(m.Name, search) || slc.FuzzyStringCompare(m.Name, search) < 0.8
		})
	case "id":
		if v.typ != Number {
			return errors.New("Invalid argument type")
		}
		search = strconv.Itoa(v.parseNumber())
		listMods(c.mods, func(m modEntry) bool {
			return strings.Contains(strconv.Itoa(m.Id), search)
		})
	}

	fmt.Printf("No mods found that matched %+v: %+v", t.val, search)
	return nil
}

func (c *cli) remCmd(tokens []token) error {
	if len(tokens) == 0 {
		return errors.New("Usage: rem <option> [optionValue]\noptions:\n\tsearch <string>\n\tid <number>\n\tindex <number>")
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

				if err := c.remMod(t.parseString(), false); err != nil {
					return err
				}
				continue
			case "id", "index":
				if t.typ != Number {
					return errors.New("Invalid mod id value. Expected a number")
				}

				if err := c.remMod(t.parseNumber(), prevT.val == "index"); err != nil {
					return err
				}
				continue
			}
		}

		prevT = t
	}

	return nil
}

func (c *cli) downloadCmd(tokens []token) error {
	if len(tokens) == 0 {
		return errors.New("Usage: download [directory]")
	}
	var i int
	var dir string
	t := nextNonSpaceToken(tokens, &i)
	if t != nil && t.typ == String {
		dir = t.parseString()
	}

	var txt string
	for i, m := range c.mods {
		downloaded, err := downloadFile(m.DownloadUrl, filepath.Join(dir, url.QueryEscape(m.Name)))
		if err != nil {
			txt = fmt.Sprintf("%sdownload failed\t%s", clr(218), err)
			time.Sleep(time.Second)
		} else if downloaded {
			txt = clr(48) + "downloaded"
			time.Sleep(time.Second)
		} else {
			txt = clr(45) + "already exists"
		}
		fmt.Printf("[%s%03d%s / %s%03d%s] %s%#+v %s\t%s\n", clr(156), i+1, RESET, clr(156), len(c.mods), RESET, BOLD, m.Name, txt, RESET)
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
		fmt.Printf(
			"Found %s%d%s versions locally (Run %sversion update%s to update)\n%s\n",
			clr(157),
			len(c.versions),
			RESET,
			BOLD,
			RESET,
			strings.Join(c.versions, " | "),
		)
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

		var newVersions []string
		for _, v := range versions {
			if v.Version == memVersions[0] {
				break
			}

			newVersions = append(newVersions, v.Version)
		}

		c.versions = append(newVersions, c.versions...)
		if err := c.saveCfg(); err != nil {
			return err
		}
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

func (c *cli) quitCmd(tokens []token) error {
	c.Running = false
	var i int
	if t := nextNonSpaceToken(tokens, &i); t != nil && t.typ == Symbol && t.val == "!" {
		fmt.Println("Quit without saving")
		return nil
	}

	fmt.Printf("Saved %d mods", len(c.mods))
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
					return fmt.Errorf("Invalid value %+v", v)
				}
				c.query.GameVersion = v.val
			case "modLoader":
				if v.typ != Keyword {
					return errors.New("Invalid value")
				}
				c.query.ModLoader = slices.Index(modLoaderKeywords, v.val)
			}
		} else {
			return fmt.Errorf("Unknown query key %s", k.val)
		}
	}

	fmt.Println("Query Updated:", c.query)
	return c.saveCfg()
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
