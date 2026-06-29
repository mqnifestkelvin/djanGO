package management

import (
	"flag"
	"sort"
	"testing"
)

// stub command used across tests
type stubCmd struct {
	name string
	help string
	ran  bool
	args []string
}

func (s *stubCmd) Name() string             { return s.name }
func (s *stubCmd) Help() string             { return s.help }
func (s *stubCmd) AddFlags(_ *flag.FlagSet) {}
func (s *stubCmd) Execute(args []string) error {
	s.ran = true
	s.args = args
	return nil
}

func TestRegisterCommand(t *testing.T) {
	cmd := &stubCmd{name: "t_cmd_register"}
	Register(cmd)

	names := AllCommands()
	found := false
	for _, n := range names {
		if n == "t_cmd_register" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 't_cmd_register' in AllCommands() after Register()")
	}
}

func TestAllCommandsReturnsSortedList(t *testing.T) {
	Register(&stubCmd{name: "t_cmd_zebra"})
	Register(&stubCmd{name: "t_cmd_alpha"})
	Register(&stubCmd{name: "t_cmd_mango"})

	names := AllCommands()
	if !sort.StringsAreSorted(names) {
		t.Errorf("expected AllCommands() to be sorted, got: %v", names)
	}
}

func TestRegisterOverwritesSameNameSilently(t *testing.T) {
	first := &stubCmd{name: "t_cmd_overwrite", help: "first"}
	second := &stubCmd{name: "t_cmd_overwrite", help: "second"}
	Register(first)
	Register(second)

	registry.mu.RLock()
	got := registry.commands["t_cmd_overwrite"]
	registry.mu.RUnlock()

	if got.Help() != "second" {
		t.Errorf("expected second registration to overwrite first, got help: %s", got.Help())
	}
}

func TestSimilarity(t *testing.T) {
	score := similarity("runserver", "runserver")
	if score != 9 {
		t.Errorf("identical strings should score 9, got %d", score)
	}

	typoScore := similarity("runservr", "runserver")
	if typoScore < 7 {
		t.Errorf("expected high score for near-match, got %d", typoScore)
	}

	zeroScore := similarity("xyz", "abc")
	if zeroScore != 0 {
		t.Errorf("expected 0 for no overlap, got %d", zeroScore)
	}
}

func TestClosestCommandSuggestion(t *testing.T) {
	Register(&stubCmd{name: "t_cmd_runserver"})

	// closestCommand works against ALL registered commands — just verify the
	// typo for our registered command scores above the threshold (>=2).
	suggestion := closestCommand("t_cmd_runservr")
	if suggestion == "" {
		t.Error("expected a suggestion for near-match typo, got empty string")
	}
}

func TestClosestCommandNoSuggestionForGarbage(t *testing.T) {
	// "xxxxxxx" should not match any registered command with score >= 2
	suggestion := closestCommand("xxxxxxxxxxxxxxxx")
	if suggestion != "" {
		t.Errorf("expected no suggestion for garbage input, got: %s", suggestion)
	}
}
