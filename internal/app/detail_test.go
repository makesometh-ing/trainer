package app

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/makesometh-ing/trainer/internal/skills"
)

func TestDetailFileListDoesNotOverflow(t *testing.T) {
	dir := t.TempDir()
	refDir := filepath.Join(dir, "references")
	if err := os.MkdirAll(refDir, 0o755); err != nil {
		t.Fatal(err)
	}
	var refs []skills.SkillFile
	for i := 0; i < 80; i++ {
		name := fmt.Sprintf("ref-%02d.md", i)
		p := filepath.Join(refDir, name)
		if err := os.WriteFile(p, []byte("# "+name+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		refs = append(refs, skills.SkillFile{Name: name, Path: p})
	}
	res := skills.ScanResult{
		Scope:  skills.Scope{Name: "Global", Path: dir},
		Skills: []skills.Skill{{Name: "many", Path: dir, References: refs}},
	}

	const h = 40
	var m tea.Model = NewModel(res)
	m = resize(m, 120, h)
	m = press(m, "3") // focus Details
	m = press(m, "r") // References tab: 80 files

	if gotH := lipgloss.Height(view(m)); gotH > h {
		t.Errorf("detail file list overflowed the terminal: frame height %d > %d", gotH, h)
	}
}

var ansiRE = regexp.MustCompile("\x1b\\[[0-9;]*m")

func plain(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

func detailResult(t *testing.T) skills.ScanResult {
	t.Helper()
	dir := t.TempDir()

	refDir := filepath.Join(dir, "references")
	scriptDir := filepath.Join(dir, "scripts")
	assetDir := filepath.Join(dir, "assets")
	for _, d := range []string{refDir, scriptDir, assetDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	refPath := filepath.Join(refDir, "guide.md")
	if err := os.WriteFile(refPath, []byte("# Reference Heading\n\nBody.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	scriptPath := filepath.Join(scriptDir, "run.go")
	if err := os.WriteFile(scriptPath, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	weirdPath := filepath.Join(scriptDir, "notes.weirdext")
	if err := os.WriteFile(weirdPath, []byte("raw plain content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	assetPath := filepath.Join(assetDir, "logo.png")
	if err := os.WriteFile(assetPath, []byte("binary"), 0o644); err != nil {
		t.Fatal(err)
	}

	skillPath := filepath.Join(dir, "SKILL.md")
	skillBody := "---\nname: alpha\ndescription: First skill\n---\n\n# Alpha Overview\n\nHow to use alpha.\n"
	if err := os.WriteFile(skillPath, []byte(skillBody), 0o644); err != nil {
		t.Fatal(err)
	}

	return skills.ScanResult{
		Scope: skills.Scope{Name: "Global", Path: dir},
		Skills: []skills.Skill{
			{
				Name:        "alpha",
				Description: "First skill",
				Body:        "# Alpha Overview\n\nHow to use alpha.\n",
				Frontmatter: "---\nname: alpha\ndescription: First skill\nlicense: MIT\n---",
				Path:        dir,
				SkillPath:   skillPath,
				References:  []skills.SkillFile{{Name: "guide.md", Path: refPath}},
				Scripts: []skills.SkillFile{
					{Name: "notes.weirdext", Path: weirdPath},
					{Name: "run.go", Path: scriptPath},
				},
				Assets: []skills.SkillFile{{Name: "logo.png", Path: assetPath}},
			},
		},
	}
}

func TestSkillTabRendersSkillBody(t *testing.T) {
	m := press(press(NewModel(detailResult(t)), "3"), "i")

	out := plain(view(m))
	if !strings.Contains(out, "Alpha Overview") {
		t.Errorf("expected SKILL.md tab to render the skill body, got %q", out)
	}
}

func TestSkillTabShowsFrontmatter(t *testing.T) {
	m := press(press(NewModel(detailResult(t)), "3"), "i")

	out := plain(view(m))
	// The frontmatter is shown in full: its fields, including one that is neither
	// name nor description, and its fence lines.
	for _, want := range []string{"name: alpha", "description: First skill", "license: MIT"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected SKILL.md tab to show frontmatter field %q, got:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "---") {
		t.Errorf("expected SKILL.md tab to show the frontmatter --- fences, got:\n%s", out)
	}
	// The body still renders below the frontmatter.
	if !strings.Contains(out, "Alpha Overview") {
		t.Errorf("expected SKILL.md tab to also render the body, got:\n%s", out)
	}
}

func TestReferencesTabShowsFileList(t *testing.T) {
	m := press(press(NewModel(detailResult(t)), "3"), "r")

	out := view(m)
	if !strings.Contains(out, "guide.md") {
		t.Errorf("expected References tab to list reference filenames, got %q", out)
	}
}

func TestSelectingReferenceRendersMarkdown(t *testing.T) {
	m := press(press(NewModel(detailResult(t)), "3"), "r")

	out := plain(view(m))
	if !strings.Contains(out, "Reference Heading") {
		t.Errorf("expected selected reference markdown content, got %q", out)
	}
}

func scriptOnlyResult(t *testing.T, name, body string) skills.ScanResult {
	t.Helper()
	dir := t.TempDir()
	scriptDir := filepath.Join(dir, "scripts")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(scriptDir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return skills.ScanResult{
		Scope: skills.Scope{Name: "Global", Path: dir},
		Skills: []skills.Skill{{
			Name:    "alpha",
			Path:    dir,
			Scripts: []skills.SkillFile{{Name: name, Path: p}},
		}},
	}
}

func TestScriptsTabHighlightsCode(t *testing.T) {
	m := press(press(NewModel(scriptOnlyResult(t, "run.go", "package main\n\nfunc main() {}\n")), "3"), "s")

	out := plain(view(m))
	if !strings.Contains(out, "func main") {
		t.Errorf("expected Scripts tab to show highlighted script source, got %q", out)
	}
}

func TestUnknownScriptExtensionFallsBackToPlainText(t *testing.T) {
	m := press(press(NewModel(scriptOnlyResult(t, "notes.weirdext", "raw plain content")), "3"), "s")

	out := plain(view(m))
	if !strings.Contains(out, "raw plain content") {
		t.Errorf("expected unknown extension to show raw source, got %q", out)
	}
}

func TestAssetsTabShowsNoPreview(t *testing.T) {
	m := press(press(NewModel(detailResult(t)), "3"), "a")

	out := plain(view(m))
	if !strings.Contains(out, "logo.png") {
		t.Errorf("expected Assets tab to list asset filenames, got %q", out)
	}
	if !strings.Contains(out, "No preview available") {
		t.Errorf("expected Assets tab to show No preview available, got %q", out)
	}
}

func longScriptResult(t *testing.T) skills.ScanResult {
	t.Helper()
	dir := t.TempDir()
	scriptDir := filepath.Join(dir, "scripts")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&b, "line-%03d\n", i)
	}
	p := filepath.Join(scriptDir, "big.weirdext")
	if err := os.WriteFile(p, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return skills.ScanResult{
		Scope: skills.Scope{Name: "Global", Path: dir},
		Skills: []skills.Skill{{
			Name:    "alpha",
			Path:    dir,
			Scripts: []skills.SkillFile{{Name: "big.weirdext", Path: p}},
		}},
	}
}

func sized(m tea.Model, w, h int) tea.Model {
	next, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return next
}

func TestContentScrollbarAppearsOnOverflow(t *testing.T) {
	var m tea.Model = NewModel(longScriptResult(t))
	m = sized(m, 120, 24)
	m = press(m, "3")
	m = press(m, "s")

	out := plain(view(m))
	if !strings.Contains(out, "█") {
		t.Errorf("expected a scrollbar thumb on overflowing content, got:\n%s", out)
	}
}

func TestScrollbarReachesBottom(t *testing.T) {
	var m tea.Model = NewModel(longScriptResult(t)) // 100 lines: line-000..line-099
	m = sized(m, 120, 30)
	m = press(m, "3") // focus Details
	m = press(m, "s") // Scripts tab
	m = press(m, "G") // jump to the bottom of the content

	out := plain(view(m))
	if !strings.Contains(out, "█") {
		t.Fatal("expected a scrollbar on overflowing content")
	}
	// The viewport's scroll height must match its render height, or the bottom is
	// never reachable. After G the last content line is visible.
	if !strings.Contains(out, "line-099") {
		t.Errorf("expected the last line visible at the bottom after G, got:\n%s", out)
	}
}

func TestContentScrollbarAbsentWhenContentFits(t *testing.T) {
	var m tea.Model = NewModel(scriptOnlyResult(t, "small.weirdext", "one\ntwo\n"))
	m = sized(m, 120, 40)
	m = press(m, "3")
	m = press(m, "s")

	out := plain(view(m))
	if strings.Contains(out, "█") {
		t.Errorf("expected no scrollbar when content fits the viewport, got:\n%s", out)
	}
}

func TestContentScrollKeysMoveViewport(t *testing.T) {
	var m tea.Model = NewModel(longScriptResult(t))
	m = sized(m, 120, 24)
	m = press(m, "3")
	m = press(m, "s")

	top := plain(view(m))
	if !strings.Contains(top, "line-000") {
		t.Fatalf("expected top of content visible initially, got %q", top)
	}

	m = press(m, "ctrl+d")
	scrolled := plain(view(m))
	if strings.Contains(scrolled, "line-000") {
		t.Errorf("expected ctrl+d to scroll past the first line, got %q", scrolled)
	}

	m = press(m, "g")
	m = press(m, "g")
	back := plain(view(m))
	if !strings.Contains(back, "line-000") {
		t.Errorf("expected gg to return content to top, got %q", back)
	}
}

func twoReferencesResult(t *testing.T) skills.ScanResult {
	t.Helper()
	dir := t.TempDir()
	refDir := filepath.Join(dir, "references")
	if err := os.MkdirAll(refDir, 0o755); err != nil {
		t.Fatal(err)
	}
	aPath := filepath.Join(refDir, "a-guide.md")
	bPath := filepath.Join(refDir, "b-guide.md")
	if err := os.WriteFile(aPath, []byte("# Alpha Heading\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bPath, []byte("# Bravo Heading\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return skills.ScanResult{
		Scope: skills.Scope{Name: "Global", Path: dir},
		Skills: []skills.Skill{{
			Name: "alpha",
			Path: dir,
			References: []skills.SkillFile{
				{Name: "a-guide.md", Path: aPath},
				{Name: "b-guide.md", Path: bPath},
			},
		}},
	}
}

func TestNoTextSectionHeaders(t *testing.T) {
	var m tea.Model = NewModel(twoReferencesResult(t))
	m = sized(m, 120, 40)
	m = press(m, "3")
	m = press(m, "r")

	out := plain(view(m))
	for _, unwanted := range []string{"▸ Files", "▸ Content", "Content ("} {
		if strings.Contains(out, unwanted) {
			t.Errorf("expected sections demarcated by dividers, not the text header %q, got:\n%s", unwanted, out)
		}
	}
}

func TestSubfocusIndicatorMovesWithTab(t *testing.T) {
	var m tea.Model = NewModel(twoReferencesResult(t))
	m = sized(m, 120, 40)
	m = press(m, "3")
	m = press(m, "r")

	listView := view(m)
	m = press(m, "tab")
	contentView := view(m)

	if listView == contentView {
		t.Errorf("expected the subfocus indicator to change when toggling file list vs content")
	}
}

func TestTabTogglesSubfocusBetweenListAndContent(t *testing.T) {
	// List subfocus: j moves the selected file, switching rendered content.
	var listMode tea.Model = NewModel(twoReferencesResult(t))
	listMode = sized(listMode, 120, 40)
	listMode = press(listMode, "3")
	listMode = press(listMode, "r")
	if !strings.Contains(plain(view(listMode)), "Alpha Heading") {
		t.Fatalf("expected first reference content initially")
	}
	listMode = press(listMode, "j")
	if !strings.Contains(plain(view(listMode)), "Bravo Heading") {
		t.Errorf("expected j in list subfocus to select the second reference")
	}

	// Content subfocus: j scrolls content, leaving file selection unchanged.
	var contentMode tea.Model = NewModel(twoReferencesResult(t))
	contentMode = sized(contentMode, 120, 40)
	contentMode = press(contentMode, "3")
	contentMode = press(contentMode, "r")
	contentMode = press(contentMode, "tab")
	contentMode = press(contentMode, "j")
	if strings.Contains(plain(view(contentMode)), "Bravo Heading") {
		t.Errorf("expected j in content subfocus to leave file selection unchanged")
	}
}

func TestSkillContentHasNoLeadingBlankLine(t *testing.T) {
	m := NewModel(detailResult(t))

	lines := strings.Split(plain(m.currentContent()), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		t.Errorf("expected SKILL.md content to start on real text, got a leading blank:\n%q", m.currentContent())
	}
}

func TestScriptsAndReferencesShowNoFilesWhenEmpty(t *testing.T) {
	res := skills.ScanResult{
		Scope:  skills.Scope{Name: "Global", Path: "/root"},
		Skills: []skills.Skill{{Name: "empty", Path: "/root/empty"}},
	}
	for _, tab := range []string{"s", "r"} {
		var m tea.Model = NewModel(res)
		m = sized(m, 120, 40)
		m = press(m, "3")
		m = press(m, tab)
		if !strings.Contains(plain(view(m)), "No files") {
			t.Errorf("expected tab %q with no files to show 'No files', got:\n%s", tab, plain(view(m)))
		}
	}
}

func TestSelectedFileUsesHighlightNotCaret(t *testing.T) {
	var m tea.Model = NewModel(twoReferencesResult(t))
	m = sized(m, 120, 40)
	m = press(m, "3")
	m = press(m, "r")

	selected := lineContaining(view(m), "a-guide.md")
	if strings.Contains(plain(selected), "> ") {
		t.Errorf("expected no caret on the selected file, got %q", plain(selected))
	}

	m = press(m, "j") // move selection off a-guide
	unselected := lineContaining(view(m), "a-guide.md")
	if selected == unselected {
		t.Errorf("expected the selected file to render with a highlight, differing from unselected")
	}
}
