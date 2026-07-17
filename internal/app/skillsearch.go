package app

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/harmonica"

	"github.com/makesometh-ing/trainer/internal/actions"
	"github.com/makesometh-ing/trainer/internal/marketplace"
	"github.com/makesometh-ing/trainer/internal/render"
	"github.com/makesometh-ing/trainer/internal/skills"
)

// Grow-animation tuning. The spring runs at a fixed frame rate; freq and damping
// give an underdamped settle (a small overshoot then rest), per the design.
const (
	springFPS  = 60
	springFreq = 6.0
	springDamp = 0.7

	// settleEps / settleVel are how close the spring must be to its target, in
	// cells and cells-per-frame, before the loop snaps to the target and stops.
	settleEps = 0.5
	settleVel = 0.5
)

// animFrameMsg is one tick of the grow animation, delivered through Update. It
// is dropped when no overlay is open (see Model.Update).
type animFrameMsg struct{}

// Skill Search query lifecycle tuning.
const (
	// searchDebounce is how long after the last keystroke a search fires.
	searchDebounce = 300 * time.Millisecond
	// searchLimit is the number of Marketplace Skills requested per search.
	searchLimit = 25
	// minSearchLen is the shortest query that produces a request; shorter
	// queries show a hint instead (the API rejects them anyway).
	minSearchLen = 2
	// spinnerFPS is how fast the loading spinner advances while a request runs.
	spinnerFPS = 10
	// dwellDelay is how long the selection must rest on a Marketplace Skill before
	// its file tree is downloaded, so quick scrolling does not fire a request per row.
	dwellDelay = 200 * time.Millisecond
)

// spinnerFrames are the braille frames of the loading spinner shown while a
// Skill Search request is in flight.
var spinnerFrames = []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}

// searchZone is which part of the settled overlay has focus. Keys are
// zone-scoped: the box owns typed letters, the list owns navigation and sorts.
type searchZone int

const (
	// zoneBox is the search input; typing edits the query.
	zoneBox searchZone = iota
	// zoneList is the results list; j/k navigate and r/p/n sort.
	zoneList
	// zoneDetail is the Skill Detail; i/r/s/a switch tabs, tab toggles subfocus,
	// j/k move the file selection or scroll the content.
	zoneDetail
)

// searchState is the phase of the Skill Search query lifecycle the results pane
// reflects.
type searchState int

const (
	// searchIdle is the initial phase: box focused, nothing typed yet.
	searchIdle searchState = iota
	// searchTooShort means the query is under minSearchLen; a hint shows and no
	// request is made.
	searchTooShort
	// searchLoading means a request is in flight; the spinner shows.
	searchLoading
	// searchOK means results have landed and render as a list.
	searchOK
	// searchEmpty means the request succeeded but matched no Marketplace Skills;
	// the results pane names the query.
	searchEmpty
	// searchError means the request failed; the results pane shows the retry hint.
	searchError
)

// searchDebounceMsg fires searchDebounce after the last keystroke. It carries
// the epoch it was scheduled under; Update drops it if a newer keystroke has
// advanced the epoch, so only the final keystroke of a burst starts a request.
type searchDebounceMsg struct{ epoch int }

// searchResultsMsg carries the outcome of one Search request back to Update. It
// carries the epoch the request was issued under; a stale result (superseded by
// a newer keystroke, or a cancelled in-flight request) is dropped.
type searchResultsMsg struct {
	epoch  int
	skills []marketplace.MarketplaceSkill
	err    error
}

// spinnerTickMsg advances the loading spinner. It carries the epoch it was
// scheduled under and stops as soon as the request is no longer loading.
type spinnerTickMsg struct{ epoch int }

// dwellMsg fires dwellDelay after the selection last moved. It carries the dwell
// epoch it was scheduled under; Update drops it if the selection has moved again,
// so only a resting selection triggers a download.
type dwellMsg struct{ dlEpoch int }

// downloadResultsMsg carries the outcome of one Download request back to Update.
// It carries the dwell epoch the request was issued under (a stale result, from a
// selection that has since moved, is dropped) and the cache key ref it was for.
type downloadResultsMsg struct {
	dlEpoch int
	ref     string
	files   marketplace.SkillFiles
	err     error
}

// dlSpinnerTickMsg advances the download spinner shown in the detail pane. It
// carries the dwell epoch it was scheduled under and stops once the download is
// no longer in flight.
type dlSpinnerTickMsg struct{ dlEpoch int }

// searchOverlay is the Skill Search overlay: a shell that grows from the
// chooser's size to near-full-window on a harmonica spring, then settles and
// focuses the search box. Later slices add results, detail, and query state; for
// now it is the animated shell plus the (settled) input box.
type searchOverlay struct {
	box    textinput.Model
	spring harmonica.Spring

	// w/h are the current animated dimensions; wVel/hVel their velocities;
	// targetW/targetH the size the spring is aiming for. growing is true while the
	// tween runs and false once settled.
	w, h             float64
	wVel, hVel       float64
	targetW, targetH float64
	growing          bool

	// Query lifecycle. epoch increments on every keystroke so stale debounce,
	// result, and spinner messages can be dropped. ctxCancel cancels the
	// in-flight request when a newer keystroke supersedes it. state drives what
	// the results pane renders; results holds the landed page; spinnerFrame is
	// the current loading-spinner frame.
	epoch        int
	ctxCancel    context.CancelFunc
	state        searchState
	results      []marketplace.MarketplaceSkill
	spinnerFrame int

	// zone is the focused part of the settled overlay (box or list).
	zone searchZone
	// selected is the highlighted row in the results list.
	selected int

	// apiOrder is the page in the Marketplace's own order, never mutated, so
	// Relevance is always re-derivable. results is apiOrder sorted by sortKey and
	// sortDir for display.
	apiOrder []marketplace.MarketplaceSkill
	sortKey  marketplace.SortField
	sortDir  marketplace.SortDir

	// Download lifecycle for the detail pane. dlEpoch increments whenever the
	// selected skill changes, so stale dwell/download/spinner messages are dropped
	// and only one download is ever in flight. dlCancel cancels that in-flight
	// download when the selection moves. dlLoading drives the detail spinner;
	// dlError shows the detail retry hint after a failed download (space re-fires
	// it). dlSpinnerFrame is its current frame. files is the session download cache,
	// keyed by InstallRef (source@skillId), so a re-selected skill renders with no
	// second request; it is discarded when the overlay closes.
	dlEpoch        int
	dlCancel       context.CancelFunc
	dlLoading      bool
	dlError        bool
	dlSpinnerFrame int
	files          map[string]marketplace.SkillFiles

	// Skill Detail state (zoneDetail). detailTab is the open tab
	// (SKILL.md/References/Scripts/Assets); detailFileSel the selected file in a
	// file tab; detailSubfocus whether the file list or the content has focus.
	// detailVP scrolls the content. rendered memoizes each file's rendered output
	// keyed by ref|tab|fileIdx|width, so re-opening a file never re-renders.
	detailTab      tab
	detailFileSel  int
	detailSubfocus subfocus
	detailVP       viewport.Model
	rendered       map[string]string

	// installedName is the Name of the last Marketplace Skill installed through
	// the add seam, so the post-install Finish path can land the browser
	// selection on it after the rescan.
	installedName string
}

