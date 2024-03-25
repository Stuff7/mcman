package main

import (
	"fmt"

	"github.com/stuff7/mcman/api"
)

func main() {
	if err := api.NewCli("> ").Run(); err != nil {
		fmt.Printf("Error: %#+v\n", err)
	}
}
