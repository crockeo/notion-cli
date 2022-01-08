package main

import (
	"fmt"
	"os"

	"github.com/jomei/notionapi"

	"github.com/crockeo/notion-cli/commands/capture"
	"github.com/crockeo/notion-cli/commands/complete"
	"github.com/crockeo/notion-cli/config"
)

func main() {
	config, err := config.Load()
	guard(err)
	client := notionapi.NewClient(config.Token)

	args := os.Args[1:]
	if len(args) != 1 {
		printHelp()
		os.Exit(1)
	}

	command := args[0]
	if command == "capture" {
		capture.Capture(config, client)
	} else if command == "complete" {
		complete.Complete(config, client)
	} else {
		printHelp()
		os.Exit(1)
	}
}

func guard(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("Usage:", os.Args[0], "<command>")
	fmt.Println("  capture     Interactively capture a task from the terminal.")
	fmt.Println("  complete    Tag items with the time at which they were completed.")
}
