package api

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/stuff7/mcman/bitstream"
	"github.com/stuff7/mcman/readln"
)

type cli struct {
	query    searchQuery
	Running  bool
	prompt   string
	dbg      bool
	versions []string
	mods     []modEntry
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
			fmt.Printf("%s%s%s\n", clr(220), err, RESET)
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

func (c *cli) saveMods() error {
	var bs bitstream.Bitstream
	for _, m := range c.mods {
		bs.WriteBits(m.id, 24)
		bs.WriteBits(m.modLoader, 4)

		idx := strings.Index(m.gameVersion[2:], ".")
		if idx < 0 {
			idx = len(m.gameVersion)
		} else {
			idx += 2
		}

		major, err := strconv.Atoi(m.gameVersion[2:idx])
		if err != nil {
			return err
		}
		bs.WriteBits(major, 5)

		minor, err := strconv.Atoi(m.gameVersion[idx:])
		if err != nil {
			bs.WriteBits(0, 4)
		} else {
			bs.WriteBits(minor, 4)
		}

		if err := bs.WritePascalString(m.name); err != nil {
			return err
		}

		if err := bs.WritePascalString(m.downloadUrl); err != nil {
			return err
		}

		bs.WriteBits64(m.uploaded.Unix(), 64)
	}

	println(bs.String())
	return bs.SaveToDisk("modlist")
}

const RESET = "\x1b[0m"
const BOLD = "\x1b[1m"

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
