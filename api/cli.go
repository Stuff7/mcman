package api

import (
	"fmt"
	"os"
	"slices"

	"github.com/stuff7/mcman/bitstream"
	"github.com/stuff7/mcman/readln"
)

type cli struct {
	query    searchQuery
	Running  bool
	prompt   string
	dbg      bool
	versions []string
}

func NewCli(prompt string) *cli {
	return &cli{Running: true, prompt: prompt}
}

func (c *cli) Run() error {
	fmt.Printf("%s\nPress q to quit\n", LOGO)
	var history []string
	var tokens []token
	var cmd Cmd

	if err := c.loadFiles(); err != nil {
		return err
	}

	for c.Running {
		_, err := readln.PushLn(c.prompt, &history, func(k readln.Key, s *string, i *int) string {
			tokens = tokenize(*s)
			cmd, tokens = c.parseCmd(tokens)
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

func (c *cli) loadFiles() error {
	c.versions = nil
	versions, err := os.ReadFile("versions")
	if err != nil {
		c.versions = memVersions
		return nil
	}

	bs := bitstream.FromBuffer(versions)
	var bitpos int
	major := nextMajor
	evenVersions, err := bs.ReadBits(&bitpos, 1)
	if err != nil {
		return err
	}

	for {
		v, err := bs.ReadBits(&bitpos, 4)
		if err != nil {
			break
		}
		c.versions = append([]string{fmt.Sprintf("1.%d", major)}, c.versions...)
		for minor := 1; minor <= v; minor++ {
			c.versions = append([]string{fmt.Sprintf("1.%d.%d", major, minor)}, c.versions...)
		}
		major++
	}

	if evenVersions != 0 && (major-nextMajor)&1 != 0 {
		c.versions = slices.Delete(c.versions, 0, 1)
	}
	c.versions = append(c.versions, memVersions...)

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
