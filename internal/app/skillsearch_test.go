package app

import (
	"errors"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/h2non/gock"

	"github.com/makesometh-ing/trainer/internal/marketplace"
	"github.com/makesometh-ing/trainer/internal/skills"
)

// searchFixture is the recorded skills.sh search page, replayed through gock.
// The path is relative to this package dir, where go test runs.
const searchFixture = "../marketplace/testdata/search_react.json"

// downloadFixture is the recorded skills.sh download of the top react result
// (vercel-labs/agent-skills@vercel-react-best-practices), replayed through gock.
const downloadFixture = "../marketplace/testdata/download_vercel-react-best-practices.json"

// emptyFixture is the recorded skills.sh search page for a query that matches
// nothing (skills:[], count:0), replayed through gock.
const emptyFixture = "../marketplace/testdata/search_empty.json"

// fireDwell delivers the dwell message the overlay's tea.Tick would produce ~200ms
// after the selection rests, carrying the overlay's current dwell epoch (as the
// real tick would). Delivering it directly avoids waiting real time.
func fireDwell(m tea.Model) (tea.Model, tea.Cmd) {
	ep := m.(Model).skillSearch.dlEpoch
	return m.Update(dwellMsg{dlEpoch: ep})
}

// typeSearch types a query into the settled overlay's search box one rune at a
// time, discarding the debounce cmds each keystroke returns.
func typeSearch(m tea.Model, s string) tea.Model {
	for _, r := range s {
		m, _ = m.Update(runeKey(r))
	}
	return m
}

// fireDebounce delivers the debounce message the overlay's tea.Tick would
// produce after the pause, carrying the overlay's current epoch (as the real
// tick would). Delivering it directly avoids waiting 300ms of real time.
func fireDebounce(m tea.Model) (tea.Model, tea.Cmd) {
	ep := m.(Model).skillSearch.epoch
	return m.Update(searchDebounceMsg{epoch: ep})
}

// runSearchCmd executes the command the debounce handler returned (a batch of
// the Search request plus the spinner tick) and feeds each resulting message
// back through Update, so the recorded page lands as results. Follow-on cmds
// are discarded so nothing loops.
func runSearchCmd(m tea.Model, cmd tea.Cmd) tea.Model {
	if cmd == nil {
		return m
	}
	msg := cmd()
	if bm, ok := msg.(tea.BatchMsg); ok {
		for _, c := range bm {
			if c == nil {
				continue
			}
			m, _ = m.Update(c())
		}
		return m
	}
	m, _ = m.Update(msg)
	return m
}

// openSkillSearch drives the add flow to the point of opening the Skill Search
// overlay: it sizes the window, opens the entry chooser, moves the cursor onto
// the "Search for skills" option, and presses enter. It returns the model and
// the command enter produced (the first frame tick of the grow animation).
func openSkillSearch(m tea.Model, w, h int) (tea.Model, tea.Cmd) {
	m, _ = m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = openChooser(m)            // :a
	m, _ = m.Update(runeKey('j')) // move onto "Search for skills"
	return m.Update(namedKey(tea.KeyEnter))
}

// settle feeds animation frames until the grow loop stops (returns a nil cmd)
// or a generous cap is reached. It returns the settled model.
func settle(m tea.Model, cmd tea.Cmd) tea.Model {
	for i := 0; i < 5000 && cmd != nil; i++ {
		m, cmd = m.Update(animFrameMsg{})
	}
	return m
}

// Cycle 1: picking "Search for skills" swaps the chooser for the overlay shell
// (rounded, titled "Skill Search") and returns a frame-tick command.
func TestSkillSearchOpensShell(t *testing.T) {
	var m tea.Model = withMarket()

	m, cmd := openSkillSearch(m, 100, 30)

	if cmd == nil {
		t.Fatal("expected a frame-tick command to start the grow animation")
	}
	out := view(m)
	if !strings.Contains(out, "Skill Search") {
		t.Errorf("expected the Skill Search title in the overlay shell, got:\n%s", out)
	}
	if strings.Contains(out, "Enter skill URL or repository") {
		t.Errorf("expected the chooser to be gone once the overlay opens, got:\n%s", out)
	}
}

// Cycle 2: frames grow the shell; once settled the ticker stops (nil cmd), the
// overlay snaps to its target size, and the search box renders.
func TestSkillSearchGrowsAndSettles(t *testing.T) {
	const w, h = 100, 30
	var m tea.Model = withMarket()

	m, _ = openSkillSearch(m, w, h)

	before := lipgloss.Width(m.(Model).renderSkillSearch())
	for i := 0; i < 10; i++ {
		m, _ = m.Update(animFrameMsg{})
	}
	after := lipgloss.Width(m.(Model).renderSkillSearch())
	if after <= before {
		t.Errorf("expected the shell to grow between frames: before=%d after=%d", before, after)
	}

	// Drive to settle. The loop must terminate: a settled overlay returns nil.
	var last tea.Cmd
	settled := false
	for i := 0; i < 5000; i++ {
		m, last = m.Update(animFrameMsg{})
		if last == nil {
			settled = true
			break
		}
	}
	if !settled {
		t.Fatal("expected the frame ticker to stop once the overlay settles")
	}

	shell := m.(Model).renderSkillSearch()
	if gotW := lipgloss.Width(shell); gotW != w-2 {
		t.Errorf("settled shell width = %d, want target %d", gotW, w-2)
	}
	if !strings.Contains(plain(shell), "Search skills…") {
		t.Errorf("expected the search box to render once settled, got:\n%s", plain(shell))
	}
}

// Cycle 3: a window resize mid-grow re-aims the spring so the settled size
// tracks the new terminal size.
func TestSkillSearchResizeMidGrowReAims(t *testing.T) {
	var m tea.Model = withMarket()

	m, _ = openSkillSearch(m, 100, 30)

	// A few frames into the grow, then resize larger.
	for i := 0; i < 3; i++ {
		m, _ = m.Update(animFrameMsg{})
	}
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	// Keep animating until it settles at the new target.
	var last tea.Cmd
	for i := 0; i < 5000; i++ {
		m, last = m.Update(animFrameMsg{})
		if last == nil {
			break
		}
	}

	if gotW := lipgloss.Width(m.(Model).renderSkillSearch()); gotW != 120-2 {
		t.Errorf("settled shell width = %d, want re-aimed target %d", gotW, 120-2)
	}
}

// A resize after the grow has settled must resize the overlay: the tween is
// stopped, so the size snaps to the new terminal instead of staying frozen at
// the old size (both shrinking and growing the terminal).
func TestSkillSearchResizeAfterSettle(t *testing.T) {
	var m tea.Model = withMarket()
	m = settle(openSkillSearch(m, 100, 30))

	// Shrink the terminal below the settled size (staying above the minWidth
	// clamp so the target is exactly width-2).
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	if gotW := lipgloss.Width(m.(Model).renderSkillSearch()); gotW != 80-2 {
		t.Errorf("after shrink, shell width = %d, want %d", gotW, 80-2)
	}

	// Grow it again.
	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 44})
	if gotW := lipgloss.Width(m.(Model).renderSkillSearch()); gotW != 140-2 {
		t.Errorf("after grow, shell width = %d, want %d", gotW, 140-2)
	}
}

// Slice 6, cycle 1: a query under two characters shows the hint in the results
// pane and never fires a request (a pending mock stays unconsumed).
func TestSkillSearchShortQueryHintNoRequest(t *testing.T) {
	defer gock.Off()
	gock.New("https://skills.sh").
		Get("/api/search").
		Reply(200).
		File(searchFixture)

	var m tea.Model = withMarket()
	m = settle(openSkillSearch(m, 100, 30))

	m = typeSearch(m, "a")

	out := plain(view(m))
	if !strings.Contains(out, "Type at least 2 characters") {
		t.Errorf("expected the short-query hint, got:\n%s", out)
	}
	if gock.IsDone() {
		t.Error("expected no Search request for a one-character query, but the mock was consumed")
	}
}

