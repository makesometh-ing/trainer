package skills

import (
	"path/filepath"
	"sort"
)

// ScopeDef is one entry in the harness registry: a skills location Trainer
// scans. Dir and Lock are paths relative to the section's base (the user's home
// for Global, the launch directory for Project). Lock is empty for scopes that
// have no lock (every harness scope); only .agents scopes name a lock. Dir and
// Lock are separate fields, not derived from each other: the .agents lock sits
// beside skills/, but the project lock is at the launch-directory root.
type ScopeDef struct {
	Label   string
	Section Section
	Dir     string
	Lock    string
}

// Registry is the extensible list of scopes the scanner, scope pane, and
// actions all iterate. Adding a harness is a single appended entry.
//
// Agents whose project skills live under .agents/skills (codex, opencode,
// cursor) share the .agents project scope, so the Project section lists only
// .agents, claude, and pi.
func Registry() []ScopeDef {
	return []ScopeDef{
		{Label: ".agents", Section: SectionGlobal, Dir: filepath.Join(".agents", "skills"), Lock: filepath.Join(".agents", ".skill-lock.json")},
		{Label: "claude", Section: SectionGlobal, Dir: filepath.Join(".claude", "skills")},
		{Label: "codex", Section: SectionGlobal, Dir: filepath.Join(".codex", "skills")},
		{Label: "opencode", Section: SectionGlobal, Dir: filepath.Join(".config", "opencode", "skills")},
		{Label: "pi", Section: SectionGlobal, Dir: filepath.Join(".pi", "agent", "skills")},
		{Label: "cursor", Section: SectionGlobal, Dir: filepath.Join(".cursor", "skills")},
		{Label: ".agents", Section: SectionProject, Dir: filepath.Join(".agents", "skills"), Lock: "skills-lock.json"},
		{Label: "claude", Section: SectionProject, Dir: filepath.Join(".claude", "skills")},
		{Label: "pi", Section: SectionProject, Dir: filepath.Join(".pi", "skills")},
	}
}

// ScanAll scans every scope in the registry, resolving Global scopes under home
// and Project scopes under cwd. It returns one result per scope that has at
// least one skill, tagged with the scope's section, label, and resolved path;
// empty or absent scopes are omitted. The registry order is preserved.
func ScanAll(home, cwd string) []ScanResult {
	var results []ScanResult
	for _, def := range Registry() {
		base := home
		if def.Section == SectionProject {
			base = cwd
		}
		dir := filepath.Join(base, def.Dir)
		lockPath := ""
		if def.Lock != "" {
			lockPath = filepath.Join(base, def.Lock)
		}

		result := Scan(dir, lockPath)
		if len(result.Skills) == 0 {
			continue
		}
		result.Scope = Scope{Name: def.Label, Section: def.Section, Path: dir}
		sort.Slice(result.Skills, func(i, j int) bool {
			return result.Skills[i].Name < result.Skills[j].Name
		})
		results = append(results, result)
	}
	return results
}