// selectedRef is the InstallRef of the currently selected Marketplace Skill, or
// "" when there are no results.
func (o *searchOverlay) selectedRef() string {
	if len(o.results) == 0 {
		return ""
	}
	return o.results[o.selected].InstallRef()
}

// applySort re-derives the display order from the untouched API page and clamps
// the selection into range.
func (o *searchOverlay) applySort() {
	o.results = marketplace.SortSkills(o.apiOrder, o.sortKey, o.sortDir)
	if o.selected > len(o.results)-1 {
		o.selected = len(o.results) - 1
	}
	if o.selected < 0 {
		o.selected = 0
	}
}

// newSkillSearchInput builds the overlay's search box.
func newSkillSearchInput() textinput.Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "Search skills…"
	return ti
}

// newSearchOverlay builds the overlay starting at the chooser's size and aiming
// for the near-full-window target.
func newSearchOverlay(startW, startH, targetW, targetH int) *searchOverlay {
	return &searchOverlay{
		box:      newSkillSearchInput(),
		spring:   harmonica.NewSpring(harmonica.FPS(springFPS), springFreq, springDamp),
		w:        float64(startW),
		h:        float64(startH),
		targetW:  float64(targetW),
		targetH:  float64(targetH),
		growing:  true,
		zone:     zoneBox,
		detailVP: viewport.New(viewport.WithWidth(1), viewport.WithHeight(1)),
	}
}

// searchTargetW is the near-full-window width the overlay grows to.
func (m Model) searchTargetW() int {
	w := m.width - 2
	if w < minWidth {
		w = minWidth
	}
	return w
}

// searchTargetH is the near-full-window height the overlay grows to. It fills the
// frame down to (but never into) the footer row: the overlay outer height plus the
// footer row must not exceed the terminal height, or the View overflows. It never
// clamps up past the available height, since minHeight is the smallest supported
// terminal and forcing the overlay taller than the terminal is what overflowed.
func (m Model) searchTargetH() int {
	h := m.height - frameMargin
	if h < paneVerticalChrome+1 {
		h = paneVerticalChrome + 1
	}
	return h
}

// animTick schedules the next animation frame.
func animTick() tea.Cmd {
	return tea.Tick(time.Second/springFPS, func(time.Time) tea.Msg {
		return animFrameMsg{}
	})
}

// updateSkillSearch handles messages while the Skill Search overlay is open. A
// window resize re-aims the spring; a frame advances the grow; esc steps back to
// the entry chooser; other keys go to the box.
func (m Model) updateSkillSearch(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.syncSize()
		o := m.skillSearch
		o.targetW = float64(m.searchTargetW())
		o.targetH = float64(m.searchTargetH())
		if !o.growing {
			// The grow tween has already settled, so nothing is driving the spring.
			// A resize re-aims the target but the ticker is stopped; snap straight to
			// the new size (velocities zeroed) so the overlay tracks the terminal
			// instead of staying frozen at the old size. Mid-grow (growing) the
			// running animTick loop picks up the re-aimed target on its own.
			o.w, o.h = o.targetW, o.targetH
			o.wVel, o.hVel = 0, 0
		}
		return m, nil
	case animFrameMsg:
		return m.advanceSkillSearchAnim()
	case searchDebounceMsg:
		return m.fireSearch(msg.epoch)
	case searchResultsMsg:
		return m.applySearchResults(msg)
	case spinnerTickMsg:
		return m.advanceSpinner(msg.epoch)
	case dwellMsg:
		return m.fireDownload(msg.dlEpoch)
	case downloadResultsMsg:
		return m.applyDownloadResults(msg)
	case dlSpinnerTickMsg:
		return m.advanceDlSpinner(msg.dlEpoch)
	case tea.KeyPressMsg:
		if msg.String() == "esc" {
			return m.escapeSkillSearch()
		}
		// / jumps straight to the search box from the results list or the detail;
		// in the box itself / is a query character, so it is scoped out of zoneBox.
		if key.Matches(msg, m.keys.search) && m.skillSearch.zone != zoneBox {
			m.skillSearch.zone = zoneBox
			m.skillSearch.box.Focus()
			return m, nil
		}
		// space re-fires a failed request from the results list or the detail;
		// outside an error state it is inert there. In the box a space is always a
		// query character (retry is list/detail only, per the control scheme), so it
		// is scoped out of zoneBox and reaches the textinput below.
		if key.Matches(msg, m.keys.mktRetry) && m.skillSearch.zone != zoneBox {
			if next, cmd, ok := m.retryFailed(); ok {
				return next, cmd
			}
		}
		// 1/2/3 focus the panes like the main browser, but only outside the search
		// box: in the box the digits are query characters, so this is scoped out of
		// zoneBox and the digits reach the textinput below.
		if key.Matches(msg, m.keys.mktFocusPanes) && m.skillSearch.zone != zoneBox {
			switch msg.String() {
			case "1":
				m.skillSearch.zone = zoneBox
				m.skillSearch.box.Focus()
			case "2":
				m.skillSearch.zone = zoneList
			case "3":
				m.skillSearch.zone = zoneDetail
			}
			return m, nil
		}
		if m.skillSearch.zone == zoneList {
			return m.updateSearchList(msg)
		}
		if m.skillSearch.zone == zoneDetail {
			return m.updateSearchDetail(msg)
		}
		// Box focus: Enter or ↓ hands focus to the results list; every other key
		// edits the query.
		if s := msg.String(); s == "enter" || s == "down" {
			m.skillSearch.zone = zoneList
			return m, nil
		}
		var cmd tea.Cmd
		m.skillSearch.box, cmd = m.skillSearch.box.Update(msg)
		_ = cmd // the box's blink cmd is dropped; the query lifecycle drives ticks
		return m.afterQueryChange()
	}
	return m, nil
}

// escapeSkillSearch closes the overlay from any zone: a single esc cancels any
// in-flight search and download and returns to the add entry chooser. There is
// no step-back ladder — box, results list, and Skill Detail all close in one
// press (1/2/3 and h/l move focus between the panes instead).
func (m Model) escapeSkillSearch() (tea.Model, tea.Cmd) {
	m.cancelSearch()
	m.cancelDownload()
	m.skillSearch = nil
	m.chooser = &addChooser{kind: chooserEntry, cursor: entrySearch}
	return m, nil
}

