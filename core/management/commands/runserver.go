package commands

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/mqnifestkelvin/djanGO/conf"
	"github.com/mqnifestkelvin/djanGO/core/management"
	"github.com/mqnifestkelvin/djanGO/core/middleware"
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

	// Build middleware chain from settings.MIDDLEWARE —
	// mirrors Django's BaseHandler.load_middleware().
	//
	// Django:
	//   for middleware_path in reversed(settings.MIDDLEWARE):
	//       handler = middleware(handler)
	//
	// djanGO: same — collect registered middleware, wrap the mux.
	var chain []middleware.Func
	if conf.IsConfigured() {
		for _, name := range conf.Global().Middleware {
			if mw, ok := middleware.Lookup(name); ok {
				chain = append(chain, mw)
			}
		}
	}

	var handler http.Handler = mux
	if len(chain) > 0 {
		handler = middleware.Chain(mux, chain...)
	}

	fmt.Printf("djanGO development server running at http://%s/\n", addr)
	fmt.Println("Quit with CTRL-C.")

	return http.ListenAndServe(addr, handler)
}
