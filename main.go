package main

import "github.com/stuff7/mcman/api"

func main() {
	if err := api.NewCli("> ").Run(); err != nil {
		println("Error:", err)
	}
}
