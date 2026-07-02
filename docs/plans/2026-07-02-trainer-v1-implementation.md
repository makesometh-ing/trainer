# Trainer v1 Implementation Plan

> **For agentic workers:** implement task-by-task. Each task should leave the app compiling and tests passing for the code it introduces.

**Goal:** Build Trainer, a Go TUI for browsing, inspecting, adding, and deleting globally installed agent skills.

**Architecture:** Trainer rebuilds transient in-memory state from `~/.agents/skills/*/SKILL.md` and `~/.agents/.skill-lock.json`. Bubble Tea owns interaction state; skill scanning, rendering, add/delete actions, and SSH key detection live in focused internal packages.

**Tech Stack:** Go 1.26.4, golangci-lint 2.12.2, Bubble Tea v2, Bubbles, Lip Gloss v2, Glamour v2, Huh v2, Chroma.

## Global Constraints

- Do not create a Trainer-owned database, cache, or persistent index.
- V1 scans only `~/.agents/skills/*/SKILL.md` and `~/.agents/.skill-lock.json`.
- V1 has one scope: `Global`.
- Add uses `npx skills add <source> -g` and does not pass agent flags.
- Delete uses `npx skills remove -g <skill-name>` for lockfile-backed skills and direct directory removal for unlocked skills.
- Use Gruvbox Dark Hard as the default theme.
- Do not depend on an external `gum` binary.
- Do not embed a true PTY in v1; suspend the TUI while interactive `npx skills` runs.
- Assets are list-only in v1 and show `No preview available`.
- Before starting the TUI, detect `node`, `npm`, and `npx`; print versions when available.
- If `npx` is unavailable, warn that adding skills is disabled and ask whether to continue.
- If `npx` is unavailable and the user continues, disable `:a` and lockfile-backed delete.
- golangci-lint 2.12.2 is the linter and must pass before work is considered complete.

---

## File Structure

- Create `go.mod`: Go module definition using `github.com/makesometh-ing/trainer`.
- Create `cmd/trainer/main.go`: application entrypoint.
- Create `internal/skills/skill.go`: skill and lockfile data types.
- Create `internal/skills/frontmatter.go`: `SKILL.md` frontmatter parsing.
- Create `internal/skills/lockfile.go`: global `.skill-lock.json` parsing.
- Create `internal/skills/scanner.go`: filesystem discovery and lockfile metadata merge.
- Create `internal/ssh/keys.go`: SSH private/public key-pair detection.
- Create `internal/actions/add.go`: add command construction and TUI suspension command.
- Create `internal/actions/delete.go`: delete strategy and command construction.
- Create `internal/runtime/dependencies.go`: startup detection for `node`, `npm`, and `npx`.
- Create `internal/render/markdown.go`: Glamour Markdown rendering.
- Create `internal/render/code.go`: Chroma syntax highlighting.
- Create `internal/app/theme.go`: Gruvbox Dark Hard theme and Lip Gloss styles.
- Create `internal/app/keymap.go`: key constants and help text.
- Create `internal/app/model.go`: Bubble Tea state model.
- Create `internal/app/update.go`: input handling and commands.
- Create `internal/app/view.go`: three-pane rendering.

---

## Task 1: Go module and core data types

**Files:**
- Create: `go.mod`
- Create: `cmd/trainer/main.go`
- Create: `internal/skills/skill.go`
- Test: `internal/skills/skill_test.go`

**Interfaces:**
- Produces:
  - `type Scope struct`
  - `type Skill struct`
  - `type SkillFile struct`
  - `type LockEntry struct`
  - `type ScanResult struct`

### Steps

- [ ] Initialize module:

```bash
go mod init github.com/makesometh-ing/trainer
```

- [ ] Add initial dependencies:

```bash
go get charm.land/bubbletea/v2 charm.land/lipgloss/v2 charm.land/glamour/v2 charm.land/huh/v2 github.com/charmbracelet/bubbles github.com/alecthomas/chroma/v2 gopkg.in/yaml.v3
```