// Slice 6, cycle 2: a two-plus-character query, 300ms after the last keystroke,
// renders the ranked page from the recorded fixture.
func TestSkillSearchRendersResultsAfterDebounce(t *testing.T) {
	defer gock.Off()
	gock.New("https://skills.sh").
		Get("/api/search").
		Reply(200).
		File(searchFixture)

	var m tea.Model = withMarket()
	m = settle(openSkillSearch(m, 100, 30))

	m = typeSearch(m, "react")
	m, cmd := fireDebounce(m)
	m = runSearchCmd(m, cmd)

	out := plain(view(m))
	if !strings.Contains(out, "vercel-react-best-practices") {
		t.Errorf("expected the fixture's top result to render, got:\n%s", out)
	}
	if !strings.Contains(out, "vercel-react-native-skills") {
		t.Errorf("expected another fixture result to render, got:\n%s", out)
	}
	if !gock.IsDone() {
		t.Error("expected the Search request to be made after the debounce")
	}
}

// Slice 6, cycle 3: a keystroke after a debounce was scheduled supersedes it —
// delivering the stale debounce fires no request; the current one does.
func TestSkillSearchSupersededDebounceFiresNoRequest(t *testing.T) {
	defer gock.Off()
	gock.New("https://skills.sh").
		Get("/api/search").
		Reply(200).
		File(searchFixture)

	var m tea.Model = withMarket()
	m = settle(openSkillSearch(m, 100, 30))

	m = typeSearch(m, "reac")
	staleEpoch := m.(Model).skillSearch.epoch
	m = typeSearch(m, "t") // supersedes: epoch advances

	// The stale debounce (older epoch) must fire no request.
	m, _ = m.Update(searchDebounceMsg{epoch: staleEpoch})
	if gock.IsDone() {
		t.Fatal("expected the superseded debounce to fire no request, but the mock was consumed")
	}

	// The current debounce fires the request and results land.
	m, cmd := fireDebounce(m)
	m = runSearchCmd(m, cmd)
	if !gock.IsDone() {
		t.Error("expected the current debounce to fire the request")
	}
	if !strings.Contains(plain(view(m)), "vercel-react-best-practices") {
		t.Errorf("expected results after the current debounce, got:\n%s", plain(view(m)))
	}
}

// Slice 6, cycle 4: a keystroke while a request is in flight supersedes it —
// the in-flight request's result (older epoch) is dropped and never renders,
// while the newer query's results win.
func TestSkillSearchInFlightResultDroppedNewerWins(t *testing.T) {
	defer gock.Off()
	gock.New("https://skills.sh").
		Get("/api/search").
		Reply(200).
		File(searchFixture)

	var m tea.Model = withMarket()
	m = settle(openSkillSearch(m, 100, 30))

	// Start a request: the overlay is now loading under this epoch.
	m = typeSearch(m, "react")
	m, _ = fireDebounce(m)
	inFlightEpoch := m.(Model).skillSearch.epoch

	// A keystroke supersedes the in-flight request (cancels it, advances epoch).
	m = typeSearch(m, "x")

	// The in-flight request's late result carries the stale epoch and is dropped.
	m, _ = m.Update(searchResultsMsg{
		epoch:  inFlightEpoch,
		skills: []marketplace.MarketplaceSkill{{Name: "stale-only-skill"}},
	})
	if strings.Contains(plain(view(m)), "stale-only-skill") {
		t.Fatalf("expected the stale in-flight result to be dropped, got:\n%s", plain(view(m)))
	}

	// The newer query's request lands and its results render.
	m, cmd := fireDebounce(m)
	m = runSearchCmd(m, cmd)
	if !strings.Contains(plain(view(m)), "vercel-react-best-practices") {
		t.Errorf("expected the newer query's results to win, got:\n%s", plain(view(m)))
	}
}

// Slice 6, cycle 5: a spinner shows while a request is active and is gone once
// results land.
func TestSkillSearchSpinnerWhileLoading(t *testing.T) {
	defer gock.Off()
	gock.New("https://skills.sh").
		Get("/api/search").
		Reply(200).
		File(searchFixture)

	var m tea.Model = withMarket()
	m = settle(openSkillSearch(m, 100, 30))

	m = typeSearch(m, "react")
	m, cmd := fireDebounce(m)

	// Request is in flight (results not delivered yet): the spinner shows and no
	// result rows are drawn.
	loading := plain(view(m))
	if !strings.Contains(loading, spinnerFrames[0]) {
		t.Errorf("expected the loading spinner while the request is active, got:\n%s", loading)
	}
	if strings.Contains(loading, "vercel-react-best-practices") {
		t.Errorf("did not expect results while still loading, got:\n%s", loading)
	}

	// Once results land the spinner is gone and rows render.
	m = runSearchCmd(m, cmd)
	landed := plain(view(m))
	if strings.Contains(landed, spinnerFrames[0]) {
		t.Errorf("expected the spinner to stop once results land, got:\n%s", landed)
	}
	if !strings.Contains(landed, "vercel-react-best-practices") {
		t.Errorf("expected results after the spinner stops, got:\n%s", landed)
	}
}

// searchWithResults drives the overlay to a settled shell with the recorded
// react page loaded as results (box focused, default sort). The caller must
// defer gock.Off().
func searchWithResults(t *testing.T, opts ...Option) tea.Model {
	t.Helper()
	gock.New("https://skills.sh").
		Get("/api/search").
		Reply(200).
		File(searchFixture)

	var m tea.Model = withMarket(opts...)
	m = settle(openSkillSearch(m, 100, 30))
	m = typeSearch(m, "react")
	m, cmd := fireDebounce(m)
	return runSearchCmd(m, cmd)
}

var installsRE = regexp.MustCompile(`([\d,]+) installs`)

// installNums pulls the Install Count of each rendered row, in render order.
func installNums(s string) []int {
	ms := installsRE.FindAllStringSubmatch(s, -1)
	out := make([]int, 0, len(ms))
	for _, sub := range ms {
		n, _ := strconv.Atoi(strings.ReplaceAll(sub[1], ",", ""))
		out = append(out, n)
	}
	return out
}

// assertDescending fails if the numbers are not in non-increasing order.
func assertDescending(t *testing.T, nums []int) {
	t.Helper()
	for i := 1; i < len(nums); i++ {
		if nums[i] > nums[i-1] {
			t.Errorf("expected rows in descending Install Count, got %v", nums)
			return
		}
	}
}

// Slice 7, cycle 1: a result renders as a two-line row — the name on line one,
// `source · <Install Count> installs` (comma-separated) on line two.
func TestSkillSearchRowShowsNameAndInstalls(t *testing.T) {
	defer gock.Off()
	m := searchWithResults(t)

	out := plain(view(m))
	if !strings.Contains(out, "vercel-react-best-practices") {
		t.Errorf("expected the skill name on the row, got:\n%s", out)
	}
	if !strings.Contains(out, "vercel-labs/agent-skills · 540,366 installs") {
		t.Errorf("expected `source · N,NNN installs` with comma separators, got:\n%s", out)
	}
}

