package app

import (
	"errors"
	"os/exec"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/h2non/gock"
)

// footerOf drives the model, then returns the plain (ANSI-stripped) footer line.
func footerOf(m tea.Model) string {
	return plain(m.(Model).renderFooter())
}

func TestFooterSkillsContext(t *testing.T) {
	var m tea.Model = newTestModel(browseResult())
	m = press(m, "2") // focus the Skills pane

	f := footerOf(m)
	if !strings.Contains(f, "SKILLS") {
		t.Errorf("expected SKILLS chip, got:\n%s", f)
	}
	for _, want := range []string{"j/k", "/", "f", "r"} {
		if !strings.Contains(f, want) {
			t.Errorf("expected Skills key %q in footer, got:\n%s", want, f)
		}
	}
	for _, unwanted := range []string{"1/2/3", "i/r/s/a"} {
		if strings.Contains(f, unwanted) {
			t.Errorf("did not expect on-screen keys %q in footer, got:\n%s", unwanted, f)
		}
	}
}

func TestFooterScopeContext(t *testing.T) {
	var m tea.Model = newTestModel(browseResult())
	m = press(m, "1") // focus the Scope pane

	f := footerOf(m)
	if !strings.Contains(f, "SCOPE") {
		t.Errorf("expected SCOPE chip, got:\n%s", f)
	}
	if !strings.Contains(f, "switch scope") {
		t.Errorf("expected 'switch scope' in footer, got:\n%s", f)
	}
	if !strings.Contains(f, "commands") || !strings.Contains(f, "quit") {
		t.Errorf("expected the global tail in the footer, got:\n%s", f)
	}
	for _, unwanted := range []string{"search", "filter", "reset"} {
		if strings.Contains(f, unwanted) {
			t.Errorf("did not expect Skills key %q in Scope footer, got:\n%s", unwanted, f)
		}
	}
}

func TestFooterDetailsSkillTab(t *testing.T) {
	var m tea.Model = newTestModel(browseResult())
	m = press(m, "3") // focus the Details pane; default tab is SKILL.md

	f := footerOf(m)
	if !strings.Contains(f, "DETAILS") {
		t.Errorf("expected DETAILS chip, got:\n%s", f)
	}
	for _, want := range []string{"ctrl+d/u", "g/G", "scroll"} {
		if !strings.Contains(f, want) {
			t.Errorf("expected scroll key %q in SKILL.md footer, got:\n%s", want, f)
		}
	}
	if strings.Contains(f, "i/r/s/a") {
		t.Errorf("did not expect tab keys in footer, got:\n%s", f)
	}
	if strings.Contains(f, "ctrl+f/b") {
		t.Errorf("did not expect full-page scroll in the footer, got:\n%s", f)
	}
	if strings.Contains(f, "focus content") || strings.Contains(f, "focus files") {
		t.Errorf("did not expect a file-list toggle on the SKILL.md tab, got:\n%s", f)
	}
}

func TestFooterDetailsFileTabListActive(t *testing.T) {
	var m tea.Model = newTestModel(detailResult(t))
	m = press(m, "3") // focus Details
	m = press(m, "r") // References tab; subfocus defaults to the file list

	f := footerOf(m)
	if !strings.Contains(f, "DETAILS") {
		t.Errorf("expected DETAILS chip, got:\n%s", f)
	}
	if !strings.Contains(f, "select file") {
		t.Errorf("expected 'select file' when the file list is active, got:\n%s", f)
	}
	if !strings.Contains(f, "focus content") {
		t.Errorf("expected 'tab focus content' when the file list is active, got:\n%s", f)
	}
	if strings.Contains(f, "half-page") {
		t.Errorf("did not expect content scroll keys while the file list is active, got:\n%s", f)
	}
}

func TestFooterDetailsFileTabContentActive(t *testing.T) {
	var m tea.Model = newTestModel(detailResult(t))
	m = press(m, "3")   // focus Details
	m = press(m, "r")   // References tab
	m = press(m, "tab") // toggle subfocus to the content pane

	f := footerOf(m)
	if !strings.Contains(f, "DETAILS") {
		t.Errorf("expected DETAILS chip, got:\n%s", f)
	}
	for _, want := range []string{"scroll", "half-page", "g/G", "focus files"} {
		if !strings.Contains(f, want) {
			t.Errorf("expected %q when the content is active, got:\n%s", want, f)
		}
	}
	if strings.Contains(f, "select file") || strings.Contains(f, "focus content") {
		t.Errorf("did not expect the list-active keys while content is active, got:\n%s", f)
	}
}

func TestFooterSearchContext(t *testing.T) {
	var m tea.Model = newTestModel(browseResult())
	m = press(m, "2") // focus Skills
	m = press(m, "/") // enter search

	f := footerOf(m)
	if !strings.Contains(f, "SEARCH") {
		t.Errorf("expected SEARCH chip, got:\n%s", f)
	}
	for _, want := range []string{"enter", "apply", "esc", "clear"} {
		if !strings.Contains(f, want) {
			t.Errorf("expected %q in SEARCH footer, got:\n%s", want, f)
		}
	}
	if strings.Contains(f, "commands") || strings.Contains(f, "move focus") {
		t.Errorf("did not expect the global tail in the SEARCH footer, got:\n%s", f)
	}
}

