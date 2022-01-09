package commands

import (
	"fmt"
	"os"
)

func Guard(err error) {
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func GuardOk(ok bool, message string) {
	if !ok {
		fmt.Println(message)
		os.Exit(1)
	}
}

func PrintHelp() {
	fmt.Println("Usage:", os.Args[0], "<command>")
	fmt.Println("  capture [JSON blob]   Interactively capture a task from the terminal.")
	fmt.Println("                        If JSON is provided, use it as additional default values.")
	fmt.Println("")
	fmt.Println("  complete              Tag items with the time at which they were completed.")
	fmt.Println("")
	fmt.Println("  dump                  Dumps information about the database in JSON.")
}