// updateSearchList handles a key press while the results list has focus. j/k
// move the selection; sort keys reorder the page.
func (m Model) updateSearchList(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	o := m.skillSearch
	before := o.selectedRef()
	switch {
	case key.Matches(msg, m.keys.mktInstall):
		return m.runAddFromSearch()
	case key.Matches(msg, m.keys.move):
		switch msg.String() {
		case "j", "down":
			if o.selected < len(o.results)-1 {
				o.selected++
			}
		case "k", "up":
			if o.selected > 0 {
				o.selected--
			}
		}
	case key.Matches(msg, m.keys.mktSort):
		switch msg.String() {
		case "r":
			o.setSort(marketplace.SortRelevance)
		case "p":
			o.setSort(marketplace.SortInstalls)
		case "n":
			o.setSort(marketplace.SortName)
		}
	case key.Matches(msg, m.keys.mktToDetail):
		// l opens the Skill Detail for the resting selection. h is deliberately inert
		// in the Results list — the search box is reached only with 1 or /.
		o.zone = zoneDetail
		return m, nil
	}
	// A move or a sort that lands on a different skill rests the selection anew:
	// cancel any in-flight download, reset to SKILL.md, and start a fresh dwell.
	if o.selectedRef() != before {
		return m.restSelection()
	}
	return m, nil
}

// runAddFromSearch installs the currently selected Marketplace Skill through the
// existing add seam: `npx skills@latest add <source>@<skillId>` via the injected
// AddRunner, with no SSH key. Unlike the manual wizard's runAdd it leaves the
// overlay open, so the post-install chooser (slice 11) can take over once the
// install command exits.
func (m Model) runAddFromSearch() (tea.Model, tea.Cmd) {
	o := m.skillSearch
	if len(o.results) == 0 || m.addRunner == nil {
		return m, nil
	}
	sel := o.results[o.selected]
	// Remember the installed skill's name so the post-install Finish can land the
	// browser selection on it once the rescan lists it.
	o.installedName = sel.Name
	cmd := actions.AddCommand(sel.InstallRef(), "")
	// Carry the install error through so the post-install chooser can tell success
	// from failure rather than always claiming the skill was installed.
	run := m.addRunner(cmd, func(err error) tea.Msg { return addFinishedMsg{err: err} })
	return m, run
}

// retryInstall re-attempts the install of the still-selected Marketplace Skill
// after a failure: it closes the failure chooser and fires the add command again
// (the results and selection are preserved underneath).
func (m Model) retryInstall() (tea.Model, tea.Cmd) {
	m.chooser = nil
	return m.runAddFromSearch()
}

// findMoreSkills returns from the post-install chooser to the Skill Search box
// with the query, results, and download cache preserved. No rescan runs and no
// request is re-issued.
func (m Model) findMoreSkills() (tea.Model, tea.Cmd) {
	m.chooser = nil
	if m.skillSearch != nil {
		m.skillSearch.zone = zoneBox
	}
	return m, nil
}

// finishFromSearch closes the post-install chooser and the Skill Search overlay,
// rescans the disk once, and lands the browser selection on the newly installed
// skill when it is findable. The session download cache is discarded with the
// overlay.
func (m Model) finishFromSearch() (tea.Model, tea.Cmd) {
	name := ""
	if m.skillSearch != nil {
		name = m.skillSearch.installedName
		// Cancel any in-flight search or download before discarding the overlay, so
		// no request outlives the Skill Search session (mirrors escapeSkillSearch's
		// box→chooser rung).
		m.cancelSearch()
		m.cancelDownload()
	}
	m.chooser = nil
	m.skillSearch = nil
	m = m.refreshFromDisk()
	m.landOnSkill(name)
	return m, nil
}

// landOnSkill best-effort moves the browser selection onto the skill with the
// given name, searching every scope. It clears the search and filter first so a
// freshly installed skill is visible; if no scope holds the name the selection
// is left where refreshFromDisk clamped it.
func (m *Model) landOnSkill(name string) {
	if name == "" {
		return
	}
	m.resetSearchFilter()
	for si, r := range m.results {
		for i, s := range r.Skills {
			if s.Name == name {
				m.selectedScope = si
				m.selected = i
				m.syncContent()
				return
			}
		}
	}
}

// setSort switches the results ordering. Pressing the current sort key again
// toggles its direction; switching to a different key resets to that key's
// natural direction (Popularity descending, Relevance and Name ascending).
func (o *searchOverlay) setSort(field marketplace.SortField) {
	if o.sortKey == field {
		if o.sortDir == marketplace.Asc {
			o.sortDir = marketplace.Desc
		} else {
			o.sortDir = marketplace.Asc
		}
	} else {
		o.sortKey = field
		o.sortDir = naturalSortDir(field)
	}
	o.applySort()
}

// naturalSortDir is the default direction for a sort field: Popularity leads
// with the most-installed, the others read ascending.
func naturalSortDir(field marketplace.SortField) marketplace.SortDir {
	if field == marketplace.SortInstalls {
		return marketplace.Desc
	}
	return marketplace.Asc
}

// updateSearchDetail handles a key press while the Skill Detail has focus. h
// returns to the results list; i/r/s/a switch tabs; tab toggles the file-list /
// content subfocus; j/k move the file selection or scroll the content.
func (m Model) updateSearchDetail(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	o := m.skillSearch
	switch {
	case key.Matches(msg, m.keys.mktInstall):
		return m.runAddFromSearch()
	case key.Matches(msg, m.keys.mktToList):
		o.zone = zoneList
	case key.Matches(msg, m.keys.tabs):
		m.setMktTab(msg.String())
	case key.Matches(msg, m.keys.subfocus):
		if o.detailSubfocus == subfocusList {
			o.detailSubfocus = subfocusContent
		} else {
			o.detailSubfocus = subfocusList
		}
	case key.Matches(msg, m.keys.detailMove):
		m.moveMktDetail(msg.String())
	}
	return m, nil
}

// setMktTab switches the open Skill Detail tab and resets the file selection and
// subfocus, so a tab always opens on its first file with the file list focused.
func (m Model) setMktTab(k string) {
	o := m.skillSearch
	switch k {
	case "i":
		o.detailTab = tabSkill
	case "r":
		o.detailTab = tabReferences
	case "s":
		o.detailTab = tabScripts
	case "a":
		o.detailTab = tabAssets
	}
	o.detailFileSel = 0
	o.detailSubfocus = subfocusList
	o.detailVP.GotoTop()
}

// moveMktDetail moves the file selection (list subfocus) or scrolls the content
// (content subfocus) by one line in the given direction.
func (m Model) moveMktDetail(k string) {
	o := m.skillSearch
	delta := 1
	if k == "k" || k == "up" {
		delta = -1
	}
	// The SKILL.md tab has no file list (mirrors moveContent in update.go): j/k
	// scroll the content viewport directly, regardless of subfocus.
	if o.detailTab != tabSkill && o.detailSubfocus == subfocusList {
		m.moveMktFileSel(delta)
		return
	}
	m.syncMktDetail()
	if delta > 0 {
		o.detailVP.ScrollDown(1)
	} else {
		o.detailVP.ScrollUp(1)
	}
}

// moveMktFileSel moves the file selection within the current tab, clamped to the
// file list, and scrolls the newly selected file's content back to the top.
func (m Model) moveMktFileSel(delta int) {
	o := m.skillSearch
	files := m.mktCurrentFiles()
	if len(files) == 0 {
		return
	}
	next := o.detailFileSel + delta
	if next < 0 {
		next = 0
	}
	if next > len(files)-1 {
		next = len(files) - 1
	}
	if next != o.detailFileSel {
		o.detailFileSel = next
		m.syncMktDetail()
		o.detailVP.GotoTop()
	}
}

