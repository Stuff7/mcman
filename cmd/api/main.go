package api

import (
	"fmt"
)

func SearchMods(search string, gameVersion string, modLoader ModLoaderType) ([]cfMod, error) {
	var mods []cfMod
	if err := getJSON(
		&mods,
		fmt.Sprintf("/v1/mods/search?gameId=%d&searchFilter=%s&gameVersion=%s&modLoaderType=%d&sortField=2&sortOrder=desc", MINECRAFT_ID, search, gameVersion, modLoader),
	); err != nil {
		return mods, err
	}
	return mods, nil
}

func GetModFiles(id int, gameVersion string, modLoader ModLoaderType) (ModFiles, error) {
	ret := ModFiles{ID: id, GameVersion: gameVersion, ModLoader: modLoader}
	if err := getJSON(&ret.Files, fmt.Sprintf("/v1/mods/%d/files?gameVersion=%s&modLoaderType=%d", id, gameVersion, modLoader)); err != nil {
		return ret, err
	}

	return ret, nil
}