// Slice 7, cycle 2: Enter from the box focuses the results list; j moves the
// selection and the window follows so a row far down the list becomes visible.
func TestSkillSearchListNavigationWindows(t *testing.T) {
	defer gock.Off()
	m := searchWithResults(t)

	// The last skill by Install Count is off the initial window.
	if strings.Contains(plain(view(m)), "json-render-react") {
		t.Fatalf("expected the last skill to be off the initial window, got:\n%s", plain(view(m)))
	}

	// Enter jumps into the list; j walks the selection to the bottom.
	m, _ = m.Update(namedKey(tea.KeyEnter))
	for i := 0; i < 24; i++ {
		m, _ = m.Update(runeKey('j'))
	}

	if !strings.Contains(plain(view(m)), "json-render-react") {
		t.Errorf("expected navigation to window the last skill into view, got:\n%s", plain(view(m)))
	}
}

// Slice 7, cycle 3: with the fixture's out-of-order Install Counts, the default
// order is Popularity descending — visible rows run high to low.
func TestSkillSearchDefaultOrderPopularityDesc(t *testing.T) {
	defer gock.Off()
	m := searchWithResults(t)

	nums := installNums(plain(view(m)))
	if len(nums) < 5 {
		t.Fatalf("expected several rows to assert order over, got %v", nums)
	}
	if nums[0] != 540366 {
		t.Errorf("expected the most-installed skill first, got %d", nums[0])
	}
	assertDescending(t, nums)
}

// Slice 7, cycle 4: in the list, n sorts by Name A–Z, n again toggles to Z–A, p
// returns to Popularity, and r restores the Marketplace's own (API) order.
func TestSkillSearchSortSwitchingAndToggle(t *testing.T) {
	defer gock.Off()
	m := searchWithResults(t)
	m, _ = m.Update(namedKey(tea.KeyEnter)) // focus the list

	// n → Name ascending: an alphabetically-early skill windows into view that
	// popularity kept off-window.
	m, _ = m.Update(runeKey('n'))
	if !strings.Contains(plain(view(m)), "clerk-react-patterns") {
		t.Errorf("expected Name A–Z to surface an early name, got:\n%s", plain(view(m)))
	}

	// n again → Name descending: an alphabetically-late skill leads instead.
	m, _ = m.Update(runeKey('n'))
	if !strings.Contains(plain(view(m)), "upgrading-react-native") {
		t.Errorf("expected the same key to toggle to Z–A, got:\n%s", plain(view(m)))
	}

	// p → Popularity descending again.
	m, _ = m.Update(runeKey('p'))
	nums := installNums(plain(view(m)))
	if len(nums) == 0 || nums[0] != 540366 {
		t.Errorf("expected p to return to Popularity desc, got %v", nums)
	}
	assertDescending(t, nums)

	// r → Relevance (API order): a low-install skill that sits high in the API
	// page is visible again, which Popularity had pushed off-window.
	m, _ = m.Update(runeKey('r'))
	if !strings.Contains(plain(view(m)), "3,989 installs") {
		t.Errorf("expected r to restore the Marketplace's API order, got:\n%s", plain(view(m)))
	}
}

// Slice 7, cycle 5: sort letters act only in list focus. In the box, n types
// into the query and leaves the result order untouched.
func TestSkillSearchSortKeysAreListScoped(t *testing.T) {
	defer gock.Off()
	m := searchWithResults(t) // box focused, results in Popularity desc

	m, _ = m.Update(runeKey('n')) // in the box this is a query character

	out := plain(view(m))
	if !strings.Contains(out, "reactn") {
		t.Errorf("expected n to type into the query in box focus, got:\n%s", out)
	}
	nums := installNums(out)
	if len(nums) == 0 || nums[0] != 540366 {
		t.Errorf("expected the order to stay Popularity desc (n did not sort), got %v", nums)
	}
	assertDescending(t, nums)
}

// Slice 8, cycle 1: resting the selection ~200ms fetches the skill's file tree
// with one download call and renders SKILL.md in the detail pane.
func TestSkillSearchDwellDownloadsAndRendersSkillMD(t *testing.T) {
	defer gock.Off()
	m := searchWithResults(t) // selection rests on the top result

	gock.New("https://skills.sh").
		Get("/api/download/vercel-labs/agent-skills/vercel-react-best-practices").
		Reply(200).
		File(downloadFixture)

	m, cmd := fireDwell(m)
	m = runSearchCmd(m, cmd)

	out := plain(view(m))
	if !strings.Contains(out, "Comprehensive") {
		t.Errorf("expected SKILL.md to render in the detail pane after the dwell, got:\n%s", out)
	}
	if !gock.IsDone() {
		t.Error("expected the download request to be made on dwell")
	}
}

// Slice 8, cycle 2: a spinner shows in the detail pane while the download is in
// flight and is gone once SKILL.md lands.
func TestSkillSearchDetailSpinnerWhileDownloading(t *testing.T) {
	defer gock.Off()
	m := searchWithResults(t)

	gock.New("https://skills.sh").
		Get("/api/download/vercel-labs/agent-skills/vercel-react-best-practices").
		Reply(200).
		File(downloadFixture)

	m, cmd := fireDwell(m)

	// Download in flight (not delivered yet): the detail spinner shows and no
	// SKILL.md is drawn.
	loading := plain(view(m))
	if !strings.Contains(loading, spinnerFrames[0]) {
		t.Errorf("expected the detail spinner while the download is active, got:\n%s", loading)
	}
	if strings.Contains(loading, "Comprehensive") {
		t.Errorf("did not expect SKILL.md while still downloading, got:\n%s", loading)
	}

	// Once files land the spinner is gone and SKILL.md renders.
	m = runSearchCmd(m, cmd)
	landed := plain(view(m))
	if strings.Contains(landed, spinnerFrames[0]) {
		t.Errorf("expected the detail spinner to stop once files land, got:\n%s", landed)
	}
	if !strings.Contains(landed, "Comprehensive") {
		t.Errorf("expected SKILL.md after the spinner stops, got:\n%s", landed)
	}
}

// Slice 8, cycle 3: moving the selection cancels the in-flight download; the
// superseded skill's content never renders even when its late result arrives.
func TestSkillSearchMovingSelectionDropsStaleDownload(t *testing.T) {
	defer gock.Off()
	m := searchWithResults(t)

	// Start a download for the top result.
	m, _ = fireDwell(m)
	staleEpoch := m.(Model).skillSearch.dlEpoch

	// Move the selection before the download lands: focus the list and press j.
	m, _ = m.Update(namedKey(tea.KeyEnter))
	m, _ = m.Update(runeKey('j'))

	// The superseded download's late result carries the stale dwell epoch and is
	// dropped, so its content never appears in the detail pane.
	m, _ = m.Update(downloadResultsMsg{
		dlEpoch: staleEpoch,
		ref:     "vercel-labs/agent-skills@vercel-react-best-practices",
		files: marketplace.SkillFiles{Files: []marketplace.File{
			{Path: "SKILL.md", Contents: "---\nname: stale\n---\n\nStaleDetailMarker body"},
		}},
	})

	out := plain(view(m))
	if strings.Contains(out, "StaleDetailMarker") {
		t.Errorf("expected the superseded download result to be dropped after moving, got:\n%s", out)
	}
}