// afterQueryChange runs after every keystroke edits the search box. It bumps the
// epoch (so any in-flight debounce/request/spinner becomes stale) and cancels
// the in-flight request. A query under minSearchLen shows the hint with no
// request; a longer one schedules a debounced search.
func (m Model) afterQueryChange() (tea.Model, tea.Cmd) {
	o := m.skillSearch
	o.epoch++
	m.cancelSearch()

	if len(strings.TrimSpace(o.box.Value())) < minSearchLen {
		o.state = searchTooShort
		return m, nil
	}
	return m, debounceTick(o.epoch)
}

// cancelSearch aborts the in-flight request, if any, so its result is discarded
// (it arrives under a stale epoch and is dropped).
func (m Model) cancelSearch() {
	o := m.skillSearch
	if o.ctxCancel != nil {
		o.ctxCancel()
		o.ctxCancel = nil
	}
}

// fireSearch starts a request for the current query when the debounce that
// scheduled it is still current. A stale debounce (superseded by a newer
// keystroke) is dropped. It moves to the loading state and returns the request
// plus the spinner tick.
func (m Model) fireSearch(epoch int) (tea.Model, tea.Cmd) {
	o := m.skillSearch
	if epoch != o.epoch {
		return m, nil
	}
	query := strings.TrimSpace(o.box.Value())

	// Cancel any request still in flight before overwriting its cancel func, so a
	// superseding fire can never leak the prior cancel (leaving a request
	// uncancellable) or race a second request under the same epoch.
	if o.ctxCancel != nil {
		o.ctxCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	o.ctxCancel = cancel
	o.state = searchLoading
	o.spinnerFrame = 0
	return m, tea.Batch(searchCmd(m.market, ctx, query, epoch), spinnerTick(epoch))
}

// applySearchResults records a landed page when it belongs to the current
// query. A stale result (older epoch) is dropped so it never renders.
func (m Model) applySearchResults(msg searchResultsMsg) (tea.Model, tea.Cmd) {
	o := m.skillSearch
	if msg.epoch != o.epoch {
		return m, nil
	}
	o.ctxCancel = nil
	if msg.err != nil {
		// A failed request stops the spinner and shows the retry hint; space
		// re-fires it (see retryFailed).
		o.state = searchError
		return m, nil
	}
	// A landed page resets to the default order (Popularity descending) with the
	// selection at the top.
	o.apiOrder = msg.skills
	o.sortKey = marketplace.SortInstalls
	o.sortDir = marketplace.Desc
	o.selected = 0
	o.applySort()
	if len(o.results) == 0 {
		// A successful request that matched nothing: name the query, no dwell.
		o.state = searchEmpty
		return m, nil
	}
	o.state = searchOK
	// The selection now rests on the top result; start its dwell so the detail
	// pane downloads and shows SKILL.md.
	return m.restSelection()
}

// advanceSpinner steps the loading spinner one frame while the request it
// belongs to is still loading; otherwise it stops (returns no tick).
func (m Model) advanceSpinner(epoch int) (tea.Model, tea.Cmd) {
	o := m.skillSearch
	if epoch != o.epoch || o.state != searchLoading {
		return m, nil
	}
	o.spinnerFrame++
	return m, spinnerTick(epoch)
}

// debounceTick schedules the debounce message after the pause.
func debounceTick(epoch int) tea.Cmd {
	return tea.Tick(searchDebounce, func(time.Time) tea.Msg {
		return searchDebounceMsg{epoch: epoch}
	})
}

// spinnerTick schedules the next spinner frame.
func spinnerTick(epoch int) tea.Cmd {
	return tea.Tick(time.Second/spinnerFPS, func(time.Time) tea.Msg {
		return spinnerTickMsg{epoch: epoch}
	})
}

// searchCmd runs one Skill Search request and reports its outcome, tagged with
// the epoch it was issued under.
func searchCmd(client *marketplace.Client, ctx context.Context, query string, epoch int) tea.Cmd {
	return func() tea.Msg {
		skills, err := client.Search(ctx, query, searchLimit)
		return searchResultsMsg{epoch: epoch, skills: skills, err: err}
	}
}

// restSelection is called whenever the selected Marketplace Skill changes. It
// cancels any in-flight download, advances the dwell epoch (so stale download
// messages are dropped), and schedules the dwell that will download the newly
// selected skill after it has rested.
func (m Model) restSelection() (tea.Model, tea.Cmd) {
	o := m.skillSearch
	m.cancelDownload()
	o.dlLoading = false
	o.dlError = false
	o.dlEpoch++
	// A new selection resets the Skill Detail to the SKILL.md tab with the file
	// list focused and scrolled to the top, so the detail always opens on the
	// fast first view.
	o.detailTab = tabSkill
	o.detailFileSel = 0
	o.detailSubfocus = subfocusList
	o.detailVP.GotoTop()
	return m, dwellTick(o.dlEpoch)
}

// cancelDownload aborts the in-flight download, if any, so its result is
// discarded (it arrives under a stale dwell epoch and is dropped).
func (m Model) cancelDownload() {
	o := m.skillSearch
	if o.dlCancel != nil {
		o.dlCancel()
		o.dlCancel = nil
	}
}

// fireDownload downloads the resting selection's file tree when the dwell that
// scheduled it is still current. A stale dwell (the selection has since moved) is
// dropped, and an already-downloaded skill is served from the session cache with
// no request. It moves to the loading state and returns the request plus the
// detail spinner tick.
func (m Model) fireDownload(dlEpoch int) (tea.Model, tea.Cmd) {
	o := m.skillSearch
	if dlEpoch != o.dlEpoch || len(o.results) == 0 {
		return m, nil
	}
	sel := o.results[o.selected]
	if _, ok := o.files[sel.InstallRef()]; ok {
		// Already in the session cache: it renders from memory, no second request.
		return m, nil
	}
	owner, repo := sel.OwnerRepo()

	ctx, cancel := context.WithCancel(context.Background())
	o.dlCancel = cancel
	o.dlLoading = true
	o.dlSpinnerFrame = 0
	return m, tea.Batch(downloadCmd(m.market, ctx, owner, repo, sel.SkillId, sel.InstallRef(), dlEpoch), dlSpinnerTick(dlEpoch))
}

// applyDownloadResults caches a landed file tree when it belongs to the current
// selection. A stale result (the selection has since moved) is dropped so the
// wrong skill's content never renders.
func (m Model) applyDownloadResults(msg downloadResultsMsg) (tea.Model, tea.Cmd) {
	o := m.skillSearch
	if msg.dlEpoch != o.dlEpoch {
		return m, nil
	}
	o.dlCancel = nil
	o.dlLoading = false
	if msg.err != nil {
		// A failed download shows the retry hint in the detail pane; space
		// re-fires it (see retryFailed).
		o.dlError = true
		return m, nil
	}
	o.dlError = false
	if o.files == nil {
		o.files = make(map[string]marketplace.SkillFiles)
	}
	o.files[msg.ref] = msg.files
	return m, nil
}

// retryFailed re-fires whichever request is currently in an error state: a
// failed search retries the current query, a failed download retries the current
// selection. It reports ok=false when nothing is in an error state, so the caller
// can let space fall through to its normal handling.
func (m Model) retryFailed() (tea.Model, tea.Cmd, bool) {
	o := m.skillSearch
	switch {
	case o.state == searchError:
		next, cmd := m.fireSearch(o.epoch)
		return next, cmd, true
	case o.dlError:
		o.dlError = false
		o.dlEpoch++
		next, cmd := m.fireDownload(o.dlEpoch)
		return next, cmd, true
	default:
		return m, nil, false
	}
}

// advanceDlSpinner steps the detail spinner one frame while the download it
// belongs to is still in flight; otherwise it stops (returns no tick).
func (m Model) advanceDlSpinner(dlEpoch int) (tea.Model, tea.Cmd) {
	o := m.skillSearch
	if dlEpoch != o.dlEpoch || !o.dlLoading {
		return m, nil
	}
	o.dlSpinnerFrame++
	return m, dlSpinnerTick(dlEpoch)
}

// dwellTick schedules the dwell message after the selection has rested.
func dwellTick(dlEpoch int) tea.Cmd {
	return tea.Tick(dwellDelay, func(time.Time) tea.Msg {
		return dwellMsg{dlEpoch: dlEpoch}
	})
}

// dlSpinnerTick schedules the next detail-spinner frame.
func dlSpinnerTick(dlEpoch int) tea.Cmd {
	return tea.Tick(time.Second/spinnerFPS, func(time.Time) tea.Msg {
		return dlSpinnerTickMsg{dlEpoch: dlEpoch}
	})
}

// downloadCmd runs one Download request and reports its outcome, tagged with the
// dwell epoch it was issued under and the cache key ref it was for.
func downloadCmd(client *marketplace.Client, ctx context.Context, owner, repo, skillId, ref string, dlEpoch int) tea.Cmd {
	return func() tea.Msg {
		files, err := client.Download(ctx, owner, repo, skillId)
		return downloadResultsMsg{dlEpoch: dlEpoch, ref: ref, files: files, err: err}
	}
}

// advanceSkillSearchAnim steps the spring one frame. While growing it returns
// the next tick; when the spring reaches its target it snaps to the target,
// focuses the box, and stops ticking.
func (m Model) advanceSkillSearchAnim() (tea.Model, tea.Cmd) {
	o := m.skillSearch
	if !o.growing {
		return m, nil
	}
	o.w, o.wVel = o.spring.Update(o.w, o.wVel, o.targetW)
	o.h, o.hVel = o.spring.Update(o.h, o.hVel, o.targetH)
	if springSettled(o) {
		o.w, o.h = o.targetW, o.targetH
		o.wVel, o.hVel = 0, 0
		o.growing = false
		// Focus the box; drop the blink command so nothing ticks once settled.
		o.box.Focus()
		return m, nil
	}
	return m, animTick()
}

// springSettled reports whether both dimensions are close enough to their
// targets, and moving slowly enough, to stop the animation.
func springSettled(o *searchOverlay) bool {
	return math.Abs(o.w-o.targetW) < settleEps &&
		math.Abs(o.h-o.targetH) < settleEps &&
		math.Abs(o.wVel) < settleVel &&
		math.Abs(o.hVel) < settleVel
}

// searchBoxPaneHeight is the outer height of the (1) Search pane: its rounded
// border (2), its numbered title row, and the single input row.
const searchBoxPaneHeight = 4

// searchMinColumnHeight is the smallest outer height the (2) Results / (3)
// Details panes are allowed to shrink to.
const searchMinColumnHeight = 5

// renderSkillSearch draws the overlay shell at its current animated size. While
// growing it shows only the title (an empty shell — no heavy content mid-tween);
// once settled it shows three bordered, numbered panes — (1) Search on top,
// (2) Results and (3) Details side by side — inside the outer "Skill Search"
// frame. The focused pane carries the active border; the others a muted one.
func (m Model) renderSkillSearch() string {
	o := m.skillSearch
	w := int(math.Round(o.w))
	h := int(math.Round(o.h))
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}

	title := lipgloss.NewStyle().Foreground(m.theme.Accent).Bold(true).Render("Skill Search")

	body := title
	if !o.growing {
		body = lipgloss.JoinVertical(lipgloss.Left,
			title,
			m.renderSearchInputPane(),
			m.renderSearchColumns(),
		)
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.ActiveBorder).
		Padding(0, 1).
		Width(w).
		Height(h).
		Render(body)
}

