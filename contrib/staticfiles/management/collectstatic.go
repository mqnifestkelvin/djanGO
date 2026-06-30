// Package management registers the collectstatic management command.
//
// Import this package with _ to register the command:
//
//	import _ "github.com/mqnifestkelvin/djanGO/contrib/staticfiles/management"
package management

// collectstatic mirrors Django's collectstatic management command.
//
// Django:
//
//	python manage.py collectstatic
//	# Copies all static files from each app's static/ dir and STATICFILES_DIRS
//	# into STATIC_ROOT.
//
// djanGO:
//
//	go run . collectstatic
//
// Output mirrors Django:
//
//	You have requested to collect static files at the destination
//	location as specified in your settings:
//
//	    /path/to/staticroot
//
//	131 static files copied to '/path/to/staticroot'.

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/mqnifestkelvin/djanGO/conf"
	"github.com/mqnifestkelvin/djanGO/contrib/staticfiles"
	"github.com/mqnifestkelvin/djanGO/core/management"
)

func init() {
	management.Register(&collectstaticCmd{})
}

type collectstaticCmd struct {
	management.BaseCommand
	noinput bool
	dryRun  bool
}

func (c *collectstaticCmd) Name() string { return "collectstatic" }
func (c *collectstaticCmd) Help() string {
	return "Collect static files into STATIC_ROOT"
}

func (c *collectstaticCmd) AddFlags(fs *flag.FlagSet) {
	fs.BoolVar(&c.noinput, "noinput", false, "Do NOT prompt the user for input")
	fs.BoolVar(&c.dryRun, "dry-run", false, "Do everything except modify the filesystem")
}

func (c *collectstaticCmd) Execute(args []string) error {
	settings := conf.Global()
	staticRoot := settings.StaticRoot
	if staticRoot == "" {
		return fmt.Errorf("collectstatic: STATIC_ROOT is not set in your settings")
	}

	files := staticfiles.AllFiles()
	if len(files) == 0 {
		fmt.Println("No static files found.")
		return nil
	}

	absRoot, err := filepath.Abs(staticRoot)
	if err != nil {
		return err
	}

	// Django prints this confirmation header.
	fmt.Printf("\nYou have requested to collect static files at the destination\n")
	fmt.Printf("location as specified in your settings:\n\n")
	fmt.Printf("    %s\n\n", absRoot)

	if !c.noinput {
		fmt.Printf("This will overwrite existing files!\nAre you sure you want to do this?\n")
		fmt.Printf("Type 'yes' to continue, or 'no' to cancel: ")
		var answer string
		fmt.Scanln(&answer)
		if answer != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	if !c.dryRun {
		if err := os.MkdirAll(absRoot, 0755); err != nil {
			return fmt.Errorf("collectstatic: could not create STATIC_ROOT: %w", err)
		}
	}

	copied := 0
	for rel, src := range files {
		dst := filepath.Join(absRoot, rel)
		if !c.dryRun {
			if err := copyFile(src, dst); err != nil {
				return fmt.Errorf("collectstatic: failed to copy %s: %w", rel, err)
			}
		}
		fmt.Printf("Copying '%s'\n", rel)
		copied++
	}

	fmt.Printf("\n%d static file(s) copied to '%s'.\n", copied, absRoot)
	return nil
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