// Slice 8, cycle 4: a previously downloaded skill re-renders from the session
// cache with no second request. Only one download mock is registered; a second
// request would have no match.
func TestSkillSearchCacheHitMakesNoSecondRequest(t *testing.T) {
	defer gock.Off()
	m := searchWithResults(t)

	gock.New("https://skills.sh").
		Get("/api/download/vercel-labs/agent-skills/vercel-react-best-practices").
		Reply(200).
		File(downloadFixture)

	// First dwell downloads and caches the top result.
	m, cmd := fireDwell(m)
	m = runSearchCmd(m, cmd)
	if !gock.IsDone() {
		t.Fatal("expected the first dwell to consume the download mock")
	}

	// Move away to the second result and back to the top one.
	m, _ = m.Update(namedKey(tea.KeyEnter)) // focus the list
	m, _ = m.Update(runeKey('j'))           // to the second skill
	m, _ = m.Update(runeKey('k'))           // back to the cached top skill

	// A dwell on the cached skill issues no request (nil command) and re-renders
	// from memory.
	next, dcmd := fireDwell(m)
	if dcmd != nil {
		t.Error("expected no download command for a cached skill")
	}
	if !strings.Contains(plain(view(next)), "Comprehensive") {
		t.Errorf("expected the cached skill's SKILL.md to re-render, got:\n%s", plain(view(next)))
	}
}

// mktDetailFiles is a hand-authored download tree with all four tabs populated.
// The recorded vercel fixture has only SKILL.md (its rules/ dir is dropped by
// Classify), so a fixture that exercises References/Scripts/Assets must be
// supplied by hand. second.md is deliberately tall so content-subfocus scrolling
// can reveal its bottom marker.
func mktDetailFiles() marketplace.SkillFiles {
	tall := "# SecondHeadingMarker\n\n" + strings.Repeat("Filler paragraph.\n\n", 40) + "SecondBottomMarker\n"
	return marketplace.SkillFiles{Files: []marketplace.File{
		{Path: "SKILL.md", Contents: "---\nname: react\n---\n\nSkillBodyMarker"},
		{Path: "references/guide.md", Contents: "# GuideHeadingMarker\n\nGuide body text."},
		{Path: "references/second.md", Contents: tall},
		{Path: "scripts/run.sh", Contents: "#!/bin/bash\necho FirstScriptMarker\n"},
		{Path: "scripts/other.sh", Contents: "#!/bin/bash\necho SecondScriptMarker\n"},
		{Path: "assets/logo.png", Contents: "PNGDATA"},
	}}
}

// seedDownload caches a hand-authored file tree for the currently selected
// Marketplace Skill, as a landed download would, so a test can drive the Skill
// Detail tabs without a real download call.
func seedDownload(t *testing.T, m tea.Model, files marketplace.SkillFiles) tea.Model {
	t.Helper()
	o := m.(Model).skillSearch
	ref := o.results[o.selected].InstallRef()
	m, _ = m.Update(downloadResultsMsg{dlEpoch: o.dlEpoch, ref: ref, files: files})
	return m
}

// enterDetail drives a settled overlay with results into the Skill Detail zone:
// Enter focuses the results list, l opens the detail.
func enterDetail(m tea.Model) tea.Model {
	m, _ = m.Update(namedKey(tea.KeyEnter)) // focus the results list
	m, _ = m.Update(runeKey('l'))           // open the Skill Detail
	return m
}

// Slice 9, cycle 1: l enters the Skill Detail; r shows the References tab with
// the downloaded file list (prefix-stripped names from Classify).
func TestSkillSearchDetailReferencesFileList(t *testing.T) {
	defer gock.Off()
	m := searchWithResults(t)
	m = seedDownload(t, m, mktDetailFiles())

	m = enterDetail(m)
	m, _ = m.Update(runeKey('r')) // References tab

	out := plain(view(m))
	if !strings.Contains(out, "guide.md") {
		t.Errorf("expected the References file list (prefix-stripped names), got:\n%s", out)
	}
}

// Slice 9, cycle 2: a Markdown reference renders via Glamour, a script via
// Chroma, and an asset shows "No preview available" — all over the in-memory
// downloaded contents.
func TestSkillSearchDetailRendersByFileKind(t *testing.T) {
	defer gock.Off()
	m := searchWithResults(t)
	m = seedDownload(t, m, mktDetailFiles())
	m = enterDetail(m)

	m, _ = m.Update(runeKey('r')) // References: guide.md rendered via Glamour
	if out := plain(view(m)); !strings.Contains(out, "GuideHeadingMarker") {
		t.Errorf("expected the Markdown reference to render via Glamour, got:\n%s", out)
	}

	m, _ = m.Update(runeKey('s')) // Scripts: run.sh highlighted via Chroma
	if out := plain(view(m)); !strings.Contains(out, "FirstScriptMarker") {
		t.Errorf("expected the script to render via Chroma, got:\n%s", out)
	}

	m, _ = m.Update(runeKey('a')) // Assets: no preview
	if out := plain(view(m)); !strings.Contains(out, "No preview available") {
		t.Errorf("expected assets to show `No preview available`, got:\n%s", out)
	}
}

// Slice 9, cycle 3: tab toggles the file-list / content subfocus. In list
// subfocus j moves the file selection; in content subfocus j scrolls the
// content, revealing the bottom of a tall reference.
func TestSkillSearchDetailSubfocusToggle(t *testing.T) {
	defer gock.Off()
	m := searchWithResults(t)
	m = seedDownload(t, m, mktDetailFiles())
	m = enterDetail(m)
	m, _ = m.Update(runeKey('r')) // References, list subfocus, guide.md selected

	// List subfocus: j moves the selection onto the second (tall) reference.
	m, _ = m.Update(runeKey('j'))
	if out := plain(view(m)); !strings.Contains(out, "SecondHeadingMarker") {
		t.Errorf("expected j in list subfocus to move the file selection, got:\n%s", out)
	}
	if out := plain(view(m)); strings.Contains(out, "SecondBottomMarker") {
		t.Fatalf("expected the tall reference's bottom to be below the fold initially, got:\n%s", out)
	}

	// Content subfocus: j scrolls the content down to the bottom marker.
	m, _ = m.Update(namedKey(tea.KeyTab))
	for i := 0; i < 80; i++ {
		m, _ = m.Update(runeKey('j'))
	}
	if out := plain(view(m)); !strings.Contains(out, "SecondBottomMarker") {
		t.Errorf("expected j in content subfocus to scroll to the bottom, got:\n%s", out)
	}
}

// Slice 9, cycle 4: switching tabs resets the file selection and the subfocus,
// and h returns to the results list (where the sort keys act again).
func TestSkillSearchDetailTabResetAndBackToList(t *testing.T) {
	defer gock.Off()
	m := searchWithResults(t)
	m = seedDownload(t, m, mktDetailFiles())
	m = enterDetail(m)

	// References: move onto the second file, then take content subfocus.
	m, _ = m.Update(runeKey('r'))
	m, _ = m.Update(runeKey('j'))         // second.md selected
	m, _ = m.Update(namedKey(tea.KeyTab)) // content subfocus

	// Switch to Scripts: the file selection resets to the first script (its
	// content, not the second script's, renders).
	m, _ = m.Update(runeKey('s'))
	out := plain(view(m))
	if !strings.Contains(out, "FirstScriptMarker") {
		t.Errorf("expected the tab switch to reset the file selection to the first file, got:\n%s", out)
	}
	if strings.Contains(out, "SecondScriptMarker") {
		t.Errorf("expected the second file not to be selected after a tab switch, got:\n%s", out)
	}

	// The subfocus reset to the file list: j now moves the file selection again.
	m, _ = m.Update(runeKey('j'))
	if out := plain(view(m)); !strings.Contains(out, "SecondScriptMarker") {
		t.Errorf("expected the subfocus to reset to the file list (j moves files), got:\n%s", out)
	}

	// h returns to the results list, where n sorts by Name (a list-only key).
	m, _ = m.Update(runeKey('h'))
	m, _ = m.Update(runeKey('n'))
	if out := plain(view(m)); !strings.Contains(out, "clerk-react-patterns") {
		t.Errorf("expected h to return to the list zone where n sorts by Name, got:\n%s", out)
	}
}

