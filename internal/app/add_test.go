package app

import (
	"os/exec"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/makesometh-ing/trainer/internal/skills"
	"github.com/makesometh-ing/trainer/internal/ssh"
)

// --- wizard test harness ---------------------------------------------------
//
// The add wizard is a huh.Form embedded in the model. Huh reads real key codes
// and delivers group transitions and completion as messages produced by cmds,
// so the plain `press` helper (which sends Text-only keys and discards cmds)
// cannot drive it. These helpers send realistic keys and pump the returned cmd.

// runeKey is a printable keystroke as Bubble Tea delivers it: Text and Code set.
func runeKey(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Text: string(r), Code: r})
}

// namedKey is a non-printable key (enter, tab, arrows) identified by its Code.
func namedKey(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: code})
}

// pump runs the returned cmd back into the model a bounded number of times so
// Huh's transition/completion messages are delivered. Bounded because the
// textinput cursor-blink tick would otherwise recur forever.
func pump(m tea.Model, cmd tea.Cmd) tea.Model {
	for i := 0; i < 3 && cmd != nil; i++ {
		m, cmd = m.Update(cmd())
	}
	return m
}

// wtype types text into the focused field one rune at a time. Characters insert
// synchronously inside Update, so the blink cmd can be discarded.
func wtype(m tea.Model, s string) tea.Model {
	for _, r := range s {
		m, _ = m.Update(runeKey(r))
	}
	return m
}

// wsend sends one key and pumps its cmd (used for enter/navigation that trigger
// group transitions or completion).
func wsend(m tea.Model, k tea.KeyPressMsg) tea.Model {
	next, cmd := m.Update(k)
	return pump(next, cmd)
}

// openWizard opens the add wizard via the command palette and the entry
// chooser: `:a` opens the chooser, then enter picks "Enter skill URL or
// repository". It pumps the form's Init cmd so the first field is focused and
// ready for input.
func openWizard(m tea.Model) tea.Model {
	m, _ = m.Update(runeKey(':'))
	m, _ = m.Update(runeKey('a')) // opens the chooser
	next, cmd := m.Update(namedKey(tea.KeyEnter))
	return pump(next, cmd)
}

func twoSSHKeys() []ssh.KeyPair {
	return []ssh.KeyPair{
		{Name: "id_ed25519", PrivatePath: "/ssh/id_ed25519"},
		{Name: "id_rsa", PrivatePath: "/ssh/id_rsa"},
	}
}

// --- cycles ----------------------------------------------------------------

func TestAddWizardOpensSourcePrompt(t *testing.T) {
	var m tea.Model = newTestModel(browseResult())

	m = openWizard(m)

	out := view(m)
	if !strings.Contains(out, "Add skill") {
		t.Errorf("expected the wizard modal title, got:\n%s", out)
	}
	if !strings.Contains(out, "Skill source") {
		t.Errorf("expected the huh source field, got:\n%s", out)
	}
}

func TestEmptySourceDoesNotComplete(t *testing.T) {
	ran := false
	runner := func(cmd *exec.Cmd, done func(error) tea.Msg) tea.Cmd {
		ran = true
		return func() tea.Msg { return done(nil) }
	}
	var m tea.Model = newTestModel(browseResult(), WithAddRunner(runner))

	m = openWizard(m)
	m = wsend(m, namedKey(tea.KeyEnter)) // enter with no source typed

	if ran {
		t.Error("expected an empty source not to run the add command")
	}
	if !strings.Contains(view(m), "Skill source") {
		t.Errorf("expected to stay on the source step for empty input, got:\n%s", view(m))
	}
}

func TestSSHStepShownForSSHSourceWithMultipleKeys(t *testing.T) {
	var m tea.Model = newTestModel(browseResult(), WithSSHKeys(twoSSHKeys()))

	m = openWizard(m)
	m = wtype(m, "git@github.com:owner/repo.git")
	m = wsend(m, namedKey(tea.KeyEnter))

	out := view(m)
	if !strings.Contains(out, "id_ed25519") || !strings.Contains(out, "id_rsa") {
		t.Errorf("expected the SSH key choices after an SSH source, got:\n%s", out)
	}
}

