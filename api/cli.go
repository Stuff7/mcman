package api

import (
	"errors"
	"fmt"
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
var cmdNamesFlat = arrFlat(cmdNames)

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
					cmd.HighlightArgs = c.searchHl
				case CmdSet:
					cmd.Run = c.setQueryCmd
					cmd.HighlightArgs = c.setQueryHl
				case CmdQuit:
					cmd.Run = c.quitCmd
				}
			}
		}

		return cmd
	}

	return Cmd{Name: s}
}

func (c *cli) highlightInput(k readln.Key, s *string, p *int) string {
	cmd := c.parseCmd(*s)

	if cmd.Run == nil {
		closest := findClosest(cmd.Name, cmdNamesFlat)
		if closest == nil {
			return *s
		}

		if k == readln.Tab {
			*s += *closest
			*p = len(*s)
			return c.highlightInput(readln.NA, s, p)
		}

		return cmd.Name + clr(248) + *closest + RESET
	}

	args := " " + clr(103) + cmd.Args
	if cmd.HighlightArgs != nil {
		args = cmd.HighlightArgs(k, len(cmd.Name), s, p)
	}

	return clr(154) + cmd.Name + RESET + args + RESET
}

func findClosest(in string, aliases []string) *string {
	if len(in) == 0 {
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

func (c *cli) setQueryHl(key readln.Key, start int, in *string, pos *int) string {
	args := (*in)[start:]
	closest := findClosest(args[strings.LastIndex(args, " ")+1:], queryFields)
	args = hl(args, queryFields, 14)
	if closest == nil {
		return args
	}

	if key == readln.Tab {
		*in += *closest
		*pos = len(*in)
		return c.setQueryHl(readln.NA, start, in, pos)
	}

	return args + clr(248) + *closest
}

func (c *cli) searchHl(_ readln.Key, start int, in *string, _ *int) string {
	return clr(214) + (*in)[start:]
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
		fmt.Printf("[ID: %d] %s\nDownloads: %d\n%s\n\n", mod.ID, mod.Name, mod.DownloadCount, mod.Summary)
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
	Name          string
	Args          string
	Run           func(string) error
	HighlightArgs func(readln.Key, int, *string, *int) string
}

func (c *Cmd) run() error {
	if c.Run != nil {
		return c.Run(c.Args)
	}
	return errors.New(fmt.Sprintf("Unknown command \"%s\"", c.Name))
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