// Slice 9, cycle 5: re-opening a file renders identically and issues no second
// download — the tree is fetched once and rendering is memoized. Only one
// download mock is registered; a second request would find no match.
func TestSkillSearchDetailReopenNoRefetch(t *testing.T) {
	defer gock.Off()
	m := searchWithResults(t)

	gock.New("https://skills.sh").
		Get("/api/download/vercel-labs/agent-skills/vercel-react-best-practices").
		Reply(200).
		File(downloadFixture)

	// One download lands the file tree.
	m, cmd := fireDwell(m)
	m = runSearchCmd(m, cmd)
	if !gock.IsDone() {
		t.Fatal("expected the file tree to be downloaded once")
	}

	// Open the detail on SKILL.md and capture its render.
	m = enterDetail(m)
	first := plain(view(m))

	// Round-trip to another tab and back to SKILL.md.
	m, _ = m.Update(runeKey('r')) // References (empty for this skill)
	m, _ = m.Update(runeKey('i')) // back to SKILL.md
	again := plain(view(m))

	if first != again {
		t.Errorf("expected SKILL.md to re-render identically, got first:\n%s\n\nagain:\n%s", first, again)
	}
	if !gock.IsDone() {
		t.Error("expected no second download when re-opening the file")
	}
}

// Slice 10, cycle 1: Enter on a Marketplace Skill in the results list installs
// it through the existing add seam — `npx skills@latest add <source>@<skillId>`
// with the exact ref and no SSH key (no GIT_SSH_COMMAND).
func TestSkillSearchInstallFromListRunsAddCommand(t *testing.T) {
	defer gock.Off()

	var ranArgs []string
	var ranEnv []string
	runner := func(cmd *exec.Cmd, done func(error) tea.Msg) tea.Cmd {
		ranArgs = cmd.Args
		ranEnv = cmd.Env
		return func() tea.Msg { return done(nil) }
	}

	m := searchWithResults(t, WithAddRunner(runner)) // box focused, top result selected
	m, _ = m.Update(namedKey(tea.KeyEnter))          // focus the results list
	m.Update(namedKey(tea.KeyEnter))                 // install the selected skill

	wantArgs := []string{"npx", "skills@latest", "add", "vercel-labs/agent-skills@vercel-react-best-practices"}
	if !slices.Equal(ranArgs, wantArgs) {
		t.Errorf("ran args = %v, want %v", ranArgs, wantArgs)
	}
	if envHasAnySSHCommand(ranEnv) {
		t.Errorf("expected no GIT_SSH_COMMAND for a Marketplace install, env=%v", ranEnv)
	}
}

// Slice 10, cycle 2: Enter from the Skill Detail installs the same skill through
// the add seam with the same ref.
func TestSkillSearchInstallFromDetailRunsAddCommand(t *testing.T) {
	defer gock.Off()

	var ranArgs []string
	runner := func(cmd *exec.Cmd, done func(error) tea.Msg) tea.Cmd {
		ranArgs = cmd.Args
		return func() tea.Msg { return done(nil) }
	}

	m := searchWithResults(t, WithAddRunner(runner))
	m = seedDownload(t, m, mktDetailFiles())
	m = enterDetail(m)               // Enter → list, l → detail
	m.Update(namedKey(tea.KeyEnter)) // install from the detail zone

	wantArgs := []string{"npx", "skills@latest", "add", "vercel-labs/agent-skills@vercel-react-best-practices"}
	if !slices.Equal(ranArgs, wantArgs) {
		t.Errorf("ran args = %v, want %v", ranArgs, wantArgs)
	}
}

// Slice 10, cycle 3: installing does not clear the Skill Search overlay (unlike
// the manual wizard's runAdd) — it stays rendered while the install runs.
func TestSkillSearchInstallLeavesOverlayOpen(t *testing.T) {
	defer gock.Off()

	runner := func(cmd *exec.Cmd, done func(error) tea.Msg) tea.Cmd {
		return func() tea.Msg { return done(nil) }
	}

	m := searchWithResults(t, WithAddRunner(runner))
	m, _ = m.Update(namedKey(tea.KeyEnter)) // focus the results list
	m, _ = m.Update(namedKey(tea.KeyEnter)) // install

	if m.(Model).skillSearch == nil {
		t.Fatal("expected the Skill Search overlay to stay open while the install runs")
	}
	if !strings.Contains(plain(view(m)), "vercel-react-best-practices") {
		t.Errorf("expected the overlay's results to stay rendered after install, got:\n%s", plain(view(m)))
	}
}

// nopRunner is an AddRunner that just fires the completion message, so an
// install through the search seam delivers addFinishedMsg without running npx.
func nopRunner() AddRunner {
	return func(cmd *exec.Cmd, done func(error) tea.Msg) tea.Cmd {
		return func() tea.Msg { return done(nil) }
	}
}

// installFromList installs the selected Marketplace Skill from the results list
// and pumps the completion message, so the model reflects the post-install step.
func installFromList(m tea.Model) tea.Model {
	m, _ = m.Update(namedKey(tea.KeyEnter)) // focus the results list
	next, cmd := m.Update(namedKey(tea.KeyEnter))
	return pump(next, cmd) // deliver addFinishedMsg
}

// Slice 11, cycle 1: installing with the overlay open shows the post-install
// chooser (Find more skills / Finish) and does not rescan yet; the overlay
// state stays preserved underneath.
func TestSkillSearchPostInstallChooserShownNoRescan(t *testing.T) {
	defer gock.Off()
	rescanned := false
	rescan := func() []skills.ScanResult {
		rescanned = true
		return []skills.ScanResult{browseResult()}
	}

	m := searchWithResults(t, WithAddRunner(nopRunner()), WithRescan(rescan))
	m = installFromList(m)

	out := plain(view(m))
	if !strings.Contains(out, "Find more skills") || !strings.Contains(out, "Finish") {
		t.Errorf("expected the post-install chooser after install, got:\n%s", out)
	}
	if rescanned {
		t.Error("expected no rescan while the post-install chooser is open")
	}
	if m.(Model).skillSearch == nil {
		t.Error("expected the overlay state to be preserved under the post-install chooser")
	}
}

// Slice 11, cycle 2: "Find more skills" returns to the search box with the query,
// results, and download cache intact — a re-selected downloaded skill issues no
// new request.
func TestSkillSearchFindMorePreservesState(t *testing.T) {
	defer gock.Off()
	m := searchWithResults(t, WithAddRunner(nopRunner()))

	// Download the top result once so the session cache holds it.
	gock.New("https://skills.sh").
		Get("/api/download/vercel-labs/agent-skills/vercel-react-best-practices").
		Reply(200).
		File(downloadFixture)
	m, cmd := fireDwell(m)
	m = runSearchCmd(m, cmd)
	if !gock.IsDone() {
		t.Fatal("expected the first dwell to consume the download mock")
	}

	m = installFromList(m)

	// Pick "Find more skills" (the default cursor position).
	m, _ = m.Update(namedKey(tea.KeyEnter))

	out := plain(view(m))
	if !strings.Contains(out, "vercel-react-best-practices") {
		t.Errorf("expected the results to be preserved after Find more, got:\n%s", out)
	}
	if strings.Contains(out, "Find more skills") {
		t.Errorf("expected the post-install chooser to be gone after Find more, got:\n%s", out)
	}

	// A dwell on the still-cached selection issues no request (nil command).
	if _, dcmd := fireDwell(m); dcmd != nil {
		t.Error("expected no new download for the cached skill after Find more")
	}
}

