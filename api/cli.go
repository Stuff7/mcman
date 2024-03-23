package api

import (
	"fmt"

	"github.com/stuff7/mcman/readln"
)

type cli struct {
	query   searchQuery
	Running bool
	prompt  string
	dbg     bool
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

		if c.dbg {
			fmt.Printf("Cmd\n%#+v\n", tokens)
		}
	}

	println("Quit")
	return nil
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