// searchPane draws one numbered, bordered inner pane, mirroring the main
// browser's pane styling: an accent numbered title (with an optional pre-styled
// extra riding the same row, e.g. the Results sort bar), a rounded border in the
// active color when that zone is focused else the muted border color, and its
// content clipped to BOTH the pane's inner width and height so lipgloss can never
// re-wrap a long line into extra rows and grow the pane past its Height.
func (m Model) searchPane(z searchZone, title string, width, height int, content string) string {
	border := m.theme.Border
	if m.skillSearch.zone == z {
		border = m.theme.ActiveBorder
	}
	contentW := width - paneBorderPad
	if contentW < 1 {
		contentW = 1
	}
	head := truncate(lipgloss.NewStyle().Foreground(m.theme.Accent).Render(title), contentW)
	innerH := height - paneVerticalChrome - 1 // border rows + numbered title row
	body := head + "\n" + clipBox(content, contentW, innerH)
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1)
	if width > 0 {
		style = style.Width(width)
	}
	if height > 0 {
		style = style.Height(height)
	}
	return style.Render(body)
}

// renderSearchInputPane draws the (1) Search pane: the query input across the
// full inner width, on top.
func (m Model) renderSearchInputPane() string {
	return m.searchPane(zoneBox, "(1) Search", m.searchInnerWidth(), searchBoxPaneHeight, m.renderSkillSearchBox())
}

// renderSearchColumns draws the (2) Results and (3) Details panes side by side
// beneath the search box. The sort bar is the first content row of the Results
// pane, beneath its title.
func (m Model) renderSearchColumns() string {
	paneH := m.searchColumnHeight()
	// Mirror the (2) Skills list: the sort row and a divider sit beneath the title,
	// above the rows.
	resultsBody := m.renderSortBar() + "\n" +
		m.divider(m.searchResultsWidth(), false) + "\n" +
		m.renderSkillSearchResults()
	results := m.searchPane(zoneList, "(2) Results", m.searchResultsPaneWidth(), paneH, resultsBody)
	detail := m.searchPane(zoneDetail, "(3) Details", m.searchDetailPaneWidth(), paneH,
		m.renderSkillSearchDetail())
	return lipgloss.JoinHorizontal(lipgloss.Top,
		results,
		strings.Repeat(" ", searchPaneGap),
		detail,
	)
}

