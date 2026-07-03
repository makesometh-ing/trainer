package app

import (
	"math"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/makesometh-ing/trainer/internal/render"
	"github.com/makesometh-ing/trainer/internal/skills"
)

func (m Model) View() tea.View {
	if m.tooSmall() {
		v := tea.NewView(m.renderTooSmall())
		v.AltScreen = true
		return v
	}

	scope := m.renderScope()
	list := m.renderSkillList()
	detail := m.renderDetail()

	body := lipgloss.JoinHorizontal(lipgloss.Top, scope, list, detail)
	if m.wizard != nil {
		body = m.overlayCenter(body, m.renderWizard())
	}
	if m.palette {
		body = m.overlayCenter(body, m.renderPalette())
	}
	if m.help {
		body = m.overlayCenter(body, m.renderHelp())
	}
	if m.confirm != nil {
		body = m.overlayCenter(body, m.renderConfirm())
	}
	// The footer is always the bottom row. renderFooter returns "" while a modal
	// is open, so the reserved row stays blank rather than shifting the panes.
	body = lipgloss.JoinVertical(lipgloss.Left, body, m.renderFooter())
	v := tea.NewView(body)
	v.AltScreen = true
	return v
}

func (m Model) tooSmall() bool {
	return m.width > 0 && m.height > 0 && (m.width < minWidth || m.height < minHeight)
}

func (m Model) renderTooSmall() string {
	msg := "[Too small] Resize terminal to view the full app"
	styled := lipgloss.NewStyle().Foreground(m.theme.Accent).Render(msg)
	if m.width <= 0 || m.height <= 0 {
		return styled
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, styled)
}

func (m Model) overlayCenter(base, modal string) string {
	baseW := lipgloss.Width(base)
	baseH := lipgloss.Height(base)
	modalW := lipgloss.Width(modal)
	modalH := lipgloss.Height(modal)

	x := max(0, (baseW-modalW)/2)
	y := max(0, (baseH-modalH)/2)

	comp := lipgloss.NewCompositor(
		lipgloss.NewLayer(base).Z(0),
		lipgloss.NewLayer(modal).X(x).Y(y).Z(1),
	)
	return comp.Render()
}

func (m Model) renderPalette() string {
	title := lipgloss.NewStyle().
		Foreground(m.theme.Accent).
		Bold(true).
		Render("Commands")

	cmd := func(key, label string, disabled bool) string {
		if disabled {
			// A dimmed command is muted and carries a tag naming the reason. The
			// tag never mentions lockfiles or any internal mechanism.
			dim := lipgloss.NewStyle().Foreground(m.theme.Muted)
			return dim.Render("("+key+") "+label) + "  " + dim.Render("disabled without npx")
		}
		k := lipgloss.NewStyle().Foreground(m.theme.Accent).Render("(" + key + ")")
		l := lipgloss.NewStyle().Foreground(m.theme.Fg).Render(label)
		return k + " " + l
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		cmd("a", "add skill", m.addCmdDisabled()),
		cmd("d", "delete skill", m.deleteCmdDisabled()),
		cmd("u", "update all skills", m.addCmdDisabled()),
		"",
		cmd("esc", "cancel", false),
	)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.ActiveBorder).
		Padding(0, 2).
		Render(body)
}

func (m Model) renderSkillList() string {
	title := lipgloss.NewStyle().Foreground(m.theme.Accent).Render("(2) Skills")
	header := strings.Join([]string{
		title,
		m.renderSearchBox(),
		m.renderFilter(),
		m.divider(m.listWidth()-paneBorderPad, false),
	}, "\n")

	vis := m.visibleSkills()
	rowsPerSkill := 2
	avail := m.paneContentHeight() - lipgloss.Height(header)
	if avail < rowsPerSkill {
		avail = rowsPerSkill
	}
	capacity := avail / rowsPerSkill

	start, end := windowBounds(len(vis), m.selected, capacity)

	lines := []string{header}
	for i := start; i < end; i++ {
		lines = append(lines, m.skillRow(vis[i], i == m.selected)...)
	}

	return m.pane(paneSkills, m.listWidth(), m.paneHeight(), strings.Join(lines, "\n"))
}

// windowBounds returns the [start, end) slice of a list of length n that keeps
// the selected index visible within a window of the given capacity.
func windowBounds(n, selected, capacity int) (int, int) {
	if capacity < 1 {
		capacity = 1
	}
	if n <= capacity {
		return 0, n
	}
	start := selected - capacity/2
	if start < 0 {
		start = 0
	}
	end := start + capacity
	if end > n {
		end = n
		start = end - capacity
	}
	return start, end
}