// Slice 11, cycle 3: "Finish" rescans exactly once, closes the overlay, and
// lands the browser selection on the newly installed skill by name.
func TestSkillSearchFinishRescansAndLandsOnNewSkill(t *testing.T) {
	defer gock.Off()
	rescanCount := 0
	rescan := func() []skills.ScanResult {
		rescanCount++
		return []skills.ScanResult{{
			Scope: skills.Scope{Name: ".agents", Section: skills.SectionGlobal},
			Skills: []skills.Skill{
				{Name: "some-other-skill", Path: "/root/other"},
				{Name: "vercel-react-best-practices", Path: "/root/vrbp",
					Lock: &skills.LockEntry{Source: "vercel-labs/agent-skills"}},
			},
		}}
	}

	m := searchWithResults(t, WithAddRunner(nopRunner()), WithRescan(rescan))
	m = installFromList(m) // installs the top result, opens the post-install chooser

	// Move to "Finish" and pick it.
	m, _ = m.Update(runeKey('j'))
	m, _ = m.Update(namedKey(tea.KeyEnter))

	if m.(Model).skillSearch != nil {
		t.Error("expected Finish to close the Skill Search overlay")
	}
	if rescanCount != 1 {
		t.Errorf("expected exactly one rescan on Finish, got %d", rescanCount)
	}
	out := plain(view(m))
	if !strings.Contains(out, "vercel-react-best-practices") {
		t.Errorf("expected the browser to render the rescanned skills, got:\n%s", out)
	}
	// The selection landed on the newly installed skill: its detail path shows,
	// not the first skill's.
	if !strings.Contains(out, "/root/vrbp") {
		t.Errorf("expected the selection to land on the newly installed skill, got:\n%s", out)
	}
}

// Slice 11, cycle 3b: Esc on the post-install chooser is Finish — it rescans and
// closes the overlay.
func TestSkillSearchPostInstallEscFinishes(t *testing.T) {
	defer gock.Off()
	rescanCount := 0
	rescan := func() []skills.ScanResult {
		rescanCount++
		return []skills.ScanResult{browseResult()}
	}

	m := searchWithResults(t, WithAddRunner(nopRunner()), WithRescan(rescan))
	m = installFromList(m)

	m, _ = m.Update(namedKey(tea.KeyEsc))

	if m.(Model).skillSearch != nil {
		t.Error("expected Esc on the post-install chooser to close the overlay")
	}
	if rescanCount != 1 {
		t.Errorf("expected Esc to rescan exactly once, got %d", rescanCount)
	}
}

// Slice 11, cycle 4: the manual (overlay-less) add path is untouched —
// addFinishedMsg with no overlay open rescans immediately and opens no chooser.
func TestManualAddFinishRefreshesImmediately(t *testing.T) {
	rescanned := false
	rescan := func() []skills.ScanResult {
		rescanned = true
		return []skills.ScanResult{{
			Scope:  skills.Scope{Name: ".agents", Section: skills.SectionGlobal},
			Skills: []skills.Skill{{Name: "charlie", Path: "/root/charlie"}},
		}}
	}

	var m tea.Model = newTestModel(browseResult(), WithRescan(rescan))
	m, _ = m.Update(addFinishedMsg{})

	if !rescanned {
		t.Error("expected an overlay-less addFinishedMsg to rescan immediately")
	}
	if m.(Model).chooser != nil {
		t.Error("expected no post-install chooser on the manual add path")
	}
	if !strings.Contains(view(m), "charlie") {
		t.Errorf("expected the refreshed list after the manual add, got:\n%s", view(m))
	}
}

// spaceKey presses the retry key (space) and returns the model plus the command
// it produced, so a test can pump a retried request.
func spaceKey(m tea.Model) (tea.Model, tea.Cmd) {
	return m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
}

// Slice 12, cycle 1: a query that returns zero results renders the empty-state
// message naming the query, not a blank results pane.
func TestSkillSearchEmptyResultsMessage(t *testing.T) {
	defer gock.Off()
	gock.New("https://skills.sh").
		Get("/api/search").
		Reply(200).
		File(emptyFixture)

	var m tea.Model = withMarket()
	m = settle(openSkillSearch(m, 100, 30))

	m = typeSearch(m, "zzqq")
	m, cmd := fireDebounce(m)
	m = runSearchCmd(m, cmd)

	out := plain(view(m))
	if !strings.Contains(out, `No skills found for "zzqq"`) {
		t.Errorf("expected the empty-state message naming the query, got:\n%s", out)
	}
	if !gock.IsDone() {
		t.Error("expected the Search request to be made")
	}
}

// Slice 12, cycle 2: a failed search renders the retry hint in the results pane;
// space re-fires the search and the recovered request lands results.
func TestSkillSearchSearchFailureRetry(t *testing.T) {
	defer gock.Off()
	// The first request fails; the retried request returns the good fixture.
	gock.New("https://skills.sh").Get("/api/search").Reply(500)
	gock.New("https://skills.sh").Get("/api/search").Reply(200).File(searchFixture)

	var m tea.Model = withMarket()
	m = settle(openSkillSearch(m, 100, 30))

	m = typeSearch(m, "react")
	m, cmd := fireDebounce(m)
	m = runSearchCmd(m, cmd)

	if out := plain(view(m)); !strings.Contains(out, "Search failed — space to retry") {
		t.Fatalf("expected the search failure hint, got:\n%s", out)
	}

	// Retry is a results-list / detail key (in the box a space types into the
	// query); focus the list, then space re-fires the search and the second mock
	// returns results.
	m, _ = m.Update(namedKey(tea.KeyEnter)) // box → results list
	m, cmd = spaceKey(m)
	m = runSearchCmd(m, cmd)
	if out := plain(view(m)); !strings.Contains(out, "vercel-react-best-practices") {
		t.Errorf("expected the retried search to render results, got:\n%s", out)
	}
	if !gock.IsDone() {
		t.Error("expected both the failed and the retried Search request to be made")
	}
}

// Slice 12, cycle 3: a failed download renders the retry hint in the detail
// pane; space re-fires the download and SKILL.md lands.
func TestSkillSearchDownloadFailureRetry(t *testing.T) {
	defer gock.Off()
	m := searchWithResults(t) // selection rests on the top result, dwell pending

	dl := "/api/download/vercel-labs/agent-skills/vercel-react-best-practices"
	gock.New("https://skills.sh").Get(dl).Reply(500)
	gock.New("https://skills.sh").Get(dl).Reply(200).File(downloadFixture)

	m, cmd := fireDwell(m)
	m = runSearchCmd(m, cmd)

	if out := plain(view(m)); !strings.Contains(out, "Couldn't load files — space to retry") {
		t.Fatalf("expected the download failure hint, got:\n%s", out)
	}

	// Retry is a results-list / detail key (in the box a space types); focus the
	// list, then space re-fires the download and the second mock returns the tree.
	m, _ = m.Update(namedKey(tea.KeyEnter)) // box → results list
	m, cmd = spaceKey(m)
	m = runSearchCmd(m, cmd)
	if out := plain(view(m)); !strings.Contains(out, "Comprehensive") {
		t.Errorf("expected the retried download to render SKILL.md, got:\n%s", out)
	}
	if !gock.IsDone() {
		t.Error("expected both the failed and the retried download request to be made")
	}
}

// Slice 12, cycle 4: outside an error state space is inert — it fires no request
// and returns no command.
func TestSkillSearchSpaceInertOutsideError(t *testing.T) {
	defer gock.Off()
	m := searchWithResults(t) // results loaded, no error

	m, _ = m.Update(namedKey(tea.KeyEnter)) // focus the results list

	next, cmd := spaceKey(m)
	if cmd != nil {
		t.Error("expected space to be inert (no retry command) outside an error state")
	}
	if !strings.Contains(plain(view(next)), "vercel-react-best-practices") {
		t.Errorf("expected the results to remain after an inert space, got:\n%s", plain(view(next)))
	}
}