// renderSkillSearchResults renders the results pane beneath the search box,
// reflecting the query lifecycle: a hint under two characters, the loading
// spinner while a request runs, and the ranked list once results land.
func (m Model) renderSkillSearchResults() string {
	o := m.skillSearch
	switch o.state {
	case searchIdle, searchTooShort:
		// Nothing to list yet; the (1) Search box's own placeholder ("Search skills…")
		// is the guidance, so the Results rows area stays empty rather than repeating
		// input guidance here.
		return ""
	case searchLoading:
		frame := spinnerFrames[o.spinnerFrame%len(spinnerFrames)]
		return lipgloss.NewStyle().Foreground(m.theme.Accent).Render(frame)
	case searchEmpty:
		q := strings.TrimSpace(o.box.Value())
		return lipgloss.NewStyle().Foreground(m.theme.Muted).
			Render(fmt.Sprintf("No skills found for %q", q))
	case searchError:
		return lipgloss.NewStyle().Foreground(m.theme.Muted).
			Render("Search failed — space to retry")
	case searchOK:
		capacity := m.searchResultsCapacity()
		start, end := windowBounds(len(o.results), o.selected, capacity)
		rows := make([]string, 0, (end-start)*2)
		for i := start; i < end; i++ {
			rows = append(rows, m.mktRow(o.results[i], i == o.selected)...)
		}
		return lipgloss.JoinVertical(lipgloss.Left, rows...)
	default:
		return ""
	}
}

// renderSortBar draws the sort bar shown on its own row beneath the (2) Results
// title: the three sort keys with their full labels (r relevance, p popularity,
// n name). The active key is highlighted and carries an arrow showing its
// direction, so the same-key asc/desc toggle is visible (design decision 5).
func (m Model) renderSortBar() string {
	o := m.skillSearch
	arrow := "↑"
	if o.sortDir == marketplace.Desc {
		arrow = "↓"
	}
	fields := []struct {
		key   string
		label string
		field marketplace.SortField
	}{
		{"r", "relevance", marketplace.SortRelevance},
		{"p", "popularity", marketplace.SortInstalls},
		{"n", "name", marketplace.SortName},
	}
	active := lipgloss.NewStyle().Foreground(m.theme.Accent).Bold(true)
	muted := lipgloss.NewStyle().Foreground(m.theme.Muted)
	parts := make([]string, len(fields))
	for i, f := range fields {
		text := f.key + " " + f.label
		if f.field == o.sortKey {
			parts[i] = active.Render(text + arrow)
		} else {
			parts[i] = muted.Render(text)
		}
	}
	return strings.Join(parts, "  ")
}

// mktTabLabels are the compact tab labels for the overlay's (3) Details pane,
// used only when the pane is too narrow for the full labels (renderTabsFor) that
// the installed-skill browser shows. A wrapped tab bar would steal a row and push
// the SKILL.md body off-screen, so on a narrow pane the labels are shortened to
// keep the bar on one line instead.
var mktTabLabels = []struct {
	t     tab
	label string
}{
	{tabSkill, "(i)SKILL"},
	{tabReferences, "(r)Refs"},
	{tabScripts, "(s)Scripts"},
	{tabAssets, "(a)Assets"},
}

// renderMktTabs draws the overlay Details pane's four-tab bar with the given tab
// highlighted. It uses the same full labels as the installed-skill browser when
// they fit the Details pane on one line, and only falls back to the compact
// labels on a pane too narrow for them.
func (m Model) renderMktTabs(active tab) string {
	if full := m.renderTabsFor(active); lipgloss.Width(full) <= m.searchDetailWidth() {
		return full
	}
	parts := make([]string, 0, len(mktTabLabels))
	for _, l := range mktTabLabels {
		style := lipgloss.NewStyle().Foreground(m.theme.Muted)
		if active == l.t {
			style = lipgloss.NewStyle().Foreground(m.theme.Accent).Bold(true)
		}
		parts = append(parts, style.Render(l.label))
	}
	return strings.Join(parts, " ")
}

// searchColumnHeight is the outer height of the (2) Results and (3) Details
// panes: the overlay's inner height, less the "Skill Search" title row and the
// (1) Search pane, is shared between them.
func (m Model) searchColumnHeight() int {
	innerH := int(math.Round(m.skillSearch.h)) - paneVerticalChrome
	h := innerH - 1 - searchBoxPaneHeight
	if h < searchMinColumnHeight {
		h = searchMinColumnHeight
	}
	return h
}

// searchColumnContentHeight is the number of rows available inside a Results /
// Details pane, below its numbered title (and inside its border).
func (m Model) searchColumnContentHeight() int {
	h := m.searchColumnHeight() - paneVerticalChrome - 1 // border rows + numbered title
	if h < 1 {
		h = 1
	}
	return h
}

// searchResultsCapacity is how many two-line result rows fit in the Results
// pane, below its numbered title, the sort bar, and the divider (two content rows).
func (m Model) searchResultsCapacity() int {
	return (m.searchColumnContentHeight() - 2) / 2
}

// searchPaneGap is the blank column between the Results and Details panes. It is
// 0 so both panes keep the widest possible content: at the default size the extra
// column is what lets the (3) Details pane render the SKILL.md body, not just its
// frontmatter. The panes' own borders separate them.
const searchPaneGap = 0

// searchInnerWidth is the content width inside the overlay's border and padding
// — the width the (1) Search pane and the Results+Details row each fill.
func (m Model) searchInnerWidth() int {
	w := int(math.Round(m.skillSearch.w)) - paneBorderPad
	if w < 1 {
		w = 1
	}
	return w
}

// searchResultsContentWidth is the width the (2) Results pane wants and no more:
// its two-line rows are `<name>` over `<source> · <Install Count> installs`, and
// the widest of those fits in this many columns. The Results pane is capped here
// so every extra column of a wide terminal goes to the (3) Details pane (where the
// SKILL.md body and full tab bar live) instead of being split 50/50.
const searchResultsContentWidth = 46

// searchResultsFloor / searchDetailFloor are the smallest content widths each
// pane may shrink to on a cramped terminal before the split gives up its
// preferences and divides the room evenly.
const (
	searchResultsFloor = 24
	searchDetailFloor  = 30
)

// searchResultsPaneWidth is the outer width of the (2) Results pane. Results is
// capped at the width it needs (searchResultsContentWidth) and the (3) Details
// pane takes the whole remainder, so a wide terminal grows Details, not Results.
// When there is not enough room for Results' cap plus Details' floor, Results
// yields down to its own floor, and if even that will not fit the two split evenly.
func (m Model) searchResultsPaneWidth() int {
	inner := m.searchInnerWidth() - searchPaneGap // the two pane outer widths share this
	avail := inner - 2*paneBorderPad              // the two panes' content widths share this
	if avail < 2 {
		w := inner / 2
		if w < 1 {
			w = 1
		}
		return w
	}
	results := searchResultsContentWidth
	if avail < searchResultsFloor+searchDetailFloor {
		// Too tight for both floors: split evenly so neither pane collapses.
		results = avail / 2
	} else {
		// Cap Results and hand the rest to Details; never starve Details below its
		// floor, and never drop Results below its own.
		if results > avail-searchDetailFloor {
			results = avail - searchDetailFloor
		}
		if results < searchResultsFloor {
			results = searchResultsFloor
		}
	}
	if results < 1 {
		results = 1
	}
	if results > avail-1 {
		results = avail - 1
	}
	return results + paneBorderPad
}

