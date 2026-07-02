package app

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/makesometh-ing/trainer/internal/skills"
)

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
	m := press(NewModel(detailResult(t)), "i")

	out := plain(view(m))
	if !strings.Contains(out, "Alpha Overview") {
		t.Errorf("expected SKILL.md tab to render the skill body, got %q", out)
	}
}

func TestSkillTabHidesFrontmatter(t *testing.T) {
	m := press(NewModel(detailResult(t)), "i")

	out := plain(view(m))
	if strings.Contains(out, "name: alpha") {
		t.Errorf("expected SKILL.md tab to hide raw frontmatter, got %q", out)
	}
	if strings.Contains(out, "description: First skill") {
		t.Errorf("expected SKILL.md tab to hide raw frontmatter, got %q", out)
	}
}

func TestReferencesTabShowsFileList(t *testing.T) {
	m := press(NewModel(detailResult(t)), "r")

	out := view(m)
	if !strings.Contains(out, "guide.md") {
		t.Errorf("expected References tab to list reference filenames, got %q", out)
	}
}

func TestSelectingReferenceRendersMarkdown(t *testing.T) {
	m := press(NewModel(detailResult(t)), "r")

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
	m := press(NewModel(scriptOnlyResult(t, "run.go", "package main\n\nfunc main() {}\n")), "s")

	out := plain(view(m))
	if !strings.Contains(out, "func main") {
		t.Errorf("expected Scripts tab to show highlighted script source, got %q", out)
	}
}

func TestUnknownScriptExtensionFallsBackToPlainText(t *testing.T) {
	m := press(NewModel(scriptOnlyResult(t, "notes.weirdext", "raw plain content")), "s")

	out := plain(view(m))
	if !strings.Contains(out, "raw plain content") {
		t.Errorf("expected unknown extension to show raw source, got %q", out)
	}
}

func TestAssetsTabShowsNoPreview(t *testing.T) {
	m := press(NewModel(detailResult(t)), "a")

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

func TestContentShowsScrollIndicator(t *testing.T) {
	var m tea.Model = NewModel(longScriptResult(t))
	m = sized(m, 120, 24)
	m = press(m, "s")
	m = press(m, "tab")

	before := lineContaining(view(m), "%")
	if before == "" {
		t.Fatalf("expected a scroll percentage indicator on long content, got none")
	}

	m = press(m, "ctrl+d")
	after := lineContaining(view(m), "%")
	if after == before {
		t.Errorf("expected scroll indicator to change after scrolling, got %q both times", before)
	}
}

func TestContentScrollKeysMoveViewport(t *testing.T) {
	var m tea.Model = NewModel(longScriptResult(t))
	m = sized(m, 120, 24)
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

func TestSubfocusIndicatorShowsActiveSection(t *testing.T) {
	var m tea.Model = NewModel(twoReferencesResult(t))
	m = sized(m, 120, 40)
	m = press(m, "3")
	m = press(m, "r")

	listActive := lineContaining(view(m), "Files")
	if !strings.Contains(listActive, "▸") {
		t.Errorf("expected Files section marked active by default, got %q", listActive)
	}

	m = press(m, "tab")
	contentActive := lineContaining(view(m), "Content")
	if !strings.Contains(contentActive, "▸") {
		t.Errorf("expected Content section marked active after tab, got %q", contentActive)
	}
	filesNowInactive := lineContaining(view(m), "Files")
	if strings.Contains(filesNowInactive, "▸") {
		t.Errorf("expected Files section no longer active after tab, got %q", filesNowInactive)
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
