package main

import (
	"os"

	"github.com/stuff7/mcman/api"
)

func main() {
	args := os.Args
	parse := len(args) > 1 && args[1] == "parse"

	var err error
	if parse {
		err = api.TestParser("> ")
	} else {
		err = api.NewCli("> ").Run()
	}

	if err != nil {
		println("Error:", err)
	}
}