// searchDetailPaneWidth is the outer width of the (3) Details pane: whatever the
// Results pane and the gap leave of the inner width.
func (m Model) searchDetailPaneWidth() int {
	w := m.searchInnerWidth() - searchPaneGap - m.searchResultsPaneWidth()
	if w < 1 {
		w = 1
	}
	return w
}

// searchResultsWidth is the content width inside the (2) Results pane's border.
func (m Model) searchResultsWidth() int {
	w := m.searchResultsPaneWidth() - paneBorderPad
	if w < 1 {
		w = 1
	}
	return w
}

// searchDetailWidth is the content width inside the (3) Details pane's border.
func (m Model) searchDetailWidth() int {
	w := m.searchDetailPaneWidth() - paneBorderPad
	if w < 1 {
		w = 1
	}
	return w
}

// clipBox clips s to a box of at most n rows and width columns: it keeps the
// first n lines and truncates each to width, so no single long line can wrap and
// grow the pane past its Height, and no line can overrun its Width. A line already
// within width is returned unchanged (truncate is a no-op), so pre-styled content
// keeps its ANSI intact.
func clipBox(s string, width, n int) string {
	if n < 1 {
		n = 1
	}
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	for i, l := range lines {
		lines[i] = truncate(l, width)
	}
	return strings.Join(lines, "\n")
}

// renderSkillSearchDetail renders the detail pane for the selected Marketplace
// Skill in the same four-tab layout as the installed-skill browser: a tab bar,
// a windowed file list for the file tabs (with subfocus dividers), and the
// rendered content with a scrollbar. Before the file tree is downloaded it shows
// the loading spinner (while in flight) or nothing.
func (m Model) renderSkillSearchDetail() string {
	o := m.skillSearch
	if len(o.results) == 0 {
		return ""
	}
	ref := o.results[o.selected].InstallRef()
	if _, ok := o.files[ref]; !ok {
		if o.dlLoading {
			frame := spinnerFrames[o.dlSpinnerFrame%len(spinnerFrames)]
			return lipgloss.NewStyle().Foreground(m.theme.Accent).Render(frame)
		}
		if o.dlError {
			return lipgloss.NewStyle().Foreground(m.theme.Muted).
				Render("Couldn't load files — space to retry")
		}
		return ""
	}

	width := m.searchDetailWidth()
	contentRows, fileCap, hasFiles := m.mktDetailLayout()
	m.syncMktDetail()

	// Metadata block: the Marketplace Skill's name and `source · Install Count`,
	// then a divider and the tab bar — the same shape as the installed-skill
	// browser's Details pane.
	sel := o.results[o.selected]
	bold := lipgloss.NewStyle().Foreground(m.theme.Fg).Bold(true)
	dim := lipgloss.NewStyle().Foreground(m.theme.Muted)
	parts := []string{
		bold.Render(truncate(sel.Name, width)),
		dim.Render(truncate(fmt.Sprintf("%s · %s installs", sel.Source, commaInt(sel.Installs)), width)),
		m.divider(width, false),
		m.renderMktTabs(o.detailTab),
	}
	if hasFiles {
		fileLines := m.renderMktFileList(m.mktCurrentFiles())
		start, end := windowBounds(len(fileLines), o.detailFileSel, fileCap)
		fileLines = fileLines[start:end]
		parts = append(parts, m.divider(width, o.detailSubfocus == subfocusList))
		parts = append(parts, fileLines...)
		parts = append(parts, m.divider(width, o.detailSubfocus == subfocusContent))
		parts = append(parts, m.mktContentWithScrollbar(contentRows)...)
	} else {
		// SKILL.md is the fast first view: a divider separates the tab bar from the
		// scrollable body, with no file list.
		parts = append(parts, m.divider(width, false))
		parts = append(parts, strings.Split(o.detailVP.View(), "\n")...)
	}
	return strings.Join(parts, "\n")
}

// mktClassified returns the selected Marketplace Skill's downloaded, classified
// file tree, or ok=false when nothing is cached for it yet.
func (m Model) mktClassified() (skillMD string, refs, scripts, assets []marketplace.File, ok bool) {
	o := m.skillSearch
	if len(o.results) == 0 {
		return "", nil, nil, nil, false
	}
	sf, cached := o.files[o.results[o.selected].InstallRef()]
	if !cached {
		return "", nil, nil, nil, false
	}
	skillMD, refs, scripts, assets = marketplace.Classify(sf.Files)
	return skillMD, refs, scripts, assets, true
}

// mktCurrentFiles is the file list of the open file tab, or nil for SKILL.md.
func (m Model) mktCurrentFiles() []marketplace.File {
	_, refs, scripts, assets, ok := m.mktClassified()
	if !ok {
		return nil
	}
	switch m.skillSearch.detailTab {
	case tabReferences:
		return refs
	case tabScripts:
		return scripts
	case tabAssets:
		return assets
	default:
		return nil
	}
}

// mktDetailLayout splits the detail pane's rows into the content viewport height,
// the file-list window capacity, and whether a file list is shown — the overlay
// counterpart of detailLayout, over the searchColumnContentHeight budget. The tab
// bar is compact (one row, renderMktTabs), so exactly one row is budgeted for it
// and the Details pane ends the same height as Results with bottoms aligned.
func (m Model) mktDetailLayout() (contentRows, fileCap int, hasFiles bool) {
	o := m.skillSearch
	// The Details pane mirrors the installed-skill browser: a metadata block, a
	// divider, the tab bar, then (on file tabs) the file list bracketed by dividers,
	// then the content. metaBlock is name + `source · installs`.
	const mktMetaRows = 2
	budget := m.searchColumnContentHeight() - mktMetaRows - 1 - 1 // meta + divider + tab bar
	if o.detailTab == tabSkill {
		// SKILL.md has no file list; one divider separates the tab bar from the body.
		budget--
		if budget < 1 {
			budget = 1
		}
		return budget, 0, false
	}
	fileRows := len(m.mktCurrentFiles())
	if fileRows == 0 {
		fileRows = 1 // the "No files" line
	}
	budget -= 2 // two dividers (before file list, before content)
	fileCap = fileRows
	if half := budget / 2; fileCap > half {
		fileCap = half
	}
	if fileCap < 1 {
		fileCap = 1
	}
	shown := fileRows
	if shown > fileCap {
		shown = fileCap
	}
	contentRows = budget - shown
	if contentRows < 1 {
		contentRows = 1
	}
	return contentRows, fileCap, true
}

