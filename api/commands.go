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
	"github.com/stuff7/mcman/storage"
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
	CmdModpack
	CmdProfile
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
	newCommand(CmdRem, "Remove a mod or a profile", "remove", "rm", "rem", "del"),
	newCommand(CmdModpack, "Add a modpack", "modpack"),
	newCommand(CmdProfile, "Change profile", "profile"),
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
			case CmdModpack:
				cmd.Run = c.modpackCmd
			case CmdProfile:
				parseKeywords = profileCmdKwords
				cmd.Run = c.profileCmd
			case CmdImport:
				cmd.Run = c.importCmd
			case CmdExport:
				cmd.Run = c.exportCmd
			case CmdClear:
				parseKeywords = clearCmdKwords
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

func (c *cli) clearCmd(tokens []token) error {
	i := 0
	t := nextNonSpaceToken(tokens, &i)
	if t != nil && t.typ == Keyword {
		dir := t.parseString()
		storage.RemoveAll(c.profilePath(dir))
		fmt.Printf("Removed %#+v directory\n", dir)
	} else {
		fmt.Printf("\x1b[2J\x1b[1;1H%s", LOGO)
	}

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

	if err := storage.WriteFile(out, data); err != nil {
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
		return errors.New("Usage: rem <option> [optionValue]\noptions:\n\tprofile <string>\n\tsearch <string>\n\tid <number>\n\tindex <number>")
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
			case "profile":
				if t.typ != String && t.typ != Unknown {
					return errors.New("Invalid search value. Expected a string")
				}

				prof := t.parseString()
				storage.RemoveAll(filepath.Join("profiles", prof))
				fmt.Printf("Removed profile %#+v\n", prof)

				if prof == c.profile {
					c.profile = "default"
					c.loadProfile()
				}

				continue
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

func (c *cli) downloadCmd([]token) error {
	fmt.Printf("%sDownloading %d %s%s\n", BOLD, len(c.mods), pluralize("mod", len(c.mods)), RESET)
	for i, m := range c.mods {
		downloaded, err := downloadFile(
			m.DownloadUrl,
			filepath.Join(c.profilePath("mods"), url.QueryEscape(m.Name)),
			fmt.Sprintf("[%s%03d%s / %s%03d%s] %s Downloading%s", clr(156), i+1, RESET, clr(156), len(c.mods), RESET, BOLD, RESET),
		)
		if err != nil {
			fmt.Printf("\n%s%s%s%s download failed\t%s", BOLD, m.Name, RESET, clr(218), err)
		} else if !downloaded {
			fmt.Printf("\n%s%s%s%s already exists\t%s", BOLD, m.Name, RESET, clr(218), err)
		}
	}
	fmt.Println()

	return nil
}

func (c *cli) profileCmd(tokens []token) error {
	if len(tokens) == 0 {
		return errors.New("Usage: profile [name]")
	}

	i := 0
	t := nextNonSpaceToken(tokens, &i)
	oldProfile := c.profile
	if t != nil && (t.typ == String || t.typ == Unknown || t.typ == Keyword) {
		c.profile = t.parseString()
	} else {
		c.profile = "default"
	}

	if oldProfile == c.profile {
		return nil
	}

	return c.loadProfile()
}

func (c *cli) modpackCmd(tokens []token) error {
	if len(tokens) == 0 {
		return errors.New("Usage: modpack [modpack id]")
	}
	var i int
	var id int
	t := nextNonSpaceToken(tokens, &i)
	if t != nil && t.typ == Number {
		id = t.parseNumber()
	}

	mods, err := getModFiles(id, c.query)
	if err != nil {
		fmt.Printf("%s", err)
		return err
	}

	m := slc.Get(mods.Files, 0)
	if m == nil {
		fmt.Printf("Modpack with ID %d not found", id)
	}

	filePath := filepath.Join(c.profilePath("downloads"), url.QueryEscape(m.Name))
	downloaded, err := downloadFile(*m.DownloadURL, filePath)
	if err != nil {
		fmt.Printf("%s%#+v %sdownload failed (Reason: %s)%s\n", BOLD, m.Name, clr(218), err, RESET)
		time.Sleep(time.Second)
	} else if !downloaded {
		fmt.Printf("%s%#+v %salready exists%s\n", BOLD, m.Name, clr(45), RESET)
	}
	fmt.Println()

	modpackDir := c.profilePath("modpack")
	if c.modpackId != mods.ID {
		storage.RemoveAll(modpackDir)
		storage.RemoveAll(c.profilePath("mods"))
	}

	c.isCfModpack = true
	c.modpackId = mods.ID
	c.modpackName = m.Name

	manifestDir := filepath.Join(modpackDir, "manifest.json")
	if _, err := storage.Stat(manifestDir); err != nil && os.IsNotExist(err) {
		if err := unzip(filePath, modpackDir); err != nil {
			return err
		}
		fmt.Println()
	}

	data, err := storage.ReadFileContents(manifestDir)
	if err != nil {
		return err
	}

	manifest, err := modpackManifestUnmarshal(data)
	if err != nil {
		return err
	}

	fmt.Printf("%sFetching %d %s%s\n", BOLD, len(manifest.Files), pluralize("mod", len(c.mods)), RESET)
	c.mods = nil
	pr := ProgressBar{
		Description: "Fetching mod",
		Total:       int64(len(manifest.Files)),
	}
	for i := 0; i < len(manifest.Files); i++ {
		f := &manifest.Files[i]
		mod, err := getModFile(f, c.query)
		if err != nil {
			return err
		}
		pr.Name = mod.File.Name
		pr.Progress++
		pr.printProgress()
		c.mods = append(c.mods, entryFromModFile(&mod))
	}
	fmt.Println()

	if err := c.saveProfile(); err != nil {
		return err
	}

	return c.saveMods()
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
		if err := c.saveCache(); err != nil {
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
	return c.saveProfile()
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
