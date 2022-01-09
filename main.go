package main

import (
	"os"

	"github.com/jomei/notionapi"

	"github.com/crockeo/notion-cli/commands"
	"github.com/crockeo/notion-cli/commands/capture"
	"github.com/crockeo/notion-cli/commands/complete"
	"github.com/crockeo/notion-cli/commands/dump"
	"github.com/crockeo/notion-cli/config"
)

func main() {
	config, err := config.Load()
	commands.Guard(err)
	client := notionapi.NewClient(config.Token)

	args := os.Args[1:]
	if len(args) == 0 {
		commands.PrintHelp()
		os.Exit(1)
	}

	command := args[0]
	if command == "capture" {
		capture.Capture(config, client, args[1:])
	} else if command == "complete" {
		complete.Complete(config, client, args[1:])
	} else if command == "dump" {
		dump.Dump(config, client, args[1:])
	} else {
		commands.PrintHelp()
		os.Exit(1)
	}
}