// Slice 13, cycle 1: Esc steps back one focus level per press — the Skill Detail
// returns to the results list, the list to the search box, the box to the entry
// chooser, and the chooser closes. Each rung is proven by a zone-scoped key: an
// Assets tab in the detail, a Name sort in the list, a typed character in the box.
func TestSkillSearchEscapeLadderStepsBackOneLevel(t *testing.T) {
	defer gock.Off()
	m := searchWithResults(t)
	m = seedDownload(t, m, mktDetailFiles())
	m = enterDetail(m) // Enter → list, l → Skill Detail

	// Detail zone: a opens the Assets tab (a detail-only key).
	m, _ = m.Update(runeKey('a'))
	if out := plain(view(m)); !strings.Contains(out, "No preview available") {
		t.Fatalf("expected to begin in the Skill Detail (Assets tab), got:\n%s", out)
	}

	// esc: detail → results list. The list-only Name sort now acts.
	m, _ = m.Update(namedKey(tea.KeyEsc))
	m, _ = m.Update(runeKey('n'))
	if out := plain(view(m)); !strings.Contains(out, "clerk-react-patterns") {
		t.Fatalf("expected esc to step back to the results list (n sorts by Name), got:\n%s", out)
	}

	// esc: results list → search box. Typing now edits the query.
	m, _ = m.Update(namedKey(tea.KeyEsc))
	m, _ = m.Update(runeKey('z'))
	if out := plain(view(m)); !strings.Contains(out, "reactz") {
		t.Fatalf("expected esc to step back to the search box (typing edits the query), got:\n%s", out)
	}

	// esc: search box → entry chooser. The overlay is gone, the chooser shows.
	m, _ = m.Update(namedKey(tea.KeyEsc))
	if out := plain(view(m)); strings.Contains(out, "Search skills…") {
		t.Fatalf("expected esc to leave the overlay for the chooser, got:\n%s", out)
	}
	if !strings.Contains(view(m), "Enter skill URL or repository") {
		t.Fatalf("expected esc to return to the entry chooser, got:\n%s", view(m))
	}

	// esc: entry chooser → closed.
	m, _ = m.Update(namedKey(tea.KeyEsc))
	if strings.Contains(view(m), "Enter skill URL or repository") {
		t.Errorf("expected a final esc to close the chooser, got:\n%s", view(m))
	}
}

// Slice 13, cycle 2: / jumps straight to the search box from the results list or
// the detail; the box takes focus and typing edits the query.
func TestSkillSearchSlashJumpsToBox(t *testing.T) {
	defer gock.Off()

	// From the results list.
	m := searchWithResults(t)
	m, _ = m.Update(namedKey(tea.KeyEnter)) // focus the results list
	m, _ = m.Update(runeKey('/'))           // jump to the search box
	m, _ = m.Update(runeKey('z'))           // typing edits the query
	if out := plain(view(m)); !strings.Contains(out, "reactz") {
		t.Errorf("expected / from the list to focus the box and edit the query, got:\n%s", out)
	}

	// From the Skill Detail.
	m2 := searchWithResults(t)
	m2 = seedDownload(t, m2, mktDetailFiles())
	m2 = enterDetail(m2) // Enter → list, l → Skill Detail
	m2, _ = m2.Update(runeKey('/'))
	m2, _ = m2.Update(runeKey('z'))
	if out := plain(view(m2)); !strings.Contains(out, "reactz") {
		t.Errorf("expected / from the detail to focus the box and edit the query, got:\n%s", out)
	}
}

// Slice 13, cycle 3: backing all the way out of the overlay cancels the
// in-flight request — a late result that arrives after the ladder reaches the
// chooser never renders. A request is left loading, the ladder walks the detail
// back through the list and box to the chooser, then the stale result is
// delivered.
func TestSkillSearchBackingOutDropsInFlightResult(t *testing.T) {
	defer gock.Off()
	gock.New("https://skills.sh").
		Get("/api/search").
		Reply(200).
		File(searchFixture)

	var m tea.Model = withMarket()
	m = settle(openSkillSearch(m, 100, 30))

	// Start a request: the overlay is loading under this epoch. The result cmd is
	// intentionally not run, so the request is still in flight.
	m = typeSearch(m, "react")
	m, _ = fireDebounce(m)
	inFlightEpoch := m.(Model).skillSearch.epoch

	// Move focus deep into the overlay, then walk the escape ladder all the way
	// out: detail → list → box → chooser.
	m, _ = m.Update(namedKey(tea.KeyEnter)) // box → list
	m, _ = m.Update(runeKey('l'))           // list → detail
	m, _ = m.Update(namedKey(tea.KeyEsc))   // detail → list
	m, _ = m.Update(namedKey(tea.KeyEsc))   // list → box
	m, _ = m.Update(namedKey(tea.KeyEsc))   // box → entry chooser

	if m.(Model).skillSearch != nil {
		t.Fatal("expected the full escape ladder to leave the overlay closed")
	}
	if !strings.Contains(view(m), "Enter skill URL or repository") {
		t.Fatalf("expected the entry chooser after backing out, got:\n%s", view(m))
	}

	// The in-flight request's late result arrives after the overlay is gone; it
	// must never render.
	m, _ = m.Update(searchResultsMsg{
		epoch:  inFlightEpoch,
		skills: []marketplace.MarketplaceSkill{{Name: "late-in-flight-skill"}},
	})
	if strings.Contains(plain(view(m)), "late-in-flight-skill") {
		t.Errorf("expected the late in-flight result to never render after backing out, got:\n%s", plain(view(m)))
	}
}

// typedSpace is a realistic space keystroke: its Text is a single space (so a
// textinput inserts it) and its String() is "space" (so it matches the retry
// binding). It distinguishes the box-scoped-typing fix from the retry intercept.
func typedSpace() tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Text: " ", Code: tea.KeySpace})
}

// Finding 1: after a failed search, a space typed in the search box must reach
// the textinput (retry is list/detail only), so it lands in the query instead of
// re-firing an un-debounced search.
func TestSkillSearchSpaceInBoxTypesNotRetries(t *testing.T) {
	defer gock.Off()
	gock.New("https://skills.sh").Get("/api/search").Reply(500)

	var m tea.Model = withMarket()
	m = settle(openSkillSearch(m, 100, 30))
	m = typeSearch(m, "react")
	m, cmd := fireDebounce(m)
	m = runSearchCmd(m, cmd)

	if m.(Model).skillSearch.state != searchError {
		t.Fatalf("precondition: expected the search to have failed (searchError)")
	}

	// A space in the box types into the query; it must not be hijacked as retry.
	m, _ = m.Update(typedSpace())

	if got := m.(Model).skillSearch.box.Value(); !strings.Contains(got, "react ") {
		t.Errorf("expected the space to type into the query, got %q", got)
	}
	if m.(Model).skillSearch.state == searchLoading {
		t.Error("expected typing a space in the box to schedule a debounce, not fire an un-debounced retry")
	}
}

// Finding 1: fireSearch must cancel any in-flight request's context before
// overwriting o.ctxCancel, so a superseding fire can never leak the prior cancel
// func or leave a request uncancellable.
func TestSkillSearchFireSearchCancelsPriorRequest(t *testing.T) {
	defer gock.Off()
	gock.New("https://skills.sh").Get("/api/search").Reply(200).File(searchFixture)

	var m tea.Model = withMarket()
	m = settle(openSkillSearch(m, 100, 30))
	m = typeSearch(m, "react")

	// Stand in for an in-flight request's cancel func left on the overlay.
	cancelled := false
	m.(Model).skillSearch.ctxCancel = func() { cancelled = true }

	// The debounce fires a search under the current epoch; before it overwrites
	// the in-flight cancel func it must cancel the prior context. (The overlay is a
	// shared pointer, so the fire's effect is visible without keeping the result.)
	fireDebounce(m)
	if !cancelled {
		t.Error("expected fireSearch to cancel the prior in-flight context before starting a new request")
	}
}

