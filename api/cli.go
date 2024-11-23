package api

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/stuff7/mcman/bitstream"
	"github.com/stuff7/mcman/readln"
	"github.com/stuff7/mcman/storage"
)

type cli struct {
	profile     string
	query       searchQuery
	Running     bool
	prompt      string
	dbg         bool
	versions    []string
	mods        []modEntry
	isCfModpack bool
	modpackId   int
	modpackName string
}

func NewCli(prompt string) *cli {
	return &cli{Running: true, prompt: prompt, profile: "default"}
}

func (c *cli) Run() error {
	fmt.Printf("%s\nPress q to quit\n", LOGO)
	var history []string
	var tokens []token
	var cmd Cmd

	if err := c.loadCache(); err != nil {
		return err
	}

	if err := c.loadProfile(); err != nil {
		return err
	}

	for c.Running {
		fmt.Printf(
			"%s  %s%s  %s%d%s %s  %s%s%s %s%s%s  %sMODPACK#%s%d %s%s%s\n",
			clr(225),
			c.profile,
			RESET,
			clr(194),
			len(c.mods),
			RESET,
			pluralize("mod", len(c.mods)),
			clr(226),
			modLoaderKeywords[c.query.ModLoader],
			RESET,
			clr(194),
			c.query.GameVersion,
			RESET,
			BOLD,
			clr(194),
			c.modpackId,
			clr(214),
			c.modpackName,
			RESET,
		)

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

	println("\x1b[?25h")
	return c.saveCache()
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

	if minor, err := func() (int, error) {
		if idx >= len(gameVersion) {
			return 0, nil
		}
		return strconv.Atoi(gameVersion[idx+1:])
	}(); err != nil {
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

func getVersion(v string) (int, error) {
	idx := strings.LastIndex(v, ".")
	if idx < 2 {
		idx = len(v)
	}

	curr, err := strconv.Atoi(v[2:idx])

	if err != nil {
		return curr, err
	}

	return curr, nil
}

func (c *cli) saveCache() error {
	var bs bitstream.Bitstream

	if err := bs.WritePascalString(c.profile); err != nil {
		return err
	}

	major, err := getVersion(c.versions[0])
	if err != nil {
		return err
	}

	minor := 0

	versionsPos := bs.BitPosition()
	versionsLen := 0
	bs.WriteBits(0, 8) // Allocate 8 bits for the length
	for _, v := range c.versions {
		curr, err := getVersion(v)
		if err != nil {
			return err
		}

		if major != curr {
			major = curr
			bs.WriteBits(minor-1, 4)
			minor = 1
			versionsLen++
		} else {
			minor++
		}

		if v == memVersions[0] {
			break
		}
	}

	bs.SetBits(versionsLen, versionsPos, 8)
	bs.SaveToDisk("cache")

	return nil
}

func (c *cli) saveProfile() error {
	var bs bitstream.Bitstream

	if err := saveQuery(&bs, c.query.ModLoader, c.query.GameVersion); err != nil {
		return err
	}

	if c.isCfModpack {
		bs.WriteBits(1, 1)
		bs.WriteBits(c.modpackId, 24)
		bs.WritePascalString(c.modpackName)
	}

	bs.SaveToDisk(c.profilePath("cfg"))

	return nil
}

func (c *cli) loadCache() error {
	cache, err := storage.ReadOrCreate("cache")
	if err != nil || len(cache) == 0 {
		c.versions = memVersions
		return nil
	}

	bs := bitstream.FromBuffer(cache)
	var bitpos int

	profile, err := bs.ReadPascalString(&bitpos)
	if err != nil {
		return err
	}
	c.profile = profile

	versionsLen, err := bs.ReadBits(&bitpos, 8)
	if err != nil {
		return err
	}

	c.versions = nil
	major := nextMajor + versionsLen - 1
	for i := 0; i < versionsLen; i++ {
		v, err := bs.ReadBits(&bitpos, 4)
		if err != nil {
			break
		}
		for minor := v; minor > 0; minor-- {
			c.versions = append(c.versions, fmt.Sprintf("1.%d.%d", major, minor))
		}
		c.versions = append(c.versions, fmt.Sprintf("1.%d", major))
		major--
	}

	c.versions = append(c.versions, memVersions...)

	return nil
}

func (c *cli) loadProfile() error {
	c.query.GameVersion = memVersions[0]
	c.query.ModLoader = 0
	c.mods = nil
	c.isCfModpack = false
	c.modpackId = -1
	c.modpackName = ""
	if err := c.readMods(); err != nil {
		return err
	}

	cfg, err := storage.ReadOrCreate(c.profilePath("cfg"))
	if err != nil || len(cfg) == 0 {
		return nil
	}

	bs := bitstream.FromBuffer(cfg)
	bitpos := 0

	if err := readQuery(bs, &bitpos, &c.query.ModLoader, &c.query.GameVersion); err != nil {
		return err
	}

	isCfModpack, err := bs.ReadBits(&bitpos, 1)
	if err != nil {
		return err
	}
	c.isCfModpack = isCfModpack != 0

	if !c.isCfModpack {
		return nil
	}

	modpackId, err := bs.ReadBits(&bitpos, 24)
	if err != nil {
		return err
	}
	c.modpackId = modpackId

	modpackName, err := bs.ReadPascalString(&bitpos)
	if err != nil {
		return err
	}
	c.modpackName = modpackName

	return nil
}

func (c *cli) profilePath(path string) string {
	return filepath.Join("profiles", c.profile, path)
}

func (c *cli) readMods() error {
	d, err := storage.ReadOrCreate(c.profilePath("modlist"))
	if err != nil {
		return nil
	}

	if len(d) == 0 {
		c.mods = nil
		return nil
	}

	bs := bitstream.FromBuffer(d)
	b := 0
	for {
		var m modEntry
		m.Id, err = bs.ReadBits(&b, 24)
		if err != nil {
			break
		}

		if err := readQuery(bs, &b, &m.ModLoader, &m.GameVersion); err != nil {
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
			m.Deps = append(m.Deps, dep)
		}

		m.Name, err = bs.ReadPascalString(&b)
		if err != nil {
			return fmt.Errorf("Failed to read mod at index %d:\n%w\nMod: %#+v", len(c.mods), err, m)
		}
		m.DownloadUrl = fmt.Sprintf("%s%d/%d/%s", downloadURL, id1, id2, url.QueryEscape(m.Name))

		uploaded, err := bs.ReadBits64(&b, 64)
		if err != nil {
			return err
		}

		m.Uploaded = time.Unix(uploaded, 0).UTC()
		c.mods = append(c.mods, m)
	}

	return nil
}

const downloadURL = "https://edge.forgecdn.net/files/"

func (c *cli) saveMods() error {
	var bs bitstream.Bitstream
	for _, m := range c.mods {
		bs.WriteBits(m.Id, 24)
		if err := saveQuery(&bs, m.ModLoader, m.GameVersion); err != nil {
			return err
		}

		r, ok := strings.CutPrefix(m.DownloadUrl, downloadURL)
		if !ok {
			fmt.Printf("Download URL mismatch for mod %+v. URL: %+v\n", m.Name, m.DownloadUrl)
			slashCount := 0
			idx := strings.LastIndexFunc(m.DownloadUrl, func(r rune) bool {
				if r == '/' {
					slashCount++
				}

				if slashCount == 3 {
					return true
				}

				return false
			})

			if idx == -1 {
				return fmt.Errorf("Unexpected download URL form for mod %+v. URL: %+v", m.Name, m.DownloadUrl)
			}

			r = m.DownloadUrl[idx:]
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
		bs.WriteBits(len(m.Deps), 4)
		for _, dep := range m.Deps {
			bs.WriteBits(dep, 24)
		}

		if err := bs.WritePascalString(m.Name); err != nil {
			return err
		}

		bs.WriteBits64(m.Uploaded.Unix(), 64)
	}

	return bs.SaveToDisk(c.profilePath("modlist"))
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