func TestFooterFilterContext(t *testing.T) {
	var m tea.Model = newTestModel(browseResult())
	m = press(m, "2") // focus Skills
	m = press(m, "f") // focus the filter

	f := footerOf(m)
	if !strings.Contains(f, "FILTER") {
		t.Errorf("expected FILTER chip, got:\n%s", f)
	}
	for _, want := range []string{"h/l", "move option", "space", "apply", "c", "clear", "esc", "done"} {
		if !strings.Contains(f, want) {
			t.Errorf("expected %q in FILTER footer, got:\n%s", want, f)
		}
	}
	if strings.Contains(f, "commands") {
		t.Errorf("did not expect the global tail in the FILTER footer, got:\n%s", f)
	}
}

// Slice 14, cycle 1: the add-flow entry chooser keeps the footer live with the
// ADD chip and its keys — select / choose / cancel — and none of the browse or
// global-tail keys.
func TestFooterAddChooserContext(t *testing.T) {
	var m tea.Model = withMarket()
	m = openChooser(m)

	f := footerOf(m)
	if !strings.Contains(f, "ADD") {
		t.Errorf("expected ADD chip, got:\n%s", f)
	}
	for _, want := range []string{"select", "choose", "cancel"} {
		if !strings.Contains(f, want) {
			t.Errorf("expected %q in the ADD footer, got:\n%s", want, f)
		}
	}
	for _, unwanted := range []string{"commands", "switch scope", "sort", "quit"} {
		if strings.Contains(f, unwanted) {
			t.Errorf("did not expect %q in the ADD footer, got:\n%s", unwanted, f)
		}
	}
}

// Slice 14, cycle 2: the Skill Search box shows the SEARCH chip and its keys —
// type to search / results / back — and none of the results-list sort keys.
func TestFooterSkillSearchBoxContext(t *testing.T) {
	var m tea.Model = withMarket()
	m = settle(openSkillSearch(m, 100, 30))

	f := footerOf(m)
	if !strings.Contains(f, "SEARCH") {
		t.Errorf("expected SEARCH chip in the search box, got:\n%s", f)
	}
	for _, want := range []string{"type to search", "results", "back"} {
		if !strings.Contains(f, want) {
			t.Errorf("expected %q in the search-box footer, got:\n%s", want, f)
		}
	}
	for _, unwanted := range []string{"sort", "commands", "quit"} {
		if strings.Contains(f, unwanted) {
			t.Errorf("did not expect %q in the search-box footer, got:\n%s", unwanted, f)
		}
	}
}

// Slice 14, cycle 3: the results list shows the RESULTS chip and its keys —
// sort / detail / install / search / back — with no retry outside the error
// state; in the error state the retry key is advertised.
func TestFooterSkillSearchResultsContext(t *testing.T) {
	defer gock.Off()
	m := searchWithResults(t)
	m, _ = m.Update(namedKey(tea.KeyEnter)) // focus the results list

	f := footerOf(m)
	if !strings.Contains(f, "RESULTS") {
		t.Errorf("expected RESULTS chip, got:\n%s", f)
	}
	for _, want := range []string{"sort", "detail", "install", "search", "back"} {
		if !strings.Contains(f, want) {
			t.Errorf("expected %q in the results footer, got:\n%s", want, f)
		}
	}
	if strings.Contains(f, "retry") {
		t.Errorf("did not expect a retry key outside the error state, got:\n%s", f)
	}

	// A failed search sets the error state; entering the list keeps it, and the
	// footer then advertises the retry key.
	gock.New("https://skills.sh").Get("/api/search").Reply(500)
	var e tea.Model = withMarket()
	e = settle(openSkillSearch(e, 100, 30))
	e = typeSearch(e, "react")
	e, cmd := fireDebounce(e)
	e = runSearchCmd(e, cmd)                // 500 → searchError, still in the box
	e, _ = e.Update(namedKey(tea.KeyEnter)) // hand off to the results list
	if ef := footerOf(e); !strings.Contains(ef, "retry") {
		t.Errorf("expected 'space retry' in the results footer error state, got:\n%s", ef)
	}
}