// Finding 2: a downloadResultsMsg arriving while the post-install chooser is open
// must be forwarded to the overlay (not dropped), so the detail is not stranded on
// a frozen spinner after Find more skills.
func TestSkillSearchPostInstallForwardsDownloadResult(t *testing.T) {
	defer gock.Off()
	m := searchWithResults(t, WithAddRunner(nopRunner()))

	// A download is in flight for the top result (dwell fired, result not yet in).
	m, _ = fireDwell(m)
	o := m.(Model).skillSearch
	dlEpoch := o.dlEpoch
	ref := o.results[o.selected].InstallRef()
	if !o.dlLoading {
		t.Fatalf("precondition: expected a download in flight after the dwell")
	}

	// Install → the post-install chooser opens over the preserved overlay.
	m = installFromList(m)
	if !strings.Contains(plain(view(m)), "Find more skills") {
		t.Fatalf("precondition: expected the post-install chooser")
	}

	// The in-flight download's result arrives while the chooser is open. Its epoch
	// is still current, so it must not be dropped.
	m, _ = m.Update(downloadResultsMsg{dlEpoch: dlEpoch, ref: ref, files: mktDetailFiles()})

	// Find more skills returns to the box; the detail renders the downloaded
	// SKILL.md instead of a frozen spinner.
	m, _ = m.Update(namedKey(tea.KeyEnter)) // Find more (default cursor)
	out := plain(view(m))
	if !strings.Contains(out, "SkillBodyMarker") {
		t.Errorf("expected the downloaded SKILL.md to render after Find more, got:\n%s", out)
	}
	if strings.Contains(out, spinnerFrames[0]) {
		t.Errorf("expected no frozen download spinner after the result arrived, got:\n%s", out)
	}
}

// Finding 2: a WindowSizeMsg while the post-install chooser is open must update
// the model's width/height (not be dropped).
func TestSkillSearchPostInstallHandlesResize(t *testing.T) {
	defer gock.Off()
	m := searchWithResults(t, WithAddRunner(nopRunner()))
	m = installFromList(m)

	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 44})

	if got := m.(Model); got.width != 140 || got.height != 44 {
		t.Errorf("expected the resize to update width/height, got %d x %d", got.width, got.height)
	}
}

// Finding 3: on the SKILL.md tab (which has no file list), j scrolls the content
// viewport directly regardless of subfocus — without first pressing tab.
func TestSkillSearchDetailSkillMDScrollsWithoutTab(t *testing.T) {
	defer gock.Off()
	m := searchWithResults(t)

	tall := "---\nname: react\n---\n\n# TopMarker\n\n" +
		strings.Repeat("Filler paragraph.\n\n", 60) + "SkillBottomMarker\n"
	m = seedDownload(t, m, marketplace.SkillFiles{Files: []marketplace.File{
		{Path: "SKILL.md", Contents: tall},
	}})
	m = enterDetail(m) // Enter → list, l → detail; default tab SKILL.md, list subfocus

	if out := plain(view(m)); strings.Contains(out, "SkillBottomMarker") {
		t.Fatalf("precondition: expected the bottom of the tall SKILL.md below the fold, got:\n%s", out)
	}

	// j on the SKILL.md tab scrolls the content directly (no tab press first).
	for i := 0; i < 200; i++ {
		m, _ = m.Update(runeKey('j'))
	}
	if out := plain(view(m)); !strings.Contains(out, "SkillBottomMarker") {
		t.Errorf("expected j on SKILL.md to scroll the content viewport, got:\n%s", out)
	}
}

// Finding 4: a freshly opened overlay (searchIdle, empty query) shows the same
// under-two-characters hint as searchTooShort, not a blank results pane.
func TestSkillSearchFreshOverlayShowsHint(t *testing.T) {
	var m tea.Model = withMarket()
	m = settle(openSkillSearch(m, 100, 30))

	out := plain(view(m))
	if !strings.Contains(out, "Type at least 2 characters") {
		t.Errorf("expected the fresh overlay to show the type-more hint, got:\n%s", out)
	}
}

// Finding 5: Finish must cancel any in-flight download before clearing the
// overlay, so no request outlives the Skill Search session.
func TestSkillSearchFinishCancelsInFlightDownload(t *testing.T) {
	defer gock.Off()
	m := searchWithResults(t, WithAddRunner(nopRunner()))
	m = installFromList(m) // opens the post-install chooser

	// Stand in for an in-flight download's cancel func on the preserved overlay.
	cancelled := false
	o := m.(Model).skillSearch
	o.dlCancel = func() { cancelled = true }
	o.dlLoading = true

	// Finish closes the overlay; it must cancel the in-flight download first.
	m, _ = m.Update(runeKey('j'))    // move onto Finish
	m.Update(namedKey(tea.KeyEnter)) // Finish

	if !cancelled {
		t.Error("expected Finish to cancel the in-flight download before closing the overlay")
	}
}

// Finding 6: a failed install must not claim success — the post-install chooser
// is titled honestly and carries no "Skill installed" claim.
func TestSkillSearchInstallFailureShowsFailureChooser(t *testing.T) {
	defer gock.Off()
	failRunner := func(cmd *exec.Cmd, done func(error) tea.Msg) tea.Cmd {
		return func() tea.Msg { return done(errors.New("npx exploded")) }
	}
	m := searchWithResults(t, WithAddRunner(failRunner))
	m = installFromList(m) // installs; completion carries the error

	out := plain(view(m))
	if strings.Contains(out, "Skill installed") {
		t.Errorf("expected no false success claim on a failed install, got:\n%s", out)
	}
	if !strings.Contains(out, "Install failed") {
		t.Errorf("expected a failure-titled chooser after a failed install, got:\n%s", out)
	}
}

// Finding 6: a failed install must not rescan or land on the (uninstalled) skill.
func TestSkillSearchInstallFailureDoesNotRescan(t *testing.T) {
	defer gock.Off()
	rescanned := false
	rescan := func() []skills.ScanResult {
		rescanned = true
		return []skills.ScanResult{browseResult()}
	}
	failRunner := func(cmd *exec.Cmd, done func(error) tea.Msg) tea.Cmd {
		return func() tea.Msg { return done(errors.New("npx exploded")) }
	}
	m := searchWithResults(t, WithAddRunner(failRunner), WithRescan(rescan))
	m = installFromList(m) // failed install → failure chooser

	// Back out of the failure chooser: no rescan runs on the failure path.
	m.Update(namedKey(tea.KeyEsc))
	if rescanned {
		t.Error("expected no rescan on the failed-install path")
	}
}

// Cycle 4: Esc from the search box clears the overlay and returns to the entry
// chooser.
func TestSkillSearchEscReturnsToChooser(t *testing.T) {
	var m tea.Model = withMarket()

	m = settle(openSkillSearch(m, 100, 30))

	m, _ = m.Update(namedKey(tea.KeyEsc))

	out := view(m)
	if strings.Contains(plain(out), "Search skills…") {
		t.Errorf("expected esc to clear the Skill Search overlay, got:\n%s", plain(out))
	}
	if !strings.Contains(out, "Enter skill URL or repository") {
		t.Errorf("expected esc to return to the entry chooser, got:\n%s", out)
	}
}
