package api

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

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

func saveQuery(bs *bitstream.Bitstream, modLoader int, gameVersion string) error {
	bs.WriteBits(modLoader, 3)

	if len(gameVersion) < 2 {
		return fmt.Errorf("Invalid game version string %#+v", gameVersion)
	}

	idx := strings.Index(gameVersion[2:], ".")
	if idx < 0 {
		idx = len(gameVersion)
	} else {
		idx += 2
	}

	major, err := strconv.Atoi(gameVersion[2:idx])
	if err != nil {
		return err
	}
	bs.WriteBits(major, 5)

	minor, err := strconv.Atoi(gameVersion[idx+1:])
	if err != nil {
		bs.WriteBits(0, 4)
	} else {
		bs.WriteBits(minor, 4)
	}

	return nil
}

func readQuery(bs *bitstream.Bitstream, b *int, modLoader *int, gameVersion *string) error {
	var err error
	*modLoader, err = bs.ReadBits(b, 3)
	if err != nil {
		return err
	}

	major, err := bs.ReadBits(b, 5)
	if err != nil {
		return err
	}

	minor, err := bs.ReadBits(b, 4)
	if err != nil {
		return err
	}

	if minor == 0 {
		*gameVersion = fmt.Sprintf("1.%d", major)
	} else {
		*gameVersion = fmt.Sprintf("1.%d.%d", major, minor)
	}

	return nil
}

func (c *cli) saveCfg() error {
	var bs bitstream.Bitstream

	if err := saveQuery(&bs, c.query.ModLoader, c.query.GameVersion); err != nil {
		return err
	}

	major := nextMajor
	minor := 0
	versionsPos := bs.BitPosition()
	versionsLen := 0
	bs.WriteBits(0, 8) // Allocate 8 bits for the length
	for _, v := range c.versions {
		idx := strings.LastIndex(v, ".")
		if idx < 2 {
			idx = len(v)
		}

		curr, err := strconv.Atoi(v[2:idx])
		if err != nil {
			return err
		}

		if major != curr {
			major = curr
			bs.WriteBits(minor-1, 4)
			minor = 0
			versionsLen++
		} else {
			minor++
		}

		if v == memVersions[0] {
			break
		}
	}

	bs.SetBits(versionsLen, versionsPos, 8)
	bs.SaveToDisk("cfg")

	return nil
}

func (c *cli) loadFiles() error {
	c.versions = nil
	c.query.GameVersion = memVersions[0]
	if err := c.readMods(); err != nil {
		return err
	}
	versions, err := os.ReadFile("cfg")
	if err != nil {
		c.versions = memVersions
		return nil
	}

	bs := bitstream.FromBuffer(versions)
	var bitpos int

	if err := readQuery(bs, &bitpos, &c.query.ModLoader, &c.query.GameVersion); err != nil {
		return err
	}

	major := nextMajor
	versionsLen, err := bs.ReadBits(&bitpos, 8)
	if err != nil {
		return err
	}

	for i := 0; i < versionsLen; i++ {
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

	c.versions = append(c.versions, memVersions...)

	return nil
}

func (c *cli) readMods() error {
	if c.mods != nil {
		return errors.New("Mods already loaded")
	}

	d, err := os.ReadFile("modlist")
	if err != nil {
		return nil
	}

	bs := bitstream.FromBuffer(d)
	b := 0
	for {
		var m modEntry
		m.id, err = bs.ReadBits(&b, 24)
		if err != nil {
			break
		}

		if err := readQuery(bs, &b, &m.modLoader, &m.gameVersion); err != nil {
			return err
		}

		id1, err := bs.ReadBits(&b, 14)
		if err != nil {
			return err
		}
		id2, err := bs.ReadBits(&b, 10)
		if err != nil {
			return err
		}

		depsLen, err := bs.ReadBits(&b, 4)
		if err != nil {
			return err
		}
		for i := 0; i < depsLen; i++ {
			dep, err := bs.ReadBits(&b, 24)
			if err != nil {
				return err
			}
			m.deps = append(m.deps, dep)
		}

		m.name, err = bs.ReadPascalString(&b)
		if err != nil {
			return err
		}
		m.downloadUrl = fmt.Sprintf("%s%d/%d/%s", downloadURL, id1, id2, m.name)

		uploaded, err := bs.ReadBits64(&b, 64)
		if err != nil {
			return err
		}

		m.uploaded = time.Unix(uploaded, 0).UTC()
		c.mods = append(c.mods, m)
	}

	return nil
}

const downloadURL = "https://edge.forgecdn.net/files/"

func (c *cli) saveMods() error {
	var bs bitstream.Bitstream
	for _, m := range c.mods {
		bs.WriteBits(m.id, 24)
		if err := saveQuery(&bs, m.modLoader, m.gameVersion); err != nil {
			return err
		}

		r, ok := strings.CutPrefix(m.downloadUrl, downloadURL)
		if !ok {
			return fmt.Errorf("Download URL mismatch %#+v", r)
		}
		ids := strings.Split(r, "/")
		if len(ids) < 2 {
			return fmt.Errorf("Download URL missing id %#+v", ids)
		}
		id1, err := strconv.Atoi(ids[0])
		if err != nil {
			return err
		}
		id2, err := strconv.Atoi(ids[1])
		if err != nil {
			return err
		}

		bs.WriteBits(id1, 14)
		bs.WriteBits(id2, 10)
		bs.WriteBits(len(m.deps), 4)
		for _, dep := range m.deps {
			bs.WriteBits(dep, 24)
		}

		if err := bs.WritePascalString(m.name); err != nil {
			return err
		}

		bs.WriteBits64(m.uploaded.Unix(), 64)
	}

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