// mktContentWidth is the width the detail content renders at: the full detail
// width for SKILL.md (no scrollbar), one column less for the file tabs whose
// scrollbar occupies the reserved gutter.
func (m Model) mktContentWidth() int {
	w := m.searchDetailWidth()
	if m.skillSearch.detailTab != tabSkill {
		w -= scrollbarWidth
	}
	if w < 1 {
		w = 1
	}
	return w
}

// syncMktDetail sizes the detail content viewport and loads the current file's
// rendered (memoized) content, so scrolling clamps to the right height.
func (m Model) syncMktDetail() {
	o := m.skillSearch
	rows, _, _ := m.mktDetailLayout()
	o.detailVP.SetWidth(m.mktContentWidth())
	o.detailVP.SetHeight(rows)
	o.detailVP.SetContent(m.mktContent())
}

// mktContent is the rendered content of the current tab and file, served from the
// per-file memo and rendered on first open (lazy rendering, not lazy fetching).
func (m Model) mktContent() string {
	o := m.skillSearch
	ref := ""
	if len(o.results) > 0 {
		ref = o.results[o.selected].InstallRef()
	}
	width := m.mktContentWidth()
	memoKey := fmt.Sprintf("%s|%d|%d|%d", ref, o.detailTab, o.detailFileSel, width)
	if o.rendered != nil {
		if s, ok := o.rendered[memoKey]; ok {
			return s
		}
	}
	out := m.renderMktContentUncached(width)
	if o.rendered == nil {
		o.rendered = make(map[string]string)
	}
	o.rendered[memoKey] = out
	return out
}

// renderMktContentUncached renders the current tab and file over the in-memory
// file contents: SKILL.md and Markdown references via Glamour, scripts via
// Chroma, assets as "No preview available" — never touching disk.
func (m Model) renderMktContentUncached(width int) string {
	skillMD, refs, scripts, assets, ok := m.mktClassified()
	if !ok {
		return ""
	}
	switch m.skillSearch.detailTab {
	case tabSkill:
		_, raw, body, _ := skills.ParseSkillMarkdown([]byte(skillMD))
		md := skillMarkdown(skills.Skill{Frontmatter: raw, Body: body})
		out, err := render.Markdown(md, width)
		if err != nil {
			return md
		}
		return render.TrimSurroundingBlankLines(out)
	case tabReferences:
		return m.mktMarkdownContent(refs, width)
	case tabScripts:
		return m.mktCodeContent(scripts)
	case tabAssets:
		if len(assets) == 0 {
			return ""
		}
		return "No preview available"
	}
	return ""
}

// mktSelectedFile is the file the detail's file selection points at, or ok=false
// when the tab has no files.
func (m Model) mktSelectedFile(files []marketplace.File) (marketplace.File, bool) {
	if len(files) == 0 {
		return marketplace.File{}, false
	}
	sel := m.skillSearch.detailFileSel
	if sel < 0 || sel >= len(files) {
		sel = 0
	}
	return files[sel], true
}

// mktMarkdownContent renders the selected reference: Glamour for .md, raw text
// otherwise.
func (m Model) mktMarkdownContent(files []marketplace.File, width int) string {
	f, ok := m.mktSelectedFile(files)
	if !ok {
		return ""
	}
	if strings.HasSuffix(strings.ToLower(f.Path), ".md") {
		out, err := render.Markdown(f.Contents, width)
		if err == nil {
			return render.TrimSurroundingBlankLines(out)
		}
	}
	return f.Contents
}

// mktCodeContent renders the selected script with Chroma syntax highlighting.
func (m Model) mktCodeContent(files []marketplace.File) string {
	f, ok := m.mktSelectedFile(files)
	if !ok {
		return ""
	}
	out, err := render.Code(f.Contents, f.Path)
	if err != nil {
		return f.Contents
	}
	return out
}

// renderMktFileList renders the detail file list for the file tabs, reusing the
// installed-skill browser's file-list rendering over the Marketplace file paths.
func (m Model) renderMktFileList(files []marketplace.File) []string {
	names := make([]string, len(files))
	for i, f := range files {
		names[i] = f.Path
	}
	return m.renderFileNames(names, m.skillSearch.detailFileSel, m.searchDetailWidth())
}

// mktContentWithScrollbar lays out the detail content viewport with a scrollbar
// in the reserved right-hand column, mirroring renderContentWithScrollbar but
// over the overlay's own viewport.
func (m Model) mktContentWithScrollbar(rows int) []string {
	o := m.skillSearch
	cw := m.mktContentWidth()
	lines := strings.Split(o.detailVP.View(), "\n")
	bar := scrollbar(rows, o.detailVP.TotalLineCount(), o.detailVP.ScrollPercent(), m.theme)
	pad := lipgloss.NewStyle().Width(cw)
	out := make([]string, rows)
	for i := 0; i < rows; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		gutter := " "
		if i < len(bar) {
			gutter = bar[i]
		}
		out[i] = pad.Render(line) + gutter
	}
	return out
}

// mktRow renders one Marketplace Skill as a two-line row mirroring skillRow: the
// name on line one and `source · <Install Count> installs` on line two, with the
// selected row drawn in an elevated highlight band.
func (m Model) mktRow(s marketplace.MarketplaceSkill, selected bool) []string {
	textW := m.searchResultsWidth()
	if textW < 1 {
		textW = 1
	}
	name := "  " + s.Name
	meta := "  " + s.Source + " · " + commaInt(s.Installs) + " installs"

	if selected {
		nameStyle := lipgloss.NewStyle().
			Foreground(m.theme.Accent).
			Background(m.theme.Elevated).
			Bold(true).
			Width(textW)
		metaStyle := lipgloss.NewStyle().
			Foreground(m.theme.Fg).
			Background(m.theme.Elevated).
			Width(textW)
		return []string{
			nameStyle.Render(truncate(name, textW)),
			metaStyle.Render(truncate(meta, textW)),
		}
	}

	nameStyle := lipgloss.NewStyle().Foreground(m.theme.Fg)
	metaStyle := lipgloss.NewStyle().Foreground(m.theme.Muted)
	return []string{
		nameStyle.Render(truncate(name, textW)),
		metaStyle.Render(truncate(meta, textW)),
	}
}

// commaInt formats a non-negative integer with comma thousands separators, e.g.
// 540366 -> "540,366". It is the local Install Count formatter (design decision
// 6: dustin/go-humanize stays an indirect dependency).
func commaInt(n int) string {
	s := strconv.Itoa(n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var b strings.Builder
	for i := range len(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteByte(s[i])
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// renderSkillSearchBox renders the overlay's search input across the (1) Search
// pane's inner width, so a long query scrolls within the box. The pane's
// numbered title already names it, so the input carries no separate label.
func (m Model) renderSkillSearchBox() string {
	in := m.skillSearch.box
	// Leave one column for the cursor cell: a focused textinput renders one column
	// wider than its set width, which would otherwise clip to a trailing "…".
	w := m.searchInnerWidth() - paneBorderPad - 1
	if w < 1 {
		w = 1
	}
	in.SetWidth(w)
	return in.View()
}
