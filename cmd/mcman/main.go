package main

import (
	"fmt"
	"log"

	"github.com/stuff7/mcman/cmd/readln"
)

func main() {
	var history []string

	for {
		cmd, err := readln.PushLn("> ", &history)
		if err != nil {
			log.Fatal("Error:", err)
		}

		fmt.Println(cmd)
		if cmd == "q" {
			break
		}
	}

	fmt.Println("Exiting...")
}
