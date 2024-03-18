package api

import (
	"fmt"
)

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
