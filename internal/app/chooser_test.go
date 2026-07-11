package app

import (
	"os/exec"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/makesometh-ing/trainer/internal/marketplace"
)

// openChooser opens the add-flow entry chooser via the command palette (`:a`).
func openChooser(m tea.Model) tea.Model {
	m, _ = m.Update(runeKey(':'))
	m, _ = m.Update(runeKey('a'))
	return m
}

// withMarket builds a test model with a Marketplace client injected, so the
// Search option is enabled rather than dimmed.
func withMarket(opts ...Option) Model {
	opts = append(opts, WithMarketplace(marketplace.New()))
	return newTestModel(browseResult(), opts...)
}

func TestChooserOpensListingBothOptions(t *testing.T) {
	var m tea.Model = withMarket()

	m = openChooser(m)

	out := view(m)
	if !strings.Contains(out, "Enter skill URL or repository") {
		t.Errorf("expected the manual-entry option, got:\n%s", out)
	}
	if !strings.Contains(out, "Search for skills") {
		t.Errorf("expected the Skill Search option, got:\n%s", out)
	}
}

func TestChooserEscCloses(t *testing.T) {
	var m tea.Model = withMarket()

	m = openChooser(m)
	m, _ = m.Update(namedKey(tea.KeyEsc))

	out := view(m)
	if strings.Contains(out, "Enter skill URL or repository") {
		t.Errorf("expected esc to close the chooser, got:\n%s", out)
	}
	if strings.Contains(out, "Skill source") {
		t.Errorf("expected esc to close without opening the wizard, got:\n%s", out)
	}
}

func TestChooserSearchDimmedAndInertWithoutMarketplace(t *testing.T) {
	// No WithMarketplace: the Search option is unavailable.
	var m tea.Model = newTestModel(browseResult())

	m = openChooser(m)

	out := view(m)
	if !strings.Contains(out, "Search for skills") {
		t.Fatalf("expected the Search option to render even when dimmed, got:\n%s", out)
	}
	if !strings.Contains(plain(out), "unavailable") {
		t.Errorf("expected the dim 'unavailable' tag on the Search option, got:\n%s", plain(out))
	}

	// Move to the Search option and press enter: nothing happens (no overlay, no
	// wizard, chooser stays open).
	m, _ = m.Update(runeKey('j'))
	m, _ = m.Update(namedKey(tea.KeyEnter))
	if !strings.Contains(view(m), "Search for skills") {
		t.Errorf("expected enter on the dimmed Search option to be inert, got:\n%s", view(m))
	}
	if strings.Contains(view(m), "Skill source") {
		t.Errorf("expected enter on the dimmed Search option not to open the wizard, got:\n%s", view(m))
	}
}

// The existing manual add path is reached by picking the first chooser option.
func TestChooserPickingManualReachesSourcePrompt(t *testing.T) {
	runner := func(cmd *exec.Cmd, done func(error) tea.Msg) tea.Cmd {
		return func() tea.Msg { return done(nil) }
	}
	var m tea.Model = withMarket(WithAddRunner(runner))

	m = openChooser(m)
	next, cmd := m.Update(namedKey(tea.KeyEnter)) // pick "Enter skill URL or repository"
	m = pump(next, cmd)

	if !strings.Contains(view(m), "Skill source") {
		t.Errorf("expected picking the manual option to open the Huh source prompt, got:\n%s", view(m))
	}
}
