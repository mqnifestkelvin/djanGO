package management

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
)

var registry = &commandRegistry{
	commands: make(map[string]Command),
}

type commandRegistry struct {
	mu       sync.RWMutex
	commands map[string]Command // keyed by command name
}

// Register adds a management command to the global registry.
// Call this from your app's init() or management/commands/*.go init().
// Equivalent to Django's auto-discovery of management/commands/ packages.
func Register(cmd Command) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.commands[cmd.Name()] = cmd
}

// Execute finds and runs a command by name, parsing its flags and args.
// Equivalent to Django's execute_from_command_line().
func Execute(args []string) {
	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	name := args[0]
	rest := args[1:]

	registry.mu.RLock()
	cmd, ok := registry.commands[name]
	registry.mu.RUnlock()

	if !ok {
		fmt.Fprintf(os.Stderr, "Error: unknown command '%s'\n\n", name)
		suggestion := closestCommand(name)
		if suggestion != "" {
			fmt.Fprintf(os.Stderr, "Did you mean '%s'?\n\n", suggestion)
		}
		printUsage()
		os.Exit(1)
	}

	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	cmd.AddFlags(fs)

	if err := fs.Parse(rest); err != nil {
		os.Exit(1)
	}

	if err := cmd.Execute(fs.Args()); err != nil {
		if ce, ok := err.(*CommandError); ok {
			fmt.Fprintf(os.Stderr, "Error: %s\n", ce.Message)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		}
		os.Exit(1)
	}
}

// AllCommands returns all registered command names sorted alphabetically.
func AllCommands() []string {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	names := make([]string, 0, len(registry.commands))
	for name := range registry.commands {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func printUsage() {
	fmt.Println("Usage: go run manage.go <command> [options] [args]")
	fmt.Println()
	fmt.Println("Available commands:")

	names := AllCommands()
	if len(names) == 0 {
		fmt.Println("  (no commands registered)")
		return
	}

	// Find longest name for alignment
	maxLen := 0
	for _, n := range names {
		if len(n) > maxLen {
			maxLen = len(n)
		}
	}

	registry.mu.RLock()
	defer registry.mu.RUnlock()
	for _, n := range names {
		cmd := registry.commands[n]
		padding := strings.Repeat(" ", maxLen-len(n))
		fmt.Printf("  %s%s  %s\n", n, padding, cmd.Help())
	}
}

// closestCommand returns the most similar registered command name,
// used to suggest corrections for typos — like Django's did-you-mean.
func closestCommand(input string) string {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	names := make([]string, 0, len(registry.commands))
	for name := range registry.commands {
		names = append(names, name)
	}

	best := ""
	bestScore := 0
	for _, name := range names {
		score := similarity(input, name)
		if score > bestScore {
			bestScore = score
			best = name
		}
	}
	if bestScore < 2 {
		return ""
	}
	return best
}

// similarity returns a simple character overlap score between two strings.
func similarity(a, b string) int {
	score := 0
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] == b[i] {
			score++
		}
	}
	return score
}
