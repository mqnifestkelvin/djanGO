package commands

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/mqnifestkelvin/djanGO/core/management"
	"github.com/mqnifestkelvin/djanGO/core/urls"
)

func init() {
	management.Register(&runserverCommand{})
}

type runserverCommand struct {
	management.BaseCommand
	addr string
}

func (c *runserverCommand) Name() string { return "runserver" }
func (c *runserverCommand) Help() string { return "Start the djanGO development server" }

func (c *runserverCommand) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.addr, "addr", "127.0.0.1:8080", "Address and port to bind (default: 127.0.0.1:8080)")
}

func (c *runserverCommand) Execute(args []string) error {
	addr := c.addr
	if len(args) > 0 {
		addr = args[0]
	}

	mux := http.NewServeMux()
	urls.Mount(mux, urls.Registered(), "")

	fmt.Printf("djanGO development server running at http://%s/\n", addr)
	fmt.Println("Quit with CTRL-C.")

	return http.ListenAndServe(addr, mux)
}
