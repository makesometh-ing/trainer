package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/makesometh-ing/trainer/internal/skills"
)

func globalAgents(names ...string) skills.ScanResult {
	r := skills.ScanResult{Scope: skills.Scope{Name: ".agents", Section: skills.SectionGlobal, Path: "/home/.agents/skills"}}
	for _, n := range names {
		r.Skills = append(r.Skills, skills.Skill{Name: n, Path: "/home/.agents/skills/" + n})
	}
	return r
}

func globalClaude(names ...string) skills.ScanResult {
	r := skills.ScanResult{Scope: skills.Scope{Name: "claude", Section: skills.SectionGlobal, Path: "/home/.claude/skills"}}
	for _, n := range names {
		r.Skills = append(r.Skills, skills.Skill{Name: n, Path: "/home/.claude/skills/" + n})
	}
	return r
}

func projectAgents(names ...string) skills.ScanResult {
	r := skills.ScanResult{Scope: skills.Scope{Name: ".agents", Section: skills.SectionProject, Path: "/cwd/.agents/skills"}}
	for _, n := range names {
		r.Skills = append(r.Skills, skills.Skill{Name: n, Path: "/cwd/.agents/skills/" + n})
	}
	return r
}

func TestProjectSectionShownOnlyWithProjectScope(t *testing.T) {
	withProject := []skills.ScanResult{globalAgents("alpha"), projectAgents("proj")}
	var mp tea.Model = NewModel(withProject)
	mp = sized(mp, 120, 30)
	if !strings.Contains(plain(view(mp)), "Project") {
		t.Errorf("expected the Project section when a project scope has skills, got:\n%s", plain(view(mp)))
	}

	globalOnly := []skills.ScanResult{globalAgents("alpha")}
	var mg tea.Model = NewModel(globalOnly)
	mg = sized(mg, 120, 30)
	if strings.Contains(plain(view(mg)), "Project") {
		t.Errorf("expected no Project section when there are no project scopes, got:\n%s", plain(view(mg)))
	}
}

func TestScopeNavigationSwitchesSkillList(t *testing.T) {
	results := []skills.ScanResult{
		globalAgents("alpha", "bravo"),
		globalClaude("gamma"),
	}
	var m tea.Model = NewModel(results)
	m = sized(m, 120, 30)

	// Focus the Scope pane, then move down to the second scope.
	m = press(m, "1")
	m = press(m, "j")

	out := plain(view(m))
	if !strings.Contains(out, "gamma") {
		t.Errorf("expected the second scope's skill gamma after navigating, got:\n%s", out)
	}
	if strings.Contains(out, "bravo") {
		t.Errorf("did not expect the first scope's skills after switching scope, got:\n%s", out)
	}
}

func TestScopeNavigationInertWithSingleScope(t *testing.T) {
	var m tea.Model = NewModel([]skills.ScanResult{globalAgents("alpha", "bravo")})
	m = sized(m, 120, 30)
	m = press(m, "1")
	m = press(m, "j")

	if m.(Model).selectedScope != 0 {
		t.Errorf("expected scope selection to stay put with a single scope, got %d", m.(Model).selectedScope)
	}
}

func TestZeroScopesRendersWithoutPanic(t *testing.T) {
	var m tea.Model = NewModel(nil)
	m = sized(m, 120, 30)
	out := plain(view(m))
	if !strings.Contains(out, "No skill selected") {
		t.Errorf("expected an empty-state detail message with no scopes, got:\n%s", out)
	}
	// Navigation keys must not panic when there is nothing to select.
	m = press(m, "1")
	m = press(m, "j")
	m = press(m, "2")
	m = press(m, "j")
	_ = view(m)
}

func TestHarnessScopeListsSkillsLocal(t *testing.T) {
	// .agents lists "shared" with a source; claude carries the same skill with no
	// lock, so it must read as local when the claude scope is selected.
	agents := globalAgents("shared")
	agents.Skills[0].Lock = &skills.LockEntry{Source: "owner/shared"}
	claude := globalClaude("shared")

	var m tea.Model = NewModel([]skills.ScanResult{agents, claude})
	m = sized(m, 120, 30)

	m = press(m, "1")
	m = press(m, "j") // select the claude harness scope

	out := plain(view(m))
	if !strings.Contains(out, "local") {
		t.Errorf("expected the harness-scope skill to read local, got:\n%s", out)
	}
	if strings.Contains(out, "owner/shared") {
		t.Errorf("harness scope must not show the .agents lock source, got:\n%s", out)
	}
}

func TestScopePaneShowsSectionRowsAndCounts(t *testing.T) {
	results := []skills.ScanResult{
		globalAgents("alpha", "bravo"),
		globalClaude("gamma"),
	}
	var m tea.Model = NewModel(results)
	m = sized(m, 120, 30)

	out := plain(view(m))

	if !strings.Contains(out, "Global") {
		t.Error("expected the Global section header")
	}
	if !strings.Contains(out, ".agents") || !strings.Contains(out, "claude") {
		t.Errorf("expected both scope labels, got:\n%s", out)
	}
	// The .agents scope has two skills; its row shows the count.
	agentsRow := lineContaining(out, ".agents")
	if !strings.Contains(agentsRow, "2") {
		t.Errorf("expected the .agents row to show its skill count 2, got %q", agentsRow)
	}
	// First scope is selected, so its skills are listed.
	if !strings.Contains(out, "alpha") || !strings.Contains(out, "bravo") {
		t.Errorf("expected the first scope's skills, got:\n%s", out)
	}
}