func (m Model) skillRow(s skills.Skill, selected bool) []string {
	textW := m.listWidth() - paneBorderPad
	if textW < 1 {
		textW = 1
	}
	name := "  " + s.Name
	meta := "  " + skillMeta(s)

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

func skillMeta(s skills.Skill) string {
	if s.Lock != nil && s.Lock.Source != "" {
		return s.Lock.Source
	}
	return "local"
}

func (m Model) renderDetail() string {
	title := lipgloss.NewStyle().Foreground(m.theme.Accent).Render("(3) Details")
	s, ok := m.selectedSkill()
	if !ok {
		return m.pane(paneDetail, m.detailPaneWidth(), m.paneHeight(), title+"\nNo skill selected")
	}

	textW := m.detailWidth()
	meta := m.metaBlock(title, s, textW)
	tabs := m.renderTabs()
	fileLines := m.fileListLines()
	contentRows, fileCap, hasFiles := m.detailLayout()

	m.content.SetWidth(m.contentWidth())
	m.content.SetHeight(contentRows)
	m.content.SetContent(m.currentContent())

	parts := append([]string{}, meta...)
	parts = append(parts, m.divider(textW, false), tabs)
	if hasFiles {
		start, end := windowBounds(len(fileLines), m.fileSel, fileCap)
		fileLines = fileLines[start:end]
		parts = append(parts, m.divider(textW, m.subfocus == subfocusList))
		parts = append(parts, fileLines...)
		parts = append(parts, m.divider(textW, m.subfocus == subfocusContent))
	} else {
		parts = append(parts, m.divider(textW, false))
	}
	parts = append(parts, m.renderContentWithScrollbar(contentRows)...)

	return m.pane(paneDetail, m.detailPaneWidth(), m.paneHeight(), strings.Join(parts, "\n"))
}

// detailLayout splits the Details pane's rows into the content viewport height,
// the file-list window capacity, and whether a file list is shown. renderDetail
// and the viewport sizing both use it, so the height the content scrolls by is
// the same height it is drawn at (otherwise the scrollbar cannot reach the
// bottom because the viewport clamps scrolling to the wrong height).
func (m Model) detailLayout() (contentRows, fileCap int, hasFiles bool) {
	s, ok := m.selectedSkill()
	if !ok {
		return m.paneContentHeight(), 0, false
	}
	meta := m.metaBlock("", s, m.detailWidth())
	fileLines := m.fileListLines()
	hasFiles = fileLines != nil

	dividers := 2 // after meta, before content
	if hasFiles {
		dividers = 3 // after meta, before files, before content
	}
	budget := m.paneContentHeight() - len(meta) - 1 /* tab bar */ - dividers
	if budget < 2 {
		budget = 2
	}
	if !hasFiles {
		return budget, 0, false
	}

	// The file list takes at most half the budget and is windowed around the
	// selection; the content viewport gets the rest.
	fileCap = len(fileLines)
	if half := budget / 2; fileCap > half {
		fileCap = half
	}
	if fileCap < 1 {
		fileCap = 1
	}
	fileRows := len(fileLines)
	if fileRows > fileCap {
		fileRows = fileCap
	}
	contentRows = budget - fileRows
	if contentRows < 1 {
		contentRows = 1
	}
	return contentRows, fileCap, true
}

// divider renders a horizontal rule spanning the detail content width. The rule
// above the active subfocus section (file list or content) is drawn in the
// accent color, so the focused section is visible without a text header.
func (m Model) divider(width int, active bool) string {
	if width < 1 {
		width = 1
	}
	c := m.theme.Border
	if active {
		c = m.theme.Accent
	}
	return lipgloss.NewStyle().Foreground(c).Render(strings.Repeat("─", width))
}

// contentWidth is the width available to rendered content: the detail content
// width less one column reserved for the scrollbar gutter.
func (m Model) contentWidth() int {
	w := m.detailWidth() - scrollbarWidth
	if w < 1 {
		w = 1
	}
	return w
}

// renderContentWithScrollbar lays out the content viewport with the scrollbar in
// the reserved right-hand column, one glyph per row.
func (m Model) renderContentWithScrollbar(rows int) []string {
	lines := strings.Split(m.content.View(), "\n")
	bar := m.scrollbarColumn(rows)
	pad := lipgloss.NewStyle().Width(m.contentWidth())
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

// scrollbarColumn returns one glyph per content row: a track with a solid thumb
// whose length is the visible fraction of the content and whose position tracks
// the scroll offset. When all the content fits, the column is blank.
func (m Model) scrollbarColumn(rows int) []string {
	col := make([]string, rows)
	total := m.content.TotalLineCount()
	if rows <= 0 || total <= rows {
		for i := range col {
			col[i] = " "
		}
		return col
	}
	thumb := int(math.Round(float64(rows) / float64(total) * float64(rows)))
	if thumb < 1 {
		thumb = 1
	}
	if thumb > rows {
		thumb = rows
	}
	start := int(math.Round(m.content.ScrollPercent() * float64(rows-thumb)))
	if start < 0 {
		start = 0
	}
	if start > rows-thumb {
		start = rows - thumb
	}
	track := lipgloss.NewStyle().Foreground(m.theme.Border)
	thumbStyle := lipgloss.NewStyle().Foreground(m.theme.Accent)
	for i := range col {
		if i >= start && i < start+thumb {
			col[i] = thumbStyle.Render("█")
		} else {
			col[i] = track.Render("░")
		}
	}
	return col
}

func (m Model) metaBlock(title string, s skills.Skill, textW int) []string {
	dim := lipgloss.NewStyle().Foreground(m.theme.Muted)
	lines := []string{
		title,
		lipgloss.NewStyle().Foreground(m.theme.Fg).Bold(true).Render(truncate(s.Name, textW)),
	}
	if s.Lock != nil {
		if s.Lock.Source != "" {
			lines = append(lines, dim.Render(truncate("source: "+s.Lock.Source, textW)))
		}
		if s.Lock.SourceURL != "" {
			lines = append(lines, dim.Render(truncate("sourceUrl: "+s.Lock.SourceURL, textW)))
		}
		if s.Lock.SkillPath != "" {
			lines = append(lines, dim.Render(truncate("skillPath: "+s.Lock.SkillPath, textW)))
		}
	}
	lines = append(lines, dim.Render(truncate("path: "+s.Path, textW)))
	return lines
}

func (m Model) fileListLines() []string {
	if m.tab == tabSkill {
		return nil
	}
	// References, Scripts, and Assets all show a file list, which reads "No files"
	// when the skill bundles none of that kind.
	files, _ := m.currentFilesAndRenderer()
	return m.renderFileList(files)
}

func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= w {
		return s
	}
	if w <= 1 {
		return "…"
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes))+1 > w {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

func (m Model) renderTabs() string {
	labels := []struct {
		t     tab
		label string
	}{
		{tabSkill, "(i) SKILL.md"},
		{tabReferences, "(r) References"},
		{tabScripts, "(s) Scripts"},
		{tabAssets, "(a) Assets"},
	}
	parts := make([]string, 0, len(labels))
	for _, l := range labels {
		style := lipgloss.NewStyle().Foreground(m.theme.Muted)
		if m.tab == l.t {
			style = lipgloss.NewStyle().Foreground(m.theme.Accent).Bold(true)
		}
		parts = append(parts, style.Render(l.label))
	}
	return strings.Join(parts, " ")
}

func (m Model) currentContent() string {
	return render.TrimSurroundingBlankLines(m.rawContent())
}

func (m Model) rawContent() string {
	if m.tab == tabSkill {
		s, ok := m.selectedSkill()
		if !ok {
			return ""
		}
		md := skillMarkdown(s)
		out, err := render.Markdown(md, m.contentWidth())
		if err != nil {
			return md
		}
		return out
	}
	files, content := m.currentFilesAndRenderer()
	if content == nil || len(files) == 0 {
		return ""
	}
	sel := m.fileSel
	if sel < 0 || sel >= len(files) {
		sel = 0
	}
	return content(files[sel])
}

// skillMarkdown builds the SKILL.md tab content: the raw frontmatter as a fenced
// YAML block, so the renderer keeps its `---` fences and every field as literal
// text rather than turning `---` into a horizontal rule, followed by the body.
func skillMarkdown(s skills.Skill) string {
	fm := strings.TrimRight(s.Frontmatter, "\n")
	if fm == "" {
		return s.Body
	}
	return "```yaml\n" + fm + "\n```\n\n" + s.Body
}

func (m Model) currentFiles() []skills.SkillFile {
	files, _ := m.currentFilesAndRenderer()
	return files
}

func (m Model) currentFilesAndRenderer() ([]skills.SkillFile, func(skills.SkillFile) string) {
	s, ok := m.selectedSkill()
	if !ok {
		return nil, nil
	}
	switch m.tab {
	case tabReferences:
		return s.References, m.renderReferenceContent
	case tabScripts:
		return s.Scripts, m.renderScriptContent
	case tabAssets:
		return s.Assets, func(skills.SkillFile) string { return "No preview available" }
	default:
		return nil, nil
	}
}

func (m Model) renderReferenceContent(f skills.SkillFile) string {
	data, err := os.ReadFile(f.Path)
	if err != nil {
		return "unable to read " + f.Name
	}
	if strings.HasSuffix(strings.ToLower(f.Name), ".md") {
		out, rerr := render.Markdown(string(data), m.contentWidth())
		if rerr == nil {
			return out
		}
	}
	return string(data)
}

func (m Model) renderScriptContent(f skills.SkillFile) string {
	data, err := os.ReadFile(f.Path)
	if err != nil {
		return "unable to read " + f.Name
	}
	out, rerr := render.Code(string(data), f.Name)
	if rerr != nil {
		return string(data)
	}
	return out
}

func (m Model) detailWidth() int {
	if m.width <= 0 {
		return 80
	}
	w := m.detailPaneWidth() - paneBorderPad
	if w < 1 {
		w = 1
	}
	return w
}

func (m Model) renderFileList(files []skills.SkillFile) []string {
	if len(files) == 0 {
		return []string{lipgloss.NewStyle().Foreground(m.theme.Muted).Render("No files")}
	}
	textW := m.detailWidth()
	if textW < 1 {
		textW = 1
	}
	lines := make([]string, 0, len(files))
	for i, f := range files {
		name := "  " + f.Name
		if i == m.fileSel {
			// The selected file is marked by an elevated highlight band, matching
			// the skill list, rather than a caret.
			lines = append(lines, lipgloss.NewStyle().
				Foreground(m.theme.Accent).
				Background(m.theme.Elevated).
				Bold(true).
				Width(textW).
				Render(truncate(name, textW)))
			continue
		}
		lines = append(lines, lipgloss.NewStyle().Foreground(m.theme.Fg).Render(truncate(name, textW)))
	}
	return lines
}

func (m Model) pane(p pane, width, height int, content string) string {
	border := m.theme.Border
	if m.focus == p {
		border = m.theme.ActiveBorder
	}
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
	return style.Render(content)
}

const (
	scopePaneWidth = 18
	minListWidth   = 16
)

// scrollbarWidth is the single column reserved on the right of the detail
// content for the scrollbar.
const scrollbarWidth = 1

// paneBorderPad is the horizontal overhead a pane adds around its content
// width: a 1-cell rounded border and 1-cell padding on each side.
const paneBorderPad = 4

// paneVerticalChrome is the vertical overhead a pane adds: top and bottom
// border rows (padding is 0 vertically).
const paneVerticalChrome = 2

// frameMargin is the blank margin left around the whole app frame.
const frameMargin = 1

// footerHeight is the single row reserved at the bottom for the context footer.
const footerHeight = 1

// paneHeight is the outer height of each pane (including its border), sized so
// the whole frame plus its margin and the footer row fits within the terminal
// height. The footer row is reserved unconditionally, so opening a modal (which
// blanks the footer) does not resize the panes.
func (m Model) paneHeight() int {
	if m.height <= 0 {
		return defaultContentHeight + paneVerticalChrome
	}
	h := m.height - frameMargin - footerHeight
	if h < paneVerticalChrome+1 {
		h = paneVerticalChrome + 1
	}
	return h
}

// paneContentHeight is the number of content rows available inside a pane.
func (m Model) paneContentHeight() int {
	h := m.paneHeight() - paneVerticalChrome
	if h < 1 {
		h = 1
	}
	return h
}

func (m Model) scopeWidth() int {
	return scopePaneWidth
}

func (m Model) listWidth() int {
	if m.width <= 0 {
		return 24
	}
	remaining := m.width - (scopePaneWidth + paneBorderPad)
	w := remaining / 3
	if w < minListWidth {
		w = minListWidth
	}
	return w
}

func (m Model) detailPaneWidth() int {
	if m.width <= 0 {
		return defaultContentWidth
	}
	// A pane's Width is its total rendered width (border and padding included),
	// so the detail pane takes exactly the width the scope and skills panes leave.
	// The three panes then fill the terminal with no dead space on the right.
	return m.width - scopePaneWidth - m.listWidth()
}
