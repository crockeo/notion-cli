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
