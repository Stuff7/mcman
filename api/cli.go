package api

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/stuff7/mcman/readln"
)

type cli struct {
	query   searchQuery
	Running bool
	prompt  string
}

func NewCli(prompt string) *cli {
	return &cli{query: searchQuery{ModLoader: Forge}, Running: true, prompt: prompt}
}

func (c *cli) Run() error {
	fmt.Print(LOGO)
	var history []string

	for c.Running {
		input, err := readln.PushLn(c.prompt, &history, c.highlightInput)
		if err != nil {
			return err
		}

		cmd := c.parseCmd(input)
		if err := cmd.run(); err != nil {
			fmt.Printf("%s  %s%s\n", clr(220), err, RESET)
		}
	}

	fmt.Println("Quit")
	return nil
}

type command int

const (
	CmdSearch command = iota
	CmdSet
	CmdQuit
)

var cmdNames = [][]string{
	{"search"},
	{"set", "global"},
	{"q", "qa"},
}

func (c *cli) parseCmd(s string) Cmd {
	args := strings.SplitN((s), " ", 2)
	if cmdStr := arrGet(args, 0); cmdStr != nil {
		cmd := Cmd{Name: *cmdStr}
		if len(args) > 1 {
			cmd.Args = args[1]
		}

		for cmdName, aliases := range cmdNames {
			if slices.Contains(aliases, cmd.Name) {
				switch command(cmdName) {
				case CmdSearch:
					cmd.Run = c.searchCmd
				case CmdSet:
					cmd.Run = c.setQueryCmd
				case CmdQuit:
					cmd.Run = c.quitCmd
				}
			}
		}

		if len(cmd.Name) != 0 && len(cmd.Args) == 0 && cmd.Run == nil {
			for _, aliases := range cmdNames {
				var closest *string
				if slices.ContainsFunc(aliases, func(a string) bool {
					trimmed, found := strings.CutPrefix(a, cmd.Name)
					if found {
						closest = &trimmed
					}
					return found
				}) {
					cmd.Closest = closest
					break
				}
			}
		}

		return cmd
	}

	return Cmd{Name: s}
}

func (c *cli) highlightInput(s string) string {
	cmd := c.parseCmd(s)
	if cmd.Closest != nil {
		return fmt.Sprintf("%s%s%s%s", cmd.Name, clr(248), *cmd.Closest, RESET)
	}

	if cmd.Run == nil {
		return s
	}

	return fmt.Sprintf("%s%s%s %s%s", clr(154), cmd.Name, clr(248), cmd.Args, RESET)
}

func (c *cli) quitCmd(_ string) error {
	c.Running = false
	return nil
}

func (c *cli) searchCmd(in string) error {
	args := strings.Split(in, " ")
	search := arrGet(args, 0)
	if search == nil {
		return errors.New("Missing search")
	}
	mods, err := searchMods(*search, c.query)
	if err != nil {
		return err
	}
	for _, mod := range mods {
		fmt.Printf("[ID: %d] %s\nDownloads: %d\n%s\n", mod.ID, mod.Name, mod.DownloadCount, mod.Summary)
	}
	return nil
}

func (c *cli) setQueryCmd(in string) error {
	if len(in) == 0 {
		fmt.Println(c.query)
		return nil
	}
	args := strings.SplitN(in, " ", 2)
	key := arrGet(args, 0)
	val := arrGet(args, 1)
	if val == nil {
		return errors.New("Usage:\nglobal <key> <val>\n\tkey: \"gameVersion\" | \"modLoader\"")
	}

	switch *key {
	case "gameVersion":
		c.query.GameVersion = *val
	case "modLoader":
		modLoader, err := strconv.Atoi(*val)
		if err != nil {
			return err
		}
		if modLoader >= int(InvalidModLoader) {
			return errors.New(fmt.Sprintf("Invalid modLoader value %d", modLoader))
		}

		c.query.ModLoader = ModLoaderType(modLoader)
		fmt.Printf("Mod Loader: %s\n", c.query.ModLoader.Name())
	default:
		return errors.New(fmt.Sprintf("Unknown query key \"%s\"", *key))
	}

	fmt.Println("Query Updated:", c.query)
	return nil
}

type Cmd struct {
	Closest *string
	Name    string
	Args    string
	Run     func(args string) error
}

func (c *Cmd) run() error {
	if c.Run != nil {
		return c.Run(c.Args)
	}
	return errors.New(fmt.Sprintf("Unknown command \"%s\"", c.Name))
}

type searchQuery struct {
	GameVersion string        `query:"gameVersion"`
	ModLoader   ModLoaderType `query:"modLoader"`
}

func (q searchQuery) String() string {
	t := reflect.TypeOf(q)
	v := reflect.ValueOf(q)
	sep := '?'
	var query strings.Builder
	for i := range t.NumField() {
		f := t.Field(i)
		tag := f.Tag.Get("query")
		val := v.Field(i).Interface()
		var strVal string
		switch val.(type) {
		case string:
			strVal = val.(string)
			if strVal == "" {
				continue
			}
		case ModLoaderType:
			val := val.(ModLoaderType)
			strVal = fmt.Sprint(val)
		}
		query.WriteRune(sep)
		query.WriteString(tag)
		query.WriteRune('=')
		query.WriteString(strVal)
		sep = '&'
	}

	return query.String()
}

func arrGet[T any](arr []T, idx int) *T {
	if idx < 0 || idx >= len(arr) {
		return nil
	}
	return &arr[idx]
}

func clr(id byte) string {
	return fmt.Sprintf("\x1b[38;5;%dm", id)
}

const RESET = "\x1b[0m"

const LOGO = `
 ███▄ ▄███▓ ▄████▄   ███▄ ▄███▓ ▄▄▄       ███▄    █ 
▓██▒▀█▀ ██▒▒██▀ ▀█  ▓██▒▀█▀ ██▒▒████▄     ██ ▀█   █ 
▓██    ▓██░▒▓█    ▄ ▓██    ▓██░▒██  ▀█▄  ▓██  ▀█ ██▒
▒██    ▒██ ▒▓▓▄ ▄██▒▒██    ▒██ ░██▄▄▄▄██ ▓██▒  ▐▌██▒
▒██▒   ░██▒▒ ▓███▀ ░▒██▒   ░██▒ ▓█   ▓██▒▒██░   ▓██░
░ ▒░   ░  ░░ ░▒ ▒  ░░ ▒░   ░  ░ ▒▒   ▓▒█░░ ▒░   ▒ ▒ 
░  ░      ░  ░  ▒   ░  ░      ░  ▒   ▒▒ ░░ ░░   ░ ▒░
░      ░   ░        ░      ░     ░   ▒      ░   ░ ░ 
       ░   ░ ░             ░         ░  ░         ░ 
           ░                                        
`
