package management

import (
	"flag"
	"fmt"
	"io"
	"os"
)

// Command is the interface every management command must implement.
// Equivalent to Django's BaseCommand in django/core/management/base.py.
type Command interface {
	// Name returns the command name used on the CLI, e.g. "runserver".
	Name() string

	// Help returns a one-line description shown in the command list.
	Help() string

	// AddFlags lets the command register its own flags on the given FlagSet.
	AddFlags(fs *flag.FlagSet)

	// Execute runs the command with the remaining positional args.
	Execute(args []string) error
}

// BaseCommand is an embeddable struct that provides default no-op
// implementations so commands only need to override what they use.
type BaseCommand struct {
	Stdout io.Writer
	Stderr io.Writer
}

func (b *BaseCommand) AddFlags(_ *flag.FlagSet) {}

func (b *BaseCommand) stdout() io.Writer {
	if b.Stdout != nil {
		return b.Stdout
	}
	return os.Stdout
}

func (b *BaseCommand) stderr() io.Writer {
	if b.Stderr != nil {
		return b.Stderr
	}
	return os.Stderr
}

// Print writes a line to the command's stdout.
func (b *BaseCommand) Print(format string, args ...interface{}) {
	fmt.Fprintf(b.stdout(), format+"\n", args...)
}

// Error writes a line to the command's stderr.
func (b *BaseCommand) Error(format string, args ...interface{}) {
	fmt.Fprintf(b.stderr(), "Error: "+format+"\n", args...)
}

// CommandError is returned when a management command fails with a user-facing message.
type CommandError struct {
	Message string
}

func (e *CommandError) Error() string { return e.Message }

// Err is a convenience constructor for CommandError.
func Err(format string, args ...interface{}) error {
	return &CommandError{Message: fmt.Sprintf(format, args...)}
}