func TestSSHStepSkippedForNonSSHSource(t *testing.T) {
	ran := false
	runner := func(cmd *exec.Cmd, done func(error) tea.Msg) tea.Cmd {
		ran = true
		return func() tea.Msg { return done(nil) }
	}
	var m tea.Model = newTestModel(browseResult(), WithSSHKeys(twoSSHKeys()), WithAddRunner(runner))

	m = openWizard(m)
	m = wtype(m, "owner/repo")
	m = wsend(m, namedKey(tea.KeyEnter))

	if !ran {
		t.Error("expected a non-SSH source to complete on the first enter (no SSH step)")
	}
	if strings.Contains(view(m), "id_ed25519") {
		t.Errorf("did not expect SSH key choices for a non-SSH source, got:\n%s", view(m))
	}
}

// wtap sends a key without pumping (in-field navigation is synchronous).
func wtap(m tea.Model, k tea.KeyPressMsg) tea.Model {
	next, _ := m.Update(k)
	return next
}

func envHasKey(env []string, keyPath string) bool {
	for _, e := range env {
		if strings.Contains(e, "GIT_SSH_COMMAND") && strings.Contains(e, keyPath) {
			return true
		}
	}
	return false
}

func envHasAnySSHCommand(env []string) bool {
	for _, e := range env {
		if strings.Contains(e, "GIT_SSH_COMMAND") {
			return true
		}
	}
	return false
}

func TestSSHKeySelectionPassesChosenKey(t *testing.T) {
	var env []string
	runner := func(cmd *exec.Cmd, done func(error) tea.Msg) tea.Cmd {
		env = cmd.Env
		return func() tea.Msg { return done(nil) }
	}
	var m tea.Model = newTestModel(browseResult(), WithSSHKeys(twoSSHKeys()), WithAddRunner(runner))

	m = openWizard(m)
	m = wtype(m, "git@github.com:owner/repo.git")
	m = wsend(m, namedKey(tea.KeyEnter)) // advance to SSH-key select
	m = wtap(m, namedKey(tea.KeyDown))   // move to the second key (id_rsa)
	wsend(m, namedKey(tea.KeyEnter))     // confirm selection -> complete (runner captures env)

	if !envHasKey(env, "/ssh/id_rsa") {
		t.Errorf("expected GIT_SSH_COMMAND with the chosen key /ssh/id_rsa, env=%v", env)
	}
}

// Gotcha #4: a hidden Select still defaults its bound value to the first option,
// so a non-SSH add must attach no key at all.
func TestNonSSHSourceAttachesNoKey(t *testing.T) {
	var env []string
	runner := func(cmd *exec.Cmd, done func(error) tea.Msg) tea.Cmd {
		env = cmd.Env
		return func() tea.Msg { return done(nil) }
	}
	var m tea.Model = newTestModel(browseResult(), WithSSHKeys(twoSSHKeys()), WithAddRunner(runner))

	m = openWizard(m)
	m = wtype(m, "owner/repo")
	wsend(m, namedKey(tea.KeyEnter)) // completes; runner captures env

	if envHasAnySSHCommand(env) {
		t.Errorf("expected no GIT_SSH_COMMAND for a non-SSH source, env=%v", env)
	}
}

func TestCtrlCQuitsFromWizard(t *testing.T) {
	var m tea.Model = newTestModel(browseResult())
	m = openWizard(m)

	_, cmd := m.Update(tea.KeyPressMsg(tea.Key{Mod: tea.ModCtrl, Code: 'c'}))
	if cmd == nil {
		t.Fatal("expected ctrl+c to return a command while the wizard is open")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("expected ctrl+c to quit from the wizard")
	}
}

func TestEscClosesWizardWithoutAdding(t *testing.T) {
	ran := false
	runner := func(cmd *exec.Cmd, done func(error) tea.Msg) tea.Cmd {
		ran = true
		return func() tea.Msg { return done(nil) }
	}
	var m tea.Model = newTestModel(browseResult(), WithAddRunner(runner))

	m = openWizard(m)
	m = wtype(m, "owner/repo")
	m = wsend(m, namedKey(tea.KeyEsc))

	if ran {
		t.Error("expected esc to cancel without running the add command")
	}
	if strings.Contains(view(m), "Skill source") {
		t.Errorf("expected esc to close the wizard, got:\n%s", view(m))
	}
}

