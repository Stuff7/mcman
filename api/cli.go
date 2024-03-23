package api

import (
	"errors"
	"fmt"
	"slices"
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
	fmt.Printf("%s\nPress q to quit\n%s\n", LOGO, HELP)
	var history []string
	var tokens []token
	var cmd Cmd

	for c.Running {
		_, err := readln.PushLn(c.prompt, &history, func(k readln.Key, s *string, i *int) string {
			tokens = tokenize(*s)
			cmd = c.parseCmd(tokens)
			return renderTokens(tokens, k, s, i)
		})

		if err != nil {
			return err
		}

		if err := cmd.run(); err != nil {
			fmt.Printf("%s  %s%s\n", clr(220), err, RESET)
		}
	}

	println("Quit")
	return nil
}

type command int

const (
	CmdSet command = iota
	CmdSearch
	CmdHelp
	CmdQuit
)

var cmdNames = [][]string{
	{"set", "global"},
	{"search", "find", "fn"},
	{"help", "h"},
	{"q", "qa", "quit", "exit"},
}
var cmdNamesFlat = arrFlat(cmdNames)

func (c *cli) parseCmd(tokens []token) Cmd {
	var i int
	var cmdx Cmd
	cmd := nextNonSpaceToken(tokens, &i)
	if cmd == nil || cmd.typ != Unknown {
		return cmdx
	}

	if ok := nextNonSpaceToken(tokens, &i); ok != nil {
		cmdx.tokens = tokens[i-1:]
	}

	for cmdName, aliases := range cmdNames {
		if slices.Contains(aliases, cmd.val) {
			cmd.typ = Command
			switch command(cmdName) {
			case CmdSearch:
				cmdx.Run = c.searchCmd
			case CmdSet:
				autocomplete(cmdx.tokens, Ident, queryFields)
				cmdx.Run = c.setQueryCmd
			case CmdHelp:
				cmdx.Run = c.helpCmd
			case CmdQuit:
				cmdx.Run = c.quitCmd
			}
		} else {
			cmd.keywords = cmdNamesFlat
		}
	}

	return cmdx
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
		fmt.Printf("[ID: %d] %s\nDownloads: %d\n%s\n\n", mod.ID, mod.Name, mod.DownloadCount, mod.Summary)
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
				if v.typ != String {
					return errors.New(fmt.Sprintf("Invalid value %+v", v))
				}
				c.query.GameVersion = v.parseString()
			case "modLoader":
				if v.typ != Number {
					return errors.New("Invalid value")
				}
				c.query.ModLoader = ModLoaderType(v.parseNumber())
			}
		} else {
			return errors.New(fmt.Sprintf("Unknown query key %s", k.val))
		}
	}

	fmt.Println("Query Updated:", c.query)
	return nil
}

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

const HELP = `
| Command           | Description       |
|-------------------|-------------------|
| help              | Print this table  |
| search <string>   | Search mods       |
| set <key> <value> | Sets global query |
| quit              | Quit              |
`
