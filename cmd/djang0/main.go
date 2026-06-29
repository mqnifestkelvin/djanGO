package main

import (
	"fmt"
	"os"

	"github.com/mqnifestkelvin/djanGO/cmd/djang0/management"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: djanGO-admin <command> [args]")
		fmt.Println()
		fmt.Println("Available commands:")
		fmt.Println("  startproject <name> [directory]  Create a new djanGO project")
		fmt.Println("  startapp     <name> [directory]  Create a new app inside a project")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "startproject":
		management.RunStartProject(os.Args[2:])
	case "startapp":
		management.RunStartApp(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		os.Exit(1)
	}
}