// Slice 14, cycle 4: the Skill Detail shows the DETAIL chip and its keys —
// tabs / files-content / move / list / install / search — and advertises the
// retry key only when the download failed.
func TestFooterSkillSearchDetailContext(t *testing.T) {
	defer gock.Off()
	m := searchWithResults(t)
	m = seedDownload(t, m, mktDetailFiles())
	m = enterDetail(m)

	f := footerOf(m)
	if !strings.Contains(f, "DETAIL") {
		t.Errorf("expected DETAIL chip, got:\n%s", f)
	}
	for _, want := range []string{"i/r/s/a", "files/content", "list", "install"} {
		if !strings.Contains(f, want) {
			t.Errorf("expected %q in the detail footer, got:\n%s", want, f)
		}
	}
	if strings.Contains(f, "retry") {
		t.Errorf("did not expect a retry key without a download error, got:\n%s", f)
	}

	// A failed download sets the detail error state; the footer then advertises
	// the retry key.
	e := searchWithResults(t)
	gock.New("https://skills.sh").
		Get("/api/download/vercel-labs/agent-skills/vercel-react-best-practices").
		Reply(500)
	e, cmd := fireDwell(e)
	e = runSearchCmd(e, cmd) // 500 → dlError
	e = enterDetail(e)
	if ef := footerOf(e); !strings.Contains(ef, "retry") {
		t.Errorf("expected 'space retry' in the detail footer download-error state, got:\n%s", ef)
	}
}

// Slice 14, cycle 5: the post-install chooser shows the ADDED chip and its keys
// — select / choose / finish.
func TestFooterPostInstallContext(t *testing.T) {
	defer gock.Off()
	m := searchWithResults(t, WithAddRunner(nopRunner()))
	m = installFromList(m)

	f := footerOf(m)
	if !strings.Contains(f, "ADDED") {
		t.Errorf("expected ADDED chip, got:\n%s", f)
	}
	for _, want := range []string{"select", "choose", "finish"} {
		if !strings.Contains(f, want) {
			t.Errorf("expected %q in the post-install footer, got:\n%s", want, f)
		}
	}
}

// A failed install titles the chooser honestly, so its footer must read FAILED
// with esc meaning Back (not the success wording ADDED / finish).
func TestFooterPostInstallFailureContext(t *testing.T) {
	defer gock.Off()
	failRunner := func(cmd *exec.Cmd, done func(error) tea.Msg) tea.Cmd {
		return func() tea.Msg { return done(errors.New("npx exploded")) }
	}
	m := searchWithResults(t, WithAddRunner(failRunner))
	m = installFromList(m)

	f := footerOf(m)
	if !strings.Contains(f, "FAILED") {
		t.Errorf("expected FAILED chip on a failed install, got:\n%s", f)
	}
	if strings.Contains(f, "finish") {
		t.Errorf("failed-install footer should not say finish, got:\n%s", f)
	}
	if !strings.Contains(f, "back") {
		t.Errorf("expected esc back on a failed install, got:\n%s", f)
	}
}

func TestFooterHiddenDuringModals(t *testing.T) {
	cases := []struct {
		name  string
		drive func(tea.Model) tea.Model
	}{
		{"palette", func(m tea.Model) tea.Model { return press(m, ":") }},
		{"help", func(m tea.Model) tea.Model { return press(m, "?") }},
		{"confirm", func(m tea.Model) tea.Model { return press(press(m, ":"), "d") }},
		{"wizard", func(m tea.Model) tea.Model { return openWizard(m) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m tea.Model = newTestModel(browseResult())
			m = tc.drive(m)
			if f := footerOf(m); f != "" {
				t.Errorf("expected empty footer while %s is open, got:\n%s", tc.name, f)
			}
		})
	}
}

func TestFooterOccupiesBottomRowAndFrameFits(t *testing.T) {
	const w, h = 100, 40
	var m tea.Model = newTestModel(browseResult())
	m = resize(m, w, h)

	out := view(m)
	if gotH := lipgloss.Height(out); gotH > h {
		t.Errorf("frame height %d exceeds terminal height %d", gotH, h)
	}

	lines := strings.Split(out, "\n")
	// The bottom-most non-blank line is the footer: it carries the SKILLS chip.
	last := ""
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(plain(lines[i])) != "" {
			last = plain(lines[i])
			break
		}
	}
	if !strings.Contains(last, "SKILLS") {
		t.Errorf("expected the footer on the bottom row, got last line:\n%s", last)
	}
}

func TestFooterTruncatesWithHelpPinned(t *testing.T) {
	var m tea.Model = newTestModel(browseResult())
	m = press(m, "3") // DETAILS / SKILL.md: the longest context line

	fullW := lipgloss.Width(m.(Model).renderFooter())
	narrow := fullW - 20 // narrower than the full line, forcing items to drop
	m = resize(m, narrow, 40)

	f := footerOf(m)
	if w := lipgloss.Width(m.(Model).renderFooter()); w > narrow {
		t.Errorf("footer width %d exceeds frame width %d:\n%s", w, narrow, f)
	}
	if !strings.Contains(f, "keys") {
		t.Errorf("expected '? keys' pinned as the final item, got:\n%s", f)
	}
	if !strings.Contains(f, "j/k") {
		t.Errorf("expected the first context key to survive, got:\n%s", f)
	}
	if !strings.Contains(f, "…") {
		t.Errorf("expected an ellipsis where items were dropped, got:\n%s", f)
	}
	if strings.Contains(f, "quit") {
		t.Errorf("expected the global tail (q quit) to be dropped, got:\n%s", f)
	}
}