- [ ] Create `internal/skills/skill.go` with these types:

```go
package skills

import "time"

type Scope struct {
	Name string
	Path string
}

type Skill struct {
	Name        string
	Description string
	Path        string
	SkillPath   string
	References  []SkillFile
	Scripts     []SkillFile
	Assets      []SkillFile
	Lock        *LockEntry
	Warnings    []string
}

type SkillFile struct {
	Name string
	Path string
}

type LockEntry struct {
	Source          string    `json:"source"`
	SourceType      string    `json:"sourceType"`
	SourceURL       string    `json:"sourceUrl"`
	Ref             string    `json:"ref,omitempty"`
	SkillPath       string    `json:"skillPath,omitempty"`
	SkillFolderHash string    `json:"skillFolderHash"`
	InstalledAt     time.Time `json:"installedAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
	PluginName      string    `json:"pluginName,omitempty"`
}

type ScanResult struct {
	Scope    Scope
	Skills   []Skill
	Warnings []string
}
```

- [ ] Create a minimal `cmd/trainer/main.go` that prints a placeholder until the Bubble Tea app exists.

- [ ] Run:

```bash
go test ./...
```

Expected: pass.

---

## Task 2: Frontmatter parsing

**Files:**
- Create: `internal/skills/frontmatter.go`
- Test: `internal/skills/frontmatter_test.go`

**Interfaces:**
- Produces:
  - `func ParseSkillMarkdown(content []byte) (SkillFrontmatter, string, error)`
  - `type SkillFrontmatter struct`

### Steps

- [ ] Write tests for valid frontmatter, missing frontmatter, invalid YAML, and missing name/description.

- [ ] Implement parser that reads YAML between leading `---` fences and returns Markdown body separately.

- [ ] Use `gopkg.in/yaml.v3`.

- [ ] Run:

```bash
go test ./internal/skills
```

Expected: pass.

---

## Task 3: Global lockfile parsing

**Files:**
- Create: `internal/skills/lockfile.go`
- Test: `internal/skills/lockfile_test.go`

**Interfaces:**
- Produces:
  - `func ReadGlobalLock(path string) (map[string]LockEntry, error)`
  - `func DefaultGlobalLockPath(home string) string`

### Steps

- [ ] Write tests for absent lockfile, valid lockfile, invalid JSON, and missing `skills` object.

- [ ] Implement parsing for `~/.agents/.skill-lock.json` shape:

```json
{
  "version": 3,
  "skills": {
    "skill-name": {
      "source": "owner/repo",
      "sourceType": "github",
      "sourceUrl": "git@github.com:owner/repo.git",
      "skillPath": "skills/skill-name/SKILL.md"
    }
  }
}
```

- [ ] Return an empty map for absent file.

- [ ] Return an error for invalid JSON so the UI can show a warning.

- [ ] Run:

```bash
go test ./internal/skills
```

Expected: pass.

---

## Task 4: Skill scanner

**Files:**
- Create: `internal/skills/scanner.go`
- Test: `internal/skills/scanner_test.go`

**Interfaces:**
- Produces:
  - `func ScanGlobal(root string, lockPath string) ScanResult`

### Steps

- [ ] Write tests with temporary directories containing:
  - one valid skill
  - one skill with `references/`
  - one skill with `scripts/`
  - one skill with `assets/`
  - one malformed skill
  - one lockfile-backed skill

- [ ] Implement scanning only one level below `root` for `*/SKILL.md`.

- [ ] Sort skills by name for deterministic UI.

- [ ] Merge lock metadata by skill name.

- [ ] Collect files recursively under `references`, `scripts`, and `assets`, sorted by relative path.

- [ ] Run:

```bash
go test ./internal/skills
```

Expected: pass.

---

## Task 5: SSH key-pair detection

**Files:**
- Create: `internal/ssh/keys.go`
- Test: `internal/ssh/keys_test.go`

**Interfaces:**
- Produces:
  - `func IsSSHGitSource(source string) bool`
  - `func FindKeyPairs(dir string) ([]KeyPair, error)`
  - `type KeyPair struct`

### Steps

- [ ] Test SSH Git detection for `git@github.com:owner/repo.git` and `ssh://git@host/owner/repo.git`.