func TestAddDisabledIsInertAndShowsNoStatus(t *testing.T) {
	var m tea.Model = newTestModel(browseResult(), WithAddEnabled(false))

	m = press(m, ":")
	m = press(m, "a") // a dimmed command is inert

	out := view(m)
	if strings.Contains(out, "Skill source") {
		t.Errorf("expected no wizard when add is disabled, got:\n%s", out)
	}
	if hasRedText(out) {
		t.Errorf("expected no red status text when a dimmed command is pressed, got:\n%s", out)
	}
}

// Regression guard for the off-screen wizard: it must render as a centered
// overlay within the terminal, not appended below the already-full-height body.
func TestWizardOverlayFitsWithinTerminal(t *testing.T) {
	const w, h = 100, 30
	var m tea.Model = newTestModel(browseResult())
	m, _ = m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = openWizard(m)

	out := view(m)
	if !strings.Contains(out, "Skill source") {
		t.Fatalf("expected the wizard to be open, got:\n%s", out)
	}
	if gotH := lipgloss.Height(out); gotH > h {
		t.Errorf("wizard overlay height = %d, exceeds terminal height %d", gotH, h)
	}
	if gotW := lipgloss.Width(out); gotW > w {
		t.Errorf("wizard overlay width = %d, exceeds terminal width %d", gotW, w)
	}
}

// The wizard form is themed to match the app's Gruvbox palette, like the
// command palette. Huh's default theme colors the select selector fuchsia
// (#F780E2 = 247;128;226); the Gruvbox theme must replace it.
func TestWizardUsesGruvboxTheme(t *testing.T) {
	var m tea.Model = newTestModel(browseResult(), WithSSHKeys(twoSSHKeys()))

	m = openWizard(m)
	m = wtype(m, "git@github.com:owner/repo.git")
	m = wsend(m, namedKey(tea.KeyEnter)) // advance to the SSH-key select

	out := view(m)
	if strings.Contains(out, "247;128;226") {
		t.Errorf("expected the Gruvbox theme to override huh's fuchsia selector, got:\n%s", out)
	}
}

// The wizard is a fixed-size modal. Huh sizes every group to the tallest
// group's height when it receives a window size, which would pad the short
// source step up to the taller SSH step and make the modal jump after load
// (form.Init requests the size). The modal height must not change when a window
// size arrives while it is open.
func TestWizardModalDoesNotJumpWhenSizeArrives(t *testing.T) {
	var m tea.Model = newTestModel(browseResult(), WithSSHKeys(twoSSHKeys()))
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = openWizard(m)

	before := lipgloss.Height(m.(Model).renderWizard())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	after := lipgloss.Height(m.(Model).renderWizard())

	if before != after {
		t.Errorf("wizard modal height jumped when the window size arrived: before=%d after=%d", before, after)
	}
}

func TestConfirmingAddRunsCommandThenRescans(t *testing.T) {
	var ranArgs []string
	runner := func(cmd *exec.Cmd, done func(error) tea.Msg) tea.Cmd {
		ranArgs = cmd.Args
		return func() tea.Msg { return done(nil) }
	}
	rescanned := false
	rescan := func() []skills.ScanResult {
		rescanned = true
		return []skills.ScanResult{{
			Scope:  skills.Scope{Name: ".agents", Section: skills.SectionGlobal},
			Skills: []skills.Skill{{Name: "charlie", Path: "/root/charlie"}},
		}}
	}

	var m tea.Model = newTestModel(browseResult(), WithAddRunner(runner), WithRescan(rescan))

	m = openWizard(m)
	m = wtype(m, "owner/repo")
	m = wsend(m, namedKey(tea.KeyEnter))

	wantArgs := []string{"npx", "skills@latest", "add", "owner/repo"}
	if !slices.Equal(ranArgs, wantArgs) {
		t.Errorf("ran args = %v, want %v", ranArgs, wantArgs)
	}
	if !rescanned {
		t.Error("expected rescan after add command exits")
	}
	if !strings.Contains(view(m), "charlie") {
		t.Errorf("expected refreshed skill list to show charlie, got:\n%s", view(m))
	}
}
