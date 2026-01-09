package main

import (
	"fmt"
	"os"

	"github.com/samuelenocsson/devops-tui/cmd"
)

func main() {
	// Check for subcommands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "logout":
			cmd.ExecuteLogout()
			return
		case "login":
			cmd.ExecuteLogin()
			return
		}
	}

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
