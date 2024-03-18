package api

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/stuff7/mcman/cmd/readln"
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

func (c *cli) parseCmd(s string) Cmd {
	args := strings.Split((s), " ")
	if cmdStr := arrGet(args, 0); cmdStr != nil {
		var cmd Cmd
		if len(args) < 2 {
			args = []string{}
		} else {
			args = args[1:]
		}
		switch *cmdStr {
		case "search":
			cmd.Run = c.searchCmd
		case "set", "global":
			cmd.Run = c.setQueryCmd
		case "quit", "qa", "q":
			cmd.Run = c.quitCmd
		}
		cmd.Name = *cmdStr
		cmd.Args = args
		return cmd
	}

	return Cmd{Name: s}
}

func (c *cli) highlightInput(s string) string {
	cmd := c.parseCmd(s)
	if cmd.Run == nil {
		return s
	}

	return fmt.Sprintf("%s%s%s %s%s", clr(154), cmd.Name, clr(248), strings.Join(cmd.Args, " "), RESET)
}

func (c *cli) quitCmd(_ []string) error {
	c.Running = false
	return nil
}

func (c *cli) searchCmd(args []string) error {
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

func (c *cli) setQueryCmd(args []string) error {
	key := arrGet(args, 0)
	if key == nil {
		fmt.Println(c.query)
		return nil
	}
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
	Name string
	Args []string
	Run  func(args []string) error
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
