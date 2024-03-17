package main

import (
	"encoding/json"
	"fmt"

	"github.com/stuff7/mcman/cmd/api"
)

func main() {
	ret, err := api.SearchMods("quark", "1.20.1", api.Forge)
	if err != nil {
		fmt.Println("Error getting mod:", err)
		return
	}

	s, _ := json.MarshalIndent(ret, "", "\t")
	fmt.Println(string(s))
}
