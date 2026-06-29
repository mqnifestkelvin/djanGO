package commands

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/mqnifestkelvin/djanGO/contrib/auth"
	"github.com/mqnifestkelvin/djanGO/core/management"
	"golang.org/x/term"
)

// readPassword reads a password — uses terminal echo-off when stdin is a TTY,
// falls back to plain line reading when piped (tests, CI).
func readPassword(r *bufio.Reader) ([]byte, error) {
	if term.IsTerminal(syscall.Stdin) {
		pw, err := term.ReadPassword(syscall.Stdin)
		fmt.Println()
		return pw, err
	}
	line, err := r.ReadString('\n')
	return []byte(strings.TrimSpace(line)), err
}

func init() {
	management.Register(&createSuperuserCmd{})
}

type createSuperuserCmd struct {
	management.BaseCommand
	username string
	email    string
}

func (c *createSuperuserCmd) Name() string { return "createsuperuser" }
func (c *createSuperuserCmd) Help() string {
	return "Create a superuser account (mirrors Django's createsuperuser)"
}

func (c *createSuperuserCmd) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.username, "username", "", "Username for the superuser")
	fs.StringVar(&c.email, "email", "", "Email address for the superuser")
}

// Execute mirrors Django's createsuperuser Command.handle().
//
// Django:
//
//	python manage.py createsuperuser
//	# prompts: Username, Email, Password, Password (again)
func (c *createSuperuserCmd) Execute(args []string) error {
	reader := bufio.NewReader(os.Stdin)

	username := c.username
	if username == "" {
		fmt.Print("Username: ")
		line, _ := reader.ReadString('\n')
		username = strings.TrimSpace(line)
		if username == "" {
			return fmt.Errorf("username cannot be blank")
		}
	}

	email := c.email
	if email == "" {
		fmt.Print("Email address: ")
		line, _ := reader.ReadString('\n')
		email = strings.TrimSpace(line)
	}

	fmt.Print("Password: ")
	pw1, err := readPassword(reader)
	if err != nil {
		return err
	}

	fmt.Print("Password (again): ")
	pw2, err := readPassword(reader)
	if err != nil {
		return err
	}

	if string(pw1) != string(pw2) {
		return fmt.Errorf("Error: Your passwords didn't match.")
	}
	if len(pw1) == 0 {
		return fmt.Errorf("Error: Blank passwords are not permitted.")
	}

	_, err = auth.CreateSuperuser(username, email, string(pw1))
	if err != nil {
		return fmt.Errorf("Error: %w", err)
	}

	fmt.Printf("Superuser created successfully.\n")
	return nil
}
