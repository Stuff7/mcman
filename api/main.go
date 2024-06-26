package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/stuff7/mcman/slc"
)

func (c *cli) importMods(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	body, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	var mods []importFile
	if err := json.Unmarshal(body, &mods); err != nil {
		return dumpJson(body, err)
	}

	for _, mod := range mods {
		if err := c.addMod(mod.ID, false); err != nil {
			fmt.Printf("%s! %s%s%s\n", clr(210), BOLD, err, RESET)
		}
		time.Sleep(time.Millisecond * 100)
	}

	return nil
}

func (c *cli) remMod(search any, isIdx bool) error {
	idx := -1
	switch id := search.(type) {
	case string:
		idx = slices.IndexFunc(c.mods, func(m modEntry) bool {
			return strings.Contains(strings.ToLower(m.Name), id)
		})
		if idx == -1 {
			return fmt.Errorf("Could not find mod %#+v", id)
		}
	case int:
		idx = id
		if !isIdx {
			idx = slices.IndexFunc(c.mods, func(m modEntry) bool { return m.Id == id })

			if idx == -1 {
				return fmt.Errorf("Could not find mod with id %d", id)
			}
		}
	default:
		return errors.New("Invalid search")
	}

	return removeModEntry(&c.mods, idx)
}

func (c *cli) addMod(search any, isDependency bool) error {
	var id int
	var f *CfFile
	switch search := search.(type) {
	case string:
		mods, err := searchMods(search, c.query)
		if err != nil {
			return err
		}

		m := slc.Get(mods, 0)
		if m == nil {
			return errors.New("No mods found")
		}

		id = m.ID
		f = slc.Last(m.Files)
	case int:
		m, err := getModFiles(search, c.query)
		if err != nil {
			return err
		}

		id = m.ID
		f = slc.Get(m.Files, 0)
	}

	if f == nil {
		return fmt.Errorf("No downloads found for %+v", search)
	}

	c.mods = appendModEntry(c.mods, id, c.query, f)
	if isDependency {
		fmt.Printf("%s+ Dep %s%s%s added\n", clr(51), BOLD, f.Name, RESET)
	} else {
		fmt.Printf("%s+ Mod %s%s%s added\n", clr(49), BOLD, f.Name, RESET)
	}
	for _, d := range f.Dependencies {
		if d.Relation == RequiredDependency && !slices.ContainsFunc(c.mods, func(m modEntry) bool { return d.ModId == m.Id }) {
			return c.addMod(d.ModId, true)
		}
	}

	return nil
}

func getVersions() ([]gameVersion, error) {
	var versions []gameVersion
	if err := getJSON(&versions, "/v1/minecraft/version"); err != nil {
		return versions, err
	}

	return versions, nil
}

func searchMods(search string, query searchQuery) ([]cfMod, error) {
	var mods []cfMod
	if err := getJSON(
		&mods,
		fmt.Sprintf(
			"/v1/mods/search%s&gameId=%d&searchFilter=%s&sortField=2&sortOrder=desc",
			query,
			MINECRAFT_ID,
			search,
		),
	); err != nil {
		return mods, err
	}
	return mods, nil
}

func getModFiles(id int, query searchQuery) (ModFiles, error) {
	ret := ModFiles{ID: id, GameVersion: query.GameVersion, ModLoader: query.ModLoader}
	if err := getJSON(&ret.Files, fmt.Sprintf("/v1/mods/%d/files%s", id, query)); err != nil {
		return ret, err
	}

	return ret, nil
}

func clr(id byte) string {
	return fmt.Sprintf("\x1b[38;5;%dm", id)
}

func hl(s string, keywords []string, color byte) string {
	for _, a := range keywords {
		s = strings.ReplaceAll(s, a, clr(color)+a+RESET)
	}
	return s
}

type searchQuery struct {
	GameVersion string `query:"gameVersion" key:"gameVersion"`
	ModLoader   int    `query:"modLoaderType" key:"modLoader"`
}

var queryFields = (searchQuery{}).getFields()

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

		switch val := val.(type) {
		case string:
			if val == "" {
				continue
			}
			strVal = val
		case int:
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

func (s searchQuery) getFields() []string {
	structType := reflect.TypeOf(s)
	fieldNames := make([]string, structType.NumField())

	for i := 0; i < structType.NumField(); i++ {
		fieldNames[i] = structType.Field(i).Tag.Get("key")
	}

	return fieldNames
}
