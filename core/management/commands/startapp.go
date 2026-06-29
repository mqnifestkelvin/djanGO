package commands

import (
	"flag"

	djmanagement "github.com/mqnifestkelvin/djanGO/cmd/djang0/management"
	"github.com/mqnifestkelvin/djanGO/core/management"
)

func init() {
	management.Register(&startappCommand{})
}

type startappCommand struct {
	management.BaseCommand
}

func (c *startappCommand) Name() string { return "startapp" }
func (c *startappCommand) Help() string { return "Create a new djanGO app" }
func (c *startappCommand) AddFlags(_ *flag.FlagSet) {}

func (c *startappCommand) Execute(args []string) error {
	if len(args) == 0 {
		return management.Err("you must provide an app name.\nUsage: go run manage.go startapp <name>")
	}
	djmanagement.RunStartApp(args)
	return nil
}