- [ ] Test non-SSH sources like `owner/repo`, HTTPS URLs, and local paths.

- [ ] Test key-pair discovery with private key plus matching `.pub`.

- [ ] Exclude `known_hosts`, `config`, files ending `.pub`, and directories.

- [ ] Run:

```bash
go test ./internal/ssh
```

Expected: pass.

---

## Task 6: Runtime dependency detection

**Files:**
- Create: `internal/runtime/dependencies.go`
- Test: `internal/runtime/dependencies_test.go`

**Interfaces:**
- Produces:
  - `type DependencyStatus struct`
  - `func CheckDependencies() DependencyStatus`
  - `func ConfirmContinueWithoutNPX(in io.Reader, out io.Writer) bool`

### Steps

- [ ] Test successful detection of `node`, `npm`, and `npx` by injecting command lookup/version functions.

- [ ] Test missing `npx` marks add as unavailable.

- [ ] Test missing `node` or `npm` prints a warning.

- [ ] Test yes/no confirmation defaults to no.

- [ ] Implement dependency status with executable paths, versions, and missing dependency names.

- [ ] `cmd/trainer/main.go` must print detected versions before launching the TUI.

- [ ] If `npx` is missing, ask whether to continue before launching the TUI.

- [ ] If the user continues without `npx`, pass dependency status into the app model so `:a` is disabled and lockfile-backed delete is disabled.

- [ ] Run:

```bash
go test ./internal/runtime
```

Expected: pass.

---

## Task 7: Add and delete action planning

**Files:**
- Create: `internal/actions/add.go`
- Create: `internal/actions/delete.go`
- Test: `internal/actions/actions_test.go`

**Interfaces:**
- Produces:
  - `func AddCommand(source string, keyPath string) *exec.Cmd`
  - `func DeleteCommand(skillName string) *exec.Cmd`
  - `func DeleteStrategy(skill skills.Skill) Strategy`

### Steps

- [ ] Test add command without SSH key produces `npx skills add <source> -g`.

- [ ] Test add command with SSH key includes `GIT_SSH_COMMAND=ssh -i <key-path>`.

- [ ] Test locked skill deletion uses `npx skills remove -g <skill-name>`.

- [ ] Test unlocked skill deletion selects filesystem removal.

- [ ] Keep command construction separate from execution so UI tests do not run `npx`.

- [ ] Run:

```bash
go test ./internal/actions
```

Expected: pass.

---

## Task 8: Markdown and code rendering

**Files:**
- Create: `internal/render/markdown.go`
- Create: `internal/render/code.go`
- Test: `internal/render/render_test.go`

**Interfaces:**
- Produces:
  - `func Markdown(content string, width int) (string, error)`
  - `func Code(content string, filename string) (string, error)`

### Steps

- [ ] Implement Markdown rendering with Glamour word wrap.

- [ ] Implement code rendering with Chroma extension detection.

- [ ] Fall back to plain text for unknown script extensions.

- [ ] Run:

```bash
go test ./internal/render
```

Expected: pass.

---

## Task 9: Theme and app model

**Files:**
- Create: `internal/app/theme.go`
- Create: `internal/app/model.go`
- Create: `internal/app/keymap.go`
- Test: `internal/app/model_test.go`

**Interfaces:**
- Produces:
  - `func GruvboxDarkHard() Theme`
  - `func NewModel(result skills.ScanResult) Model`

### Steps

- [ ] Define Gruvbox Dark Hard colors:

```text
bg0 hard  #1d2021
bg0       #282828
bg1       #3c3836
bg2       #504945
bg3       #665c54
bg4       #7c6f64
fg1       #ebdbb2
fg2       #d5c4a1
fg3       #bdae93
fg4       #a89984
gray      #928374
red       #fb4934
green     #b8bb26
yellow    #fabd2f
blue      #83a598
purple    #d3869b
aqua      #8ec07c
orange    #fe8019
```

- [ ] Model focus panes: scope, skills, detail.

- [ ] Model detail tabs: skill, references, scripts, assets.

- [ ] Model detail subfocus: file list or content.

- [ ] Track selected skill index, selected file index per tab, viewport offsets, width, and height.

- [ ] Run:

```bash
go test ./internal/app
```

Expected: pass.

---

## Task 10: Bubble Tea update logic

**Files:**
- Create: `internal/app/update.go`
- Modify: `internal/app/model.go`
- Test: `internal/app/update_test.go`

**Interfaces:**
- Produces:
  - `func (m Model) Init() tea.Cmd`
  - `func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)`

### Steps

- [ ] Implement global pane focus keys `1`, `2`, `3`.

- [ ] Implement skill list `j/k`, `h/l`.

- [ ] Implement detail tab keys `a/b/c/d`.

- [ ] Implement detail `tab` subfocus toggle.

- [ ] Implement detail scroll keys `j/k`, `ctrl+d/u`, `ctrl+f/b`, `gg/G`.

- [ ] Implement `/` help modal state.

- [ ] Implement `:` command palette state with `a`, `d`, and `esc`.

- [ ] Run:

```bash
go test ./internal/app
```

Expected: pass.

---

## Task 11: Three-pane view rendering

**Files:**
- Create: `internal/app/view.go`
- Modify: `internal/app/theme.go`
- Test: `internal/app/view_test.go`

**Interfaces:**
- Produces:
  - `func (m Model) View() string`

### Steps

- [ ] Render left scope pane with only `Global`.

- [ ] Render middle skill list with name and dim metadata line.

- [ ] Render detail header with name, source, sourceUrl, skillPath, and local path.

- [ ] Render tabs in order: `SKILL.md`, `References`, `Scripts`, `Assets`.

- [ ] Render vertical file list above content for References/Scripts/Assets.

- [ ] Render asset content as `No preview available`.

- [ ] Render help modal for `/`.

- [ ] Render command palette for `:`.

- [ ] Run:

```bash
go test ./internal/app
```

Expected: pass.

---

## Task 12: Wire runtime actions and refresh

**Files:**
- Modify: `internal/app/update.go`
- Modify: `cmd/trainer/main.go`
- Test: `internal/app/actions_test.go`

**Interfaces:**
- Consumes scanner and actions packages.
- Produces refresh behavior after add/delete.

### Steps

- [ ] On startup, resolve home directory and scan `~/.agents/skills` plus `~/.agents/.skill-lock.json`.

- [ ] For add flow, collect source input and optional SSH key selection.

- [ ] Suspend Bubble Tea and run the interactive add command.

- [ ] On command exit, rescan and refresh model.

- [ ] For delete flow, confirm deletion.

- [ ] Run locked delete through `npx skills remove -g <skill-name>`.

- [ ] Run unlocked delete through direct directory removal.

- [ ] On delete exit, rescan and refresh model.

- [ ] Run:

```bash
go test ./...
```

Expected: pass.

---

## Task 13: Final verification

**Files:**
- All source files.

### Steps

- [ ] Format code:

```bash
gofmt -w cmd internal
```

- [ ] Run tests:

```bash
go test ./...
```

- [ ] Run vet:

```bash
go vet ./...
```

- [ ] Run lint:

```bash
golangci-lint run
```

- [ ] Smoke test manually:

```bash
go run ./cmd/trainer
```

Expected:

- TUI opens.
- Left pane shows `Global`.
- Middle pane shows skills from `~/.agents/skills`.
- Detail pane updates as selection changes.
- `a/b/c/d` switch tabs.
- `/` opens shortcuts.
- `:` opens command palette.
