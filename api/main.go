package api

import (
	"fmt"
	"reflect"
	"strings"
)

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

func arrGet[T any](arr []T, idx int) *T {
	if idx < 0 || idx >= len(arr) {
		return nil
	}
	return &arr[idx]
}

func arrFlat[T any](arr [][]T) []T {
	flat := []T{}
	for _, t := range arr {
		flat = append(flat, t...)
	}
	return flat
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
	GameVersion string        `query:"gameVersion"`
	ModLoader   ModLoaderType `query:"modLoader"`
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

func (s searchQuery) getFields() []string {
	structType := reflect.TypeOf(s)
	fieldNames := make([]string, structType.NumField())

	for i := 0; i < structType.NumField(); i++ {
		fieldNames[i] = structType.Field(i).Tag.Get("query")
	}

	return fieldNames
}
