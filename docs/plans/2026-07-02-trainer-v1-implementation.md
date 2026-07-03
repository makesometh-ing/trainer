# Trainer v1 Implementation Plan (vertical slices)

> **For agentic workers:** implement slice-by-slice using TDD. Each slice is a
> vertical tracer bullet: it cuts end-to-end (disk to logic to UI) and is proven
> by **integration tests over real temp-dir fixtures**, not unit tests of
> internal shapes. Do RED then GREEN one behavior at a time; never write all
> tests up front. Each slice must leave the app compiling and tests passing.

**Goal:** Build Trainer, a Go TUI for browsing, inspecting, adding, and deleting agent skills across global and project scopes.

**Architecture:** Trainer rebuilds transient in-memory state by scanning every scope in the harness registry (global home locations and project locations under the launch directory), reading each scope's optional lock. Bubble Tea owns interaction state; skill scanning, rendering, add/delete actions, and SSH key detection live in focused internal packages behind small interfaces.

**Tech Stack:** Go 1.26.4, golangci-lint 2.12.2, Bubble Tea v2, Bubbles, Lip Gloss v2, Glamour v2, Huh v2, Chroma.

---

## Why this plan is sliced vertically

The original plan was horizontal: all data types, then all parsing, then all scanning, then the whole TUI (Tasks 1-13). That produces tautological tests — e.g. testing frontmatter parsing in isolation only proves "YAML parses into a struct", which is coupled to the parser's shape and gives zero confidence about user-facing behavior.

This rewrite re-cuts the remaining work into **vertical slices**, each provable end-to-end. Internal parsers (frontmatter, lockfile) become collaborators of the scanner and are exercised *through* `ScanGlobal`, never shape-tested directly.

## Testing philosophy for this plan

- **Integration tests are the gold standard.** Prefer tests that drive the public entrypoint of a package (`ScanGlobal`, `Model.Update`, `Model.View`) against real temp-dir fixtures. Assert on user-observable output, not internal struct fields.
- **Unit tests are secondary** — only for genuinely tricky pure logic (e.g. SSH URL classification, delete-strategy selection) where an integration test would be awkward.
- **No tautological tests.** Expected values come from hand-authored fixtures / the spec, never recomputed the way the code computes them.
- **Internal parsers are collaborators, not test targets.** `frontmatter` and `lockfile` parsing are exercised *through* `ScanGlobal`. We do not write dedicated shape-tests for them; parsing edge cases are expressed as scanner fixtures producing an observable warning or metadata value.

## Global Constraints

- Do not create a Trainer-owned database, cache, or persistent index.
- Scan every scope in the harness registry: global home locations and project locations under the launch directory (`.agents`, `claude`, `codex`, `opencode`, `pi`, `cursor`). Adding a harness is one new registry entry. Only `.agents` scopes read a lock; harness scopes are always `local`.
- Add uses `npx skills add <source>` (no scope flag) and does not pass agent flags; interactive npx prompts for skills, agents, and Project/Global scope.
- Delete uses `npx skills remove <skill-name>` for lock-listed skills (npx prompts scope) and direct directory removal for skills only on disk.
- Use Gruvbox Dark Hard as the default theme.
- Do not depend on an external `gum` binary.
- Do not embed a true PTY in v1; suspend the TUI while interactive `npx skills` runs.
- Assets are list-only in v1 and show `No preview available`.
- Before starting the TUI, detect `node`, `npm`, and `npx`; print versions when available.
- If `npx` is unavailable, warn that adding skills is disabled and ask whether to continue.
- If `npx` is unavailable and the user continues, disable `:a` and lockfile-backed delete.
- golangci-lint 2.12.2 is the linter and must pass before work is considered complete.

## Target file structure

Files land as their owning slice needs them rather than all up front.

```
cmd/trainer/main.go
internal/skills/{skill,frontmatter,lockfile,scanner,harness}.go
internal/ssh/keys.go
internal/actions/{add,delete}.go
internal/runtime/dependencies.go
internal/render/{markdown,code}.go
internal/app/{theme,keymap,model,update,view,scope}.go
```

---

## Slice map

| Slice | User-observable behavior | Absorbs old tasks | Status |
|-------|--------------------------|-------------------|--------|
| Foundation | Module + core data types | 1 | DONE |
| 1 (tracer) | Browse: scope pane, skill list, `j/k` selection, detail header, `q` quit | 2, 3, 4, browse parts of 9/10/11 | DONE |
| 2 | Inspect: tabs `a/b/c/d`, file lists, Glamour/Chroma render, asset no-preview | 8, detail parts of 9/10/11 | DONE |
| 3 | Startup dependency check + continue prompt, disable `:a` when no `npx` | 6, part of 12 | DONE |
| 4 | Add: `:a` wizard, SSH key select, suspend + run, refresh | 5, 7(add), part of 12 | DONE |
| 5 | Delete: `:d` confirm, lockfile vs on-disk strategy, refresh | 7(delete), part of 12 | DONE |
| 6 | Layout polish: shortcut labels on panes/tabs, full-screen reflow, too-small message | new | DONE |
| 7 | Detail rendering & layout polish (post-smoke): frontmatter strip, Gruvbox Glamour theme, pane fit/windowing, row highlight, `h`/`l`/`enter` nav, subfocus + scroll indicators, tab keys `i/r/s/a` | new | DONE |
| 8 | Search + filter + demarcated Details: search box, origin filter, full-width Details, frontmatter shown, content scrollbar, `?` help modal, `:u` update, no row caret, `Details` title | new | DONE |
| 9 | Rebuild the add-skill wizard on Huh v2 (remove the hand-rolled wizard) | new | DONE |
| Final | Full verification + manual smoke | 13 | DONE |
| 10 | Multi-scope browsing: harness registry, symlink-aware scanner, two-level Scope pane (Global / Project sections) + counts, per-scope lock, scope-scoped list, hide empty scopes/sections | new (v1.0) | DONE |
| 11 | Scope-aware actions: add passes no scope flag (npx prompts scope); delete passes `--global` by the skill's scope; rescan all scopes | new (v1.0) | DONE |
| 12 | Context footer + palette-dimmed commands: permanent bottom keybind bar per context; drop the persistent status line and the pre-TUI npx prompt; dim npx-only commands in the palette with a `disabled without npx` tag | new (v1.0) | TODO |

---

## Foundation (DONE): Go module and core data types

**Status:** Completed 2026-07-02. Verified with `make fmt-check vet test lint` (lint: 0 issues). Was "Task 1" in the horizontal plan.

**Files created:** `go.mod`, `cmd/trainer/main.go` (placeholder), `internal/skills/skill.go`, `Makefile`.

**Types produced:** `Scope`, `Skill`, `SkillFile`, `LockEntry`, `ScanResult` (see `internal/skills/skill.go`).

### Caveats / gotchas carried forward

- **No `skill_test.go`** — pure data types; a shape test would be tautological. First real integration test lands in Slice 1 (`ScanGlobal`). User approved.
- **`go.mod` is minimal.** `go mod tidy` dropped charm/chroma/yaml deps because nothing imports them yet. They re-resolve at latest as each slice adds imports. Don't be alarmed it's near-empty now.
- **Makefile is the entrypoint** for all maintenance tasks. Always run `make lint` as part of verification (standing instruction). `make verify` runs `fmt-check vet test lint`.
- Dep versions last in use: bubbletea/v2 v2.0.7, lipgloss/v2 v2.0.4, glamour/v2 v2.0.1, huh/v2 v2.0.3, bubbles v1.0.0, chroma/v2 v2.27.0, yaml.v3 v3.0.1.

---

## Slice 1 (TRACER BULLET): Browse installed skills — DONE

**Status:** Completed 2026-07-02. Verified with `make fmt-check vet test lint` (lint: 0 issues) and a smoke run of `ScanGlobal` against the real `~/.agents/skills` (37 skills discovered, sorted, lock metadata merged — `karpathy-guidelines` has no lockfile entry so no source is shown, hindsight-docs 84 refs / impeccable 67 scripts collected, no warnings).

**User-observable behavior:** Running `trainer` against a skill directory shows the `Global` scope pane and a list of discovered skills. A row shows the skill's `source` when the skill is present in the lockfile, otherwise it shows the local path. `j`/`k` move the selection; the detail pane header updates to show the selected skill's metadata (name, source, sourceUrl, skillPath, local path). `q` quits.

This slice proves the entire spine: **disk to scan to model to update to view**. It absorbs old Tasks 2, 3, 4 (as internal collaborators of the scanner) plus the browse-only portions of 9, 10, 11.

**Files:**
- Create: `internal/skills/frontmatter.go` (internal collaborator — no dedicated test)
- Create: `internal/skills/lockfile.go` (internal collaborator — no dedicated test)
- Create: `internal/skills/scanner.go` — `func ScanGlobal(root, lockPath string) ScanResult`
- Test: `internal/skills/scanner_test.go` — **integration test over temp-dir fixtures**
- Create: `internal/app/{theme,keymap,model,update,view}.go` (browse subset only)
- Test: `internal/app/browse_test.go` — drives `NewModel(ScanResult)` then `Update(key)` then `View()`
- Modify: `cmd/trainer/main.go` — resolve home, scan, launch Bubble Tea program

**Interfaces produced:**
- `func ScanGlobal(root string, lockPath string) ScanResult`
- `func GruvboxDarkHard() Theme`
- `func NewModel(result skills.ScanResult) Model`
- `Model.Init/Update/View`

### RED then GREEN order (one behavior per cycle)

1. **Scanner: one valid skill is discovered.** Fixture: temp dir with `skill-a/SKILL.md` containing valid frontmatter. Assert `ScanGlobal` returns one skill, name `skill-a`, description from frontmatter, correct `Path`. (Drives frontmatter parsing into existence as a collaborator.)
2. **Scanner: skills sorted by name, only one level deep.** Fixture: `skill-b`, `skill-a`, plus nested `skill-a/sub/SKILL.md` that must be ignored. Assert order `[skill-a, skill-b]` and nested ignored.
3. **Scanner: reference/script/asset files collected recursively, sorted by relative path.** Fixture: skill with `references/`, `scripts/`, `assets/` (incl. a nested file). Assert the three slices populate and sort.
4. **Scanner: lockfile metadata merged by name.** Fixture: two skills, lockfile covering one (use the real lockfile shape from `~/.agents/.skill-lock.json`). Assert the skill in the lockfile has `Lock.Source`, the other has `Lock == nil`.
5. **Scanner: malformed skill still listed with a warning.** Fixture: `SKILL.md` with no frontmatter / bad YAML. Assert the skill appears with a non-empty `Warnings` entry (proves parser edge cases via observable behavior, not a shape test).
6. **Scanner: absent lockfile means no error, all `Lock == nil`.** Fixture: no lockfile path. Assert skills present, all `Lock == nil`, no scan-level fatal.
7. **Model: initial view renders Global scope + skill list + first skill selected in detail header.** Drive `NewModel(result).View()`; assert output contains `Global`, each skill name, and the first skill's source metadata.
8. **Model: `j`/`k` move selection and update detail header.** Send key msgs through `Update`; assert `View()` header reflects the newly selected skill.
9. **Model: row shows source or local path.** Assert a skill in the lockfile shows its `source`; a skill only on disk shows its local path.
10. **Model: `q` quits.** Assert `Update` returns `tea.Quit`.

### Verification
```bash
make fmt-check vet test lint
go run ./cmd/trainer   # manual smoke: list appears, j/k works, q quits
```

### Notes / open questions
- Capture a small real slice of `~/.agents/.skill-lock.json` into the test fixture so lock parsing is grounded in the actual `npx skills` schema (version 3, `skills` map, optional `ref`).
- Confirm Bubble Tea v2 (v2.0.7) `Update` signature and key-msg types against the installed API before writing update tests — v2 changed several signatures from v1.
- `DefaultGlobalLockPath(home)` and `DefaultSkillsRoot(home)` helpers live in `internal/skills` and are used by `main.go`; they are exercised indirectly by the scanner tests via explicit paths.
- View tests assert on substrings, not full-frame snapshots, to avoid brittleness against Lip Gloss styling/width. This keeps tests behavior-focused and refactor-resilient.

### Caveats / gotchas discovered during Slice 1 (READ before Slice 2)

- **Bubble Tea v2 API differs sharply from v1 and tutorials:**
  - `Model.Update` returns `(tea.Model, tea.Cmd)` (interface, not concrete) — tests reassigning the model must type the variable as `tea.Model`.
  - `Model.View()` returns a `tea.View` **struct**, not a string. Build it with `tea.NewView(content)`; the rendered text is `view.Content`.
  - **Alt screen is a field on the `View` struct** (`v.AltScreen = true`), NOT a program option. `tea.WithAltScreen()` does not exist in v2.
  - Keys arrive as `tea.KeyPressMsg` (a `tea.Key`); match with `.String()`. In tests, construct `tea.KeyPressMsg{Text: "j"}`.
  - `tea.Quit` is a `tea.Cmd`; the emitted message is `tea.QuitMsg{}`.
- **Lip Gloss v2 `Color` is a function** returning `image/color.Color`, not a type. Theme struct fields are typed `color.Color` and populated via `lipgloss.Color("#rrggbb")`.
- **Charm modules use the `charm.land/...` path** (e.g. `charm.land/bubbletea/v2`), and the v2 module must be fetched with `go mod download charm.land/bubbletea/v2` — the bare `charm.land/bubbletea` cache dir holds v1.3.10.
- **Frontmatter/lockfile are internal collaborators** with no dedicated tests, exactly as planned. `ParseSkillMarkdown` and `ReadGlobalLock` are only exercised through `ScanGlobal`. If Slice 2 or later needs `ParseSkillMarkdown`'s body output (it returns `(SkillFrontmatter, body string, error)`), it is already implemented and returns the Markdown body below the frontmatter.
- **Scanner ignores nested `SKILL.md`** (only one level below root) and continues past malformed skills, attaching a `Warnings` entry rather than dropping them.
- Dep versions now resolved: added `charm.land/bubbletea/v2 v2.0.7`, `charm.land/lipgloss/v2 v2.0.4`, `gopkg.in/yaml.v3 v3.0.1` and their transitive deps. glamour/huh/chroma/bubbles are still NOT imported yet (arrive in Slices 2/4).

### Files created in Slice 1
- `internal/skills/frontmatter.go`, `internal/skills/lockfile.go`, `internal/skills/scanner.go`
- `internal/skills/scanner_test.go` (integration tests over temp-dir fixtures)
- `internal/app/theme.go`, `internal/app/keymap.go`, `internal/app/model.go`, `internal/app/update.go`, `internal/app/view.go`
- `internal/app/browse_test.go`
- Modified `cmd/trainer/main.go` to scan and launch the program.

---

## Slice 2: Inspect skill content — DONE

**Status:** Completed 2026-07-02. Verified with `make verify` (fmt-check, vet, test, lint: 0 issues) and `go build ./...`.

**User-observable behavior:** In the detail pane, `a/b/c/d` switch tabs (`SKILL.md`, References, Scripts, Assets). References/Scripts/Assets show a file list above content; `tab` toggles subfocus between list and content; scroll keys (`j/k`, `ctrl+d/u`, `ctrl+f/b`, `gg/G`) move content. `SKILL.md` and markdown references render via Glamour; scripts highlight via Chroma (plain-text fallback for unknown extensions); assets show `No preview available`.

Absorbs old Task 8 and the remaining detail-pane portions of 9/10/11.

**Files:**
- Create: `internal/render/markdown.go` — `func Markdown(content string, width int) (string, error)`
- Create: `internal/render/code.go` — `func Code(content string, filename string) (string, error)`
- Test: `internal/render/render_test.go` (integration over sample content)
- Modify: `internal/app/{model,update,view}.go` — detail tabs, subfocus, viewport scroll
- Test: `internal/app/detail_test.go` — drive tab/scroll keys through `Update`, assert `View()`

### RED then GREEN order

1. **Detail: `b` shows References tab with a file list.** Model built from a skill with references; press `b`; assert `View()` lists the reference filenames.
2. **Detail: selecting a reference renders its markdown content.** `tab` to file list, `j` to a `.md` reference, assert rendered content substring appears (known heading in fixture).
3. **Detail: `c` shows Scripts tab; selecting a script highlights code.** Assert a known token from the script fixture appears (Chroma output still contains source text).
4. **Detail: unknown script extension falls back to plain text.** Fixture script with `.weirdext`; assert content equals raw source.
5. **Detail: `d` shows Assets tab, list-only, content is `No preview available`.**
6. **Detail: content scroll keys move the viewport.** Long markdown; `ctrl+d` then assert first-visible line changed; `gg` returns to top.
7. **Detail: `tab` toggles subfocus so `j/k` move file selection vs content.** Assert the two modes behave differently.

### Notes / open questions
- Glamour/Chroma output includes ANSI escapes; assert on plain substrings that survive rendering, not exact formatting.
- Prefer a Bubbles `viewport.Model` for scrolling (per stack); confirm v2 compatibility.

### Caveats / gotchas discovered during Slice 2 (READ before Slice 3+)

- **Scrolling uses the Bubbles `viewport.Model` (`charm.land/bubbles/v2 v2.1.0`), as the plan required.** The model holds a `viewport.Model` in `content`; scroll keys delegate to its pager methods — `ctrl+d`→`HalfPageDown()`, `ctrl+u`→`HalfPageUp()`, `ctrl+f`→`PageDown()`, `ctrl+b`→`PageUp()`, `gg`→`GotoTop()`, `G`→`GotoBottom()`, and `j/k` in content subfocus → `ScrollDown/Up`. Content is pushed in via `SetContent`; sizing via `SetWidth/SetHeight`. No hand-rolled offset math.
- **`syncContent` / `syncSize` keep the viewport in step with model state.** `syncContent` (called on skill change, file change, tab change) resets content and `GotoTop`. `syncSize` (called on `WindowSizeMsg`) sets width/height then content. `contentHeight()` = `m.height - detailChromeHeight` (chrome constant = 10, floor 3) and `defaultContentHeight = 20`/`defaultContentWidth = 80` when no size has arrived yet. Slice 6 (layout polish) should replace the magic `detailChromeHeight` with real measured pane geometry.
- **Chroma `gruvbox` style + `terminal256` formatter** chosen to match the Gruvbox theme. `render.Code` returns raw source unchanged when `lexers.Match` is nil or `lexers.Fallback` (unknown extension), which is what the plain-text-fallback test asserts.
- **`a` (SKILL.md) tab renders the on-disk `SkillPath` body via Glamour**, not the in-memory `Description`. This was added beyond the plan's 7-step order to satisfy the slice's stated "SKILL.md renders via Glamour" behavior; it reuses `renderReferenceContent` (markdown path). The scanner already returns the body from `ParseSkillMarkdown`, but the view re-reads the file by path for consistency with references/scripts.
- **Subfocus model:** `subfocusList` (default) vs `subfocusContent`, toggled by `tab`. `j/k` route through `moveContent`: on a file tab in list subfocus they move `fileSel` (and re-sync content to top); in content subfocus they scroll the viewport; on the SKILL.md tab they always scroll; otherwise they move the skill selection (browse). Switching tabs (`setTab`) resets `fileSel`, subfocus, and re-syncs content.
- **Deps added:** `charm.land/glamour/v2 v2.0.1`, `charm.land/bubbles/v2 v2.1.0` (latest stable — v2.0.0 and v2.1.0 are the only non-pre-release v2 tags), `github.com/alecthomas/chroma/v2 v2.27.0` (+ transitive goldmark/bluemonday/etc). glamour v2 API: `glamour.NewTermRenderer(glamour.WithStandardStyle("dark"), glamour.WithWordWrap(width))` then `r.Render(content)`.

---

## Slice 3: Startup dependency check — DONE

**Status:** Completed 2026-07-02. Verified with `make verify` (fmt-check, vet, test, lint: 0 issues) and a smoke build of `cmd/trainer` (prints `node 26.4.0`, `npm 11.17.0`, `npx 11.17.0` — `v`-prefixed versions normalized).

**User-observable behavior:** Before the TUI opens, Trainer prints detected `node`/`npm`/`npx` versions. If `npx` is missing, it warns that adding is disabled and prompts `Continue? [y/N]`; declining exits before the TUI. If the user continues without `npx`, `:a` is disabled with an explanatory message and lockfile-backed delete is disabled.

### Implementation decisions & handoff notes for Slice 3 (READ before Slice 4+)

> Decisions I made that were NOT dictated by the spec/plan, and must be reviewed:
> (1) `NewModel` uses functional options defaulting to `true` (vs a struct arg).
> (2) A **minimal command palette** was built to test cycle 5 through the real key path — **enabled `:a` and `:d` currently do nothing but close the palette**; real wizards are Slices 4/5.
> (3) `printDependencies` prints `<name> not found` for missing tools (not in spec).
> (4) Disabled-add status wording `"Adding skills is disabled: npx is not available."` is my phrasing.

- **Detection is injectable.** `runtime.Check(look LookPathFunc, version VersionFunc)` takes the PATH lookup and version-reader as function values, so tests never shell out. `runtime.CheckDefault()` wires the real `SystemLookPath` (`exec.LookPath`) and `SystemVersion` (`exec.Command(name, "--version")`) for `main.go`.
- **`DependencyStatus` shape:** `Node`, `NPM`, `NPX` are each a `runtime.Tool{Name, Path, Version}`; `Missing []string` lists absent tools; `NPXAvailable bool` is the single gate for add + lockfile-backed delete. Versions are normalized by stripping a leading `v` and trimming, so `v26.4.0` and `26.4.0` both render `26.4.0`.
- **`ConfirmContinueWithoutNPX(in io.Reader, out io.Writer) bool`** writes the warning + `Continue? [y/N] ` prompt to `out`, reads one line from `in`; only `y`/`yes` (case-insensitive) return true, empty/junk default to false. `fmt.Fprint*` returns discarded (`_, _ =`) for errcheck.
- **Capability flags live on `Model` via functional options.** `app.NewModel(result, opts ...Option)` accepts `app.WithAddEnabled(bool)` and `app.WithLockedDeleteEnabled(bool)`; both default `true` (existing `NewModel(result)` calls unchanged). Read with `m.AddEnabled()` / `m.LockedDeleteEnabled()`.
- **Command palette landed here (minimally).** `:` opens the palette (`m.palette`, rendered `: (a) add  (d) delete  esc cancel`). While open, `handlePaletteKey` intercepts keys so `a`/`d` are palette commands, NOT the SKILL.md/Assets tabs. `esc` closes it. When add is disabled, `:a` sets `m.status` to `Adding skills is disabled: npx is not available.` (rendered in `theme.Error`). **Slice 4 must build the real add wizard on this hook** — currently enabled `:a` and `:d` just close the palette with no action.
- **`main.go` flow:** home → `CheckDefault()` → `printDependencies` → if `!NPXAvailable`, `ConfirmContinueWithoutNPX` (exit 0 on decline) → scan → `NewModel` with capability flags → run.
- **Deps:** no new modules; `internal/runtime` is stdlib-only (`bufio`, `fmt`, `io`, `os/exec`; `slices`/`strings` in tests).

### Files created in Slice 3
- `internal/runtime/dependencies.go`, `internal/runtime/dependencies_test.go`, `internal/app/dependency_test.go`
- Modified `internal/app/{model,update,view}.go` and `cmd/trainer/main.go`.

Absorbs old Task 6 and the dependency-gating portion of Task 12.

**Files:**
- Create: `internal/runtime/dependencies.go`
- Test: `internal/runtime/dependencies_test.go`
- Modify: `cmd/trainer/main.go` — print versions, prompt, pass status into model
- Modify: `internal/app/model.go` — carry `AddEnabled`/`LockedDeleteEnabled` capability flags

**Interfaces produced:**
- `type DependencyStatus struct`
- `func CheckDependencies() DependencyStatus`
- `func ConfirmContinueWithoutNPX(in io.Reader, out io.Writer) bool`

### RED then GREEN order

1. **Detection reports present tools with versions.** Inject a fake lookup/version func; assert status lists node/npm/npx paths + versions, `NPXAvailable == true`.
2. **Missing `npx` marks add unavailable.** Inject lookup where `npx` is absent; assert `NPXAvailable == false` and `npx` in `Missing`.
3. **Missing `node`/`npm` recorded as missing (warning-level).** Assert `Missing` contains them but detection still returns.
4. **`ConfirmContinueWithoutNPX` defaults to no on empty input.** Feed `"\n"` -> false; feed `"y\n"` -> true.
5. **Model respects capability flags.** Model with add disabled; assert `:a` path is rejected with an explanatory message (cross-reference Slice 4).

### Notes / open questions
- Detection must be injectable (function fields or interface) so tests never shell out to real `npx`. This is the one place unit-style tests are justified.
- Version parsing should tolerate `v26.4.0` and `26.4.0` shapes.

---

## Slice 4: Add skill — DONE

**Status:** Completed 2026-07-02. Verified with `make verify` (fmt-check, vet, test, lint: 0 issues) and `go build ./...`.

**User-observable behavior:** `:a` opens the add wizard. Step 1 asks for a source. Step 2 (only for SSH Git sources when 2+ usable key pairs exist) asks which SSH key to use. Step 3 suspends the TUI and runs interactive `npx skills add <source> -g` (prefixed with `GIT_SSH_COMMAND` when a key is chosen). On exit, Trainer rescans and refreshes regardless of exit code.

### Implementation decisions & handoff notes for Slice 4 (READ before Slice 5+)

> Decisions I made that were NOT dictated by the spec/plan, and must be reviewed:
> (1) **The wizard is hand-rolled with a Bubbles `textinput` + a plain key-driven SSH key list, NOT Huh v2.** The plan/spec named Huh, but Huh forms are awkward to drive through `Model.Update` key presses in integration tests (they own their own event loop). A hand-rolled wizard keeps the whole flow testable through the public `Update`/`View` path with real key messages. If Huh is required later, this is the seam to swap.
> (2) **Add execution is injected via `WithAddRunner`/`WithRescan` options** (function values), defaulting to nil (no-op) so tests never shell out. `main.go` wires the real `tea.ExecProcess` runner (suspends the TUI) and a `ScanGlobal` rescan closure. This mirrors the Slice 3 injectable-dependency pattern.
> (3) **SSH key list uses `j/k` + `enter`**, rendered as a simple `> name` list (not a Bubbles list component) — minimal and directly testable.
> (4) **`FindKeyPairs` errors in `main.go` are swallowed to `nil` keys** (missing `~/.ssh` shouldn't block the app); the SSH step simply never appears.

- **`internal/ssh`:** `IsSSHGitSource(source)` classifies `ssh://` and scp-like `git@host:...` as SSH; shorthand/HTTPS/local paths are not. `FindKeyPairs(dir)` returns `[]KeyPair{Name, PrivatePath, PublicPath}` for private keys with a matching `.pub`, ignoring lone `.pub` files, `known_hosts`, `config`, and subdirs. `os.ReadDir` yields sorted entries so pairs are name-sorted for free.
- **`internal/actions`:** `AddCommand(source, keyPath)` builds `npx skills add <source> -g`; a non-empty `keyPath` appends `GIT_SSH_COMMAND=ssh -i <keyPath>` onto `os.Environ()`. Construction is separate from execution so tests assert argv/env without running `npx`.
- **App wizard state (`internal/app/add.go`):** `m.wizard *addWizard` (nil when closed). Steps: `stepSource` → optional `stepSSHKey`. `handleWizardKey` intercepts all keys while the wizard is open (checked before the palette in `handleKey`). `esc` cancels; `enter` calls `advanceWizard`. On the final step, `runAdd` builds the command, clears the wizard, and returns the injected runner's `tea.Cmd`, whose completion emits `addFinishedMsg`; `Update` handles that by calling `refreshFromDisk` (rescan + reset selection + re-sync viewport).
- **`:a` when add is disabled** sets the status message and does NOT open the wizard (early return in `handlePaletteKey`).
- **Post-review fixes (applied):** `ctrl+c` quits from inside the wizard; an empty/whitespace source stays on the source step instead of running `npx skills add  -g`; `GIT_SSH_COMMAND` quotes the key path (`ssh -i "<path>"`) to survive spaces; `refreshFromDisk` resets `fileSel`/`subfocus` alongside `selected`.
- **Deps added:** `charm.land/bubbles/v2/textinput` (already in the bubbles v2.1.0 module; pulled in transitive `github.com/atotto/clipboard` + `github.com/rivo/uniseg` via `go mod tidy`).

### Files created in Slice 4
- `internal/ssh/keys.go`, `internal/ssh/keys_test.go`
- `internal/actions/add.go`, `internal/actions/add_test.go`
- `internal/app/add.go`, `internal/app/add_test.go`
- Modified `internal/app/{model,update,view}.go` and `cmd/trainer/main.go`.

Absorbs old Task 5 (SSH), the add half of Task 7, and the add/refresh portion of Task 12.

**Files:**
- Create: `internal/ssh/keys.go`
- Test: `internal/ssh/keys_test.go`
- Create: `internal/actions/add.go`
- Test: `internal/actions/add_test.go`
- Modify: `internal/app/{model,update,view}.go` — palette `:`, add wizard state, suspend cmd, refresh
- Test: `internal/app/add_test.go`

**Interfaces produced:**
- `func IsSSHGitSource(source string) bool`
- `func FindKeyPairs(dir string) ([]KeyPair, error)`; `type KeyPair struct`
- `func AddCommand(source string, keyPath string) *exec.Cmd`

### RED then GREEN order

1. **SSH detection classifies sources.** `git@github.com:owner/repo.git` and `ssh://git@host/owner/repo.git` -> true; `owner/repo`, HTTPS URL, local path -> false. (Justified unit tests: pure classification.)
2. **Key-pair discovery over a temp ssh dir.** Fixture with `id_ed25519` + `id_ed25519.pub`, plus `known_hosts`, `config`, a lone `.pub`, and a subdir. Assert only the valid pair returns.
3. **`AddCommand` without key -> `npx skills add <source> -g`.** Assert argv exactly.
4. **`AddCommand` with key -> `GIT_SSH_COMMAND=ssh -i <key>` in env.** Assert env entry + argv.
5. **App: `:` then `a` opens the add wizard (source prompt).** Assert `View()` shows the source input.
6. **App: SSH step appears only for SSH source with 2+ keys.** Two scenarios; assert step presence/absence.
7. **App: add disabled when `NPXAvailable == false`.** `:a` shows the explanatory message, no wizard.
8. **App: after add command exits, model rescans.** Inject a fake runner + scan func; assert refresh invoked via updated skill list. Real `npx` never runs in tests.

### Notes / open questions
- Keep command construction (`AddCommand`) separate from execution so app tests inject a fake runner and never launch `npx`.
- Bubble Tea v2 `Suspend`/`ExecProcess` API must be confirmed; suspend + resume + refresh is the riskiest integration point.
- Huh v2 forms drive the wizard; confirm v2 form composition inside a Bubble Tea model.

---

## Slice 5: Delete skill — DONE

**Status:** Completed 2026-07-02. Verified with `make verify` (fmt-check, vet, test, lint: 0 issues).

**User-observable behavior:** `:d` starts delete confirmation for the selected skill, explaining removal, possible broken symlinks, and that it affects the global directory. Confirming a skill that is in the lockfile runs `npx skills remove -g <skill-name>`; confirming a skill that is only on disk removes its directory directly. Lockfile-backed delete is disabled when `npx` is unavailable. After deletion, Trainer rescans.

Absorbs the delete half of Task 7 and the delete/refresh portion of Task 12.

**Files:**
- Create: `internal/actions/delete.go`
- Test: `internal/actions/delete_test.go`
- Modify: `internal/app/{update,view}.go` — `:d` confirm modal, strategy dispatch, refresh
- Test: `internal/app/delete_test.go`

**Interfaces produced:**
- `type Strategy` (in-lockfile = npx remove, on-disk = fs removal)
- `func DeleteStrategy(skill skills.Skill) Strategy`
- `func DeleteCommand(skillName string) *exec.Cmd`

### RED then GREEN order

1. **Strategy: skill in the lockfile -> npx-remove strategy.** Skill with `Lock != nil`.
2. **Strategy: skill only on disk -> filesystem-removal strategy.** `Lock == nil`.
3. **`DeleteCommand` -> `npx skills remove -g <name>`.** Assert argv exactly.
4. **App: `:d` shows confirmation with the warning text.** Assert substrings.
5. **App: confirming a skill only on disk removes the directory (temp-dir integration).** Real temp skill dir; assert gone after confirm and model rescanned.
6. **App: lockfile-backed delete disabled without `npx`.** Assert explanatory message, no removal.

### Notes / open questions
- Filesystem delete (skill only on disk) is the one action safe to execute for real in tests (temp dir). Lockfile-backed delete stays construction-only + injected runner.
- Deletion refresh should reuse the same rescan path as add (Slice 4) to avoid divergence.

### Implementation decisions & handoff notes for Slice 5

> Decisions not dictated by the spec/plan, review before relying on them:
> (1) The confirm modal is a **single-key `y`/anything-else prompt** (not a text field or two-button widget), matching the palette/wizard hand-rolled style so it drives through `Update` with real key messages. `ctrl+c` quits; any non-`y` key cancels.
> (2) Delete execution is injected via **`WithDeleteRunner`** (same `AddRunner` signature as add); `main.go` wires the same `tea.ExecProcess` runner for both add and delete. Tests pass no runner so npx never runs.
> (3) Filesystem removal uses `actions.RemoveDirectory` (`os.RemoveAll`) and is the only action executed for real in tests, over a temp dir.
> (4) Disabled-lockfile-delete wording: `"Deleting this skill is disabled: npx is not available."`

- **`internal/actions/delete.go`:** `Strategy` enum (`StrategyNPXRemove`, `StrategyFilesystem`); `DeleteStrategy(skill)` returns npx-remove when `skill.Lock != nil`, else filesystem; `DeleteCommand(name)` builds `npx skills remove -g <name>`; `RemoveDirectory(path)` wraps `os.RemoveAll`.
- **`internal/app/delete.go`:** `m.confirm *deleteConfirm` (nil when closed). `:d` → `startDelete` captures the selected skill and opens the confirm modal. `handleConfirmKey` intercepts keys while open (checked before palette in `handleKey`). Confirming dispatches on `DeleteStrategy`: npx path returns the injected runner's cmd (emits `deleteFinishedMsg` → `refreshFromDisk`); filesystem path removes the dir then refreshes inline. Lockfile delete with `lockedDeleteEnabled == false` sets a status message and does nothing.
- Reuses Slice 4's `refreshFromDisk` for the post-delete rescan.

### Files created in Slice 5
- `internal/actions/delete.go`, `internal/actions/delete_test.go`
- `internal/app/delete.go`, `internal/app/delete_test.go`
- Modified `internal/app/{model,update,view}.go` and `cmd/trainer/main.go`.

---

## Slice 6: Layout polish — DONE

**Status:** Completed 2026-07-02. Verified with `make verify` (fmt-check, vet, test, lint: 0 issues) and a binary build (`go build -o bin/trainer ./cmd/trainer`).

**User-observable behavior:** Pane and detail-tab labels include their keyboard shortcut (`(1) Scope`, `(2) Skills`, `(3) Detail`; `(a) SKILL.md`, `(b) References`, `(c) Scripts`, `(d) Assets`). The TUI fills the full terminal and reflows the three panes on resize. When the terminal is smaller than a minimum width/height, the app replaces the layout with a centered `[Too small] Resize terminal to view the full app` message and restores the layout once the terminal grows back.

New slice (not in the original horizontal plan) — captures full-screen/resize handling and shortcut discoverability surfaced during Slice 2.

**Files:**
- Modify: `internal/app/{view,model}.go` — labels with shortcuts, width/height-aware pane sizing, too-small guard
- Test: `internal/app/layout_test.go` — drive `WindowSizeMsg` through `Update`, assert `View()`

### RED then GREEN order

1. **Pane labels show shortcuts.** Assert `View()` contains `(1) Scope`, `(2) Skills`, `(3) Detail`.
2. **Detail tab labels show shortcuts.** Assert tabs render as `(a) SKILL.md`, `(b) References`, `(c) Scripts`, `(d) Assets`.
3. **Too-small terminal shows the resize message.** Send a `WindowSizeMsg` below the minimum; assert `View()` contains the too-small message and none of the pane titles.
4. **Growing back restores the layout.** After a too-small size, send a large `WindowSizeMsg`; assert pane titles return and the message is gone.
5. **Panes reflow to terminal width.** At two different widths, assert the rendered frame width tracks the terminal width (substring/width check, not a snapshot).

### Notes / open questions
- Pick concrete minimum thresholds (e.g. width < 60 or height < 15) and encode them as named constants.
- Keep assertions on substrings and coarse width checks; never snapshot the full frame (Lip Gloss padding is brittle).

### Implementation decisions & handoff notes for Slice 6

> Decisions not dictated by the spec/plan:
> (1) Minimum thresholds are `minWidth = 60`, `minHeight = 15` (constants in `model.go`). Below either, `View()` returns only the centered `[Too small] Resize terminal to view the full app` message (via `lipgloss.Place`); the layout returns automatically once the terminal grows back since it is recomputed from `m.width/m.height` each render.
> (2) Pane widths are computed from terminal width: scope pane fixed at `scopePaneWidth = 18`; the remainder splits into a skills list (`~1/3`, floor `minListWidth = 16`) and the detail pane (the rest). `paneBorderPad = 4` accounts for each pane's border+padding overhead. (Slice 7 note: `detailWidth` was later corrected to `detailPaneWidth − paneBorderPad`, and explicit pane heights were added — see Slice 7.)
> (3) Pane labels carry shortcuts: `(1) Scope`, `(2) Skills`, `(3) Detail`; tab labels were `(a) SKILL.md`, `(b) References`, `(c) Scripts`, `(d) Assets` here — **changed in Slice 7** to `(i)/(r)/(s)/(a)` to avoid `j`/`k` collisions.

### Files modified in Slice 6
- `internal/app/{model,view}.go` (thresholds, width-aware panes, too-small guard, labels)
- `internal/app/layout_test.go` (new)

---

## Slice 7: Detail rendering & layout polish (post-smoke fixes) — DONE

**Status:** Completed 2026-07-02. Verified with `make verify` (fmt-check, vet, test, lint: 0 issues) and a rendered smoke frame against the real `~/.agents/skills` at 130×40 (all three panes aligned and within the frame).

This slice was **not in the original plan** — it captures issues found during the first manual smoke test (screenshots) after Slices 5/6 landed. Every fix was done TDD (RED→GREEN, one behavior per cycle) with integration tests driven through `View()` / `ScanGlobal` over real temp-dir or in-memory fixtures.

### Problems found in smoke testing
1. Panes 2 and 3 overflowed past the bottom of the terminal (Lip Gloss grows a box past `Height()` instead of clipping — it does not auto-clip content).
2. Detail content was cut off at the bottom; the viewport was not sized to the available rows.
3. Raw YAML frontmatter (`## name: …`, `description: …`) leaked into the rendered `SKILL.md` tab.
4. Glamour used its built-in `dark` style, which clashed with the Gruvbox UI.
5. Skills list showed the full filesystem path instead of `local`; rows were poorly aligned with no selection highlight and no truncation.
6. No subfocus indicator (file list vs content) and no scroll indicator.
7. Tab shortcuts `(a)…(d)` collided conceptually with vim keys; user requested `(i) SKILL.md (r) References (s) Scripts (a) Assets`.
8. `h`/`l`/`enter` did not move focus between panes.

### RED→GREEN cycles (behavior per cycle)
1. **SKILL.md tab hides frontmatter.** `Skill` gained a `Body` field populated by the scanner from `ParseSkillMarkdown`; the SKILL.md tab renders `Body` via Glamour instead of re-reading the raw file by path. (`skills.Skill.Body`, `scanner.go`, `view.currentContent`.)
2. **Whole frame fits within the terminal.** New helpers `paneHeight()` / `paneContentHeight()` size every pane to `terminalHeight − frameMargin`; `pane` now takes an explicit height. Proven by `TestFrameFitsWithinTerminal` (80 skills at 100×30 must not exceed height/width).
3. **Skills list windows around the selection.** `windowBounds(n, selected, capacity)` keeps the selected skill visible; the list renders only the visible slice (2 rows/skill).
4. **Detail content viewport fits.** Header fields are truncated to one line each so header height is deterministic; the viewport is sized to `paneContentHeight − header − fileLines − contentHeader`.
5. **Skills meta shows `local`, dimmed.** `skillMeta` returns the literal `local` when unlocked (was the path). Meta line uses the `Muted` color.
6. **Selected row highlighted.** `skillRow` renders the selected skill's two lines with an `Elevated` background band + accent name; both lines truncate with `…` via `truncate`.
7. **`h`/`l`/`enter` move focus.** `focusLeft`/`focusRight` clamp at the edges; `enter` and `l` both move right.
8. **Tab labels/keys.** `(i) SKILL.md (r) References (s) Scripts (a) Assets`; `setTab` bound to `i`/`r`/`s`/`a`. Chosen so tab keys never collide with `j`/`k`.
9. **Subfocus indicator.** `sectionLabel(name, active)` renders `▸ Files` / `▸ Content` with accent when active, dim otherwise.
10. **Scroll indicator.** `contentSectionLabel` appends the viewport's `ScrollPercent()` as `Content (NN%)`.
11. **Gruvbox Glamour theme.** `render.GruvboxDarkHard()` returns a custom `ansi.StyleConfig` built from the spec palette, wired via `glamour.WithStyles`.

### Implementation decisions & handoff notes for Slice 7 (READ before further UI work)

> Decisions not dictated by the spec (spec has since been updated to match):
> (1) **Lip Gloss does not clip to `Height()`** — it grows the box. All fit/overflow work relies on pre-sizing content (windowing + viewport height + one-line-truncated header), not on the style clipping for us.
> (2) **Glamour strips color in non-TTY (test) output and exposes no color-profile override.** So the Gruvbox theme is tested by asserting `render.GruvboxDarkHard()`'s `StyleConfig` fields against the spec's literal hex values (independent source of truth), NOT by asserting rendered ANSI. Lip Gloss DOES emit color in tests, so row-highlight behavior is tested via `View()`.
> (3) **Detail content width** = `detailPaneWidth() − paneBorderPad` (was the full pane width, which caused wrapping/overflow). This is the width passed to Glamour/Chroma.
> (4) **`truncate`** is a shared helper (rune-aware via `lipgloss.Width`) used by both skill rows and detail header lines.
> (5) Section headers (`Files`/`Content`) only appear on file tabs (References/Scripts/Assets), never on the SKILL.md tab.

### Files changed in Slice 7
- `internal/skills/{skill,scanner}.go` — add and populate `Skill.Body`
- `internal/render/gruvbox.go` (new) — `GruvboxDarkHard() ansi.StyleConfig`
- `internal/render/markdown.go` — use `WithStyles(GruvboxDarkHard())`
- `internal/render/render_test.go` — Gruvbox palette assertion (+ rename local `ansi` regexp var to `ansiRE` to avoid the new import clash)
- `internal/app/{model,update,view}.go` — pane heights, windowing, truncation, row highlight, focus nav, tab keys, subfocus + scroll indicators
- `internal/app/{browse,detail,layout,palette,nav}_test.go` — new/updated integration tests

---

## Slice 8: Search + filter + demarcated Details — DONE

**Status:** Completed 2026-07-03. Verified with `make verify` (fmt-check, vet,
test, lint: 0 issues), a binary build, and a rendered smoke against the real
`~/.agents/skills` at 130×40 (search narrows the list, the origin filter
narrows and clears, the Details pane fills the width and shows the frontmatter
fences and every field above the body with a scrollbar, and the `?` help modal
lists all bindings). Eighteen RED→GREEN cycles, each proven through `Update` and
`View`. Existing tests that asserted the reversed behavior (frontmatter hidden,
the `Content (NN%)` header, the `▸` section labels, the `(3) Detail` title, and
`r`-as-References from any pane) were replaced with tests for the new behavior.

### Implementation decisions & handoff notes for Slice 8 (READ before Slice 9+)

> Decisions not dictated by the spec, review before relying on them:
> (1) **Search is a Bubbles `textinput` + `github.com/sahilm/fuzzy` (FindNoSort,
> so the list keeps name order).** Bubbles `list` was not adopted because its
> filter is an on-demand `/` prompt, not the always-visible search box the
> chosen design has.
> (2) **The origin filter and the scrollbar have no off-the-shelf component**, so
> they are built on public data: the filter is plain state rendered as a radio;
> the scrollbar is drawn from `viewport.TotalLineCount` and `ScrollPercent`. No
> scrolling or matching logic is reimplemented.
> (3) **The Skills-pane search/filter/selection is Model fields plus a focused
> `search.go` (with `visibleSkills` as the single source of truth), not a
> separate Skills-pane type.** The plan proposed a distinct type; it was kept as
> Model methods to avoid rewiring the selection/viewport/focus coupling. The
> logic is now in one place and testable, but a full type extraction is still
> available as a follow-up if the pane needs to be reused or swapped.
> (4) **`r` is the only context-dependent key** (reset in Skills, References in
> Details); `i`/`s`/`a` stay global. This is narrower than the spec's first
> draft, which gated all tab keys.
> (5) **The scrollbar reserves one column** (`scrollbarWidth`) inside the detail
> content width; content wraps to `contentWidth() = detailWidth() - 1`.
> (6) **`:u` reuses the add runner** (`WithAddRunner`) and is gated on the same
> `addEnabled`/`NPXAvailable` flag as `:a`.

### Refinements after manual testing

Fixes found by running the app, each proven by a test or a rendered smoke:
- **No frontmatter gap; the scrollbar reaches the bottom.**
  `render.TrimSurroundingBlankLines` (in `currentContent`) drops the leading and
  trailing blank lines Glamour and Chroma frame content with. The content
  viewport's height comes from `detailLayout`, the same computation `renderDetail`
  draws with, so the height the content scrolls by equals the height it is drawn
  at and the last line is reachable.
- **Filter is horizontal:** `Filter ● All ○ Remote ○ Local` on one line, wrapping
  between options in a narrow pane (a non-breaking space keeps each bullet with
  its label). `h`/`l` move the filter cursor; `j`/`k` still move the skill list.
- **Selection stays within the filtered list.** `moveSelection` walks
  `visibleSkills`, not the full list, so `j`/`k` never land on a hidden skill.
- **All detail keys are gated to Details focus** (`i`/`s`/`a`/`tab`/scroll); `r`
  resets in the Skills pane. Partial gating read as inconsistent.
- **Scripts and References show `No files`** when empty, like Assets.
- **The selected file uses a highlight band**, not a caret, matching the skill list.
- **Both modals (palette, help) are border-defined with no background fill**, so
  per-span color resets cannot leave inconsistent background gaps. The help modal
  is rendered from the `key.Binding` data with one key-column width and one key
  color.

### Files created/changed in Slice 8
- `internal/skills/{frontmatter,skill,scanner}.go` — keep raw frontmatter block
- `internal/actions/update.go` (+ test) — `UpdateCommand()`
- `internal/app/search.go` — search + origin filter + `visibleSkills`
- `internal/app/keys.go`, `internal/app/help.go` — `?` help modal via Bubbles `help`/`key`
- `internal/app/updateskills.go` — `:u` flow
- `internal/app/{model,update,view}.go` — wiring, dividers, scrollbar, `Details`
- `internal/app/{scanner,browse,detail,layout,search,palette,help}_test.go` — integration tests

**User-observable behavior:** The Skills pane has an always-visible search box
and an origin filter (All / Remote / Local) above the list. `/` focuses search;
typing narrows the list (fuzzy over name + source); `enter` keeps the narrowed
list; `esc` clears it. `f` focuses the filter; `j`/`k` move between options;
`space` selects; `c` clears to All. `r` in the Skills pane resets both. The
Details pane is titled `Details`, fills the full terminal width, drops the
description from its meta block, and shows the whole `SKILL.md` including its
frontmatter (fences and all fields) above the rendered body. Its sections (meta,
tabs, file list, content) are separated by divider lines; the content has a
scrollbar and no percentage header; the selected skill row has no caret. `?`
opens a help modal listing all bindings. `:u` runs `npx skills@latest update`.

### Off-the-shelf component decisions (the "don't hand-roll" rule)

- **Search box:** Bubbles `textinput`. **Matching:** `github.com/sahilm/fuzzy`
  (the library the Bubbles `list` component uses internally).
- **Help modal:** Bubbles `help` + `key`. The `?` modal renders the shared
  `key.Binding` values. (In the v0.99 audit pass the key handlers were changed to
  match against those same bindings with `key.Matches`, so the shown keys and the
  handled keys are one definition and cannot differ — see the audit notes.)
- **Content + scroll:** Bubbles `viewport` (already in use).
- **Not adopting Bubbles `list`:** its filter is an on-demand `/` prompt, which
  does not match the always-visible search and filter boxes in the chosen
  design. `textinput` + `fuzzy` gives the persistent boxes using off-the-shelf
  components.
- **No off-the-shelf fit, built minimally on public data (documented as such):**
  the origin filter radio (plain state + render) and the content scrollbar
  (drawn from `viewport.TotalLineCount` / `VisibleLineCount` / `ScrollPercent`;
  no scrolling logic is reimplemented).

### Modularity

The Skills-pane logic (search, filter, selection, windowing, row rendering) is
currently inlined in the app `Model`. Extract it into a single Skills-pane type
with a small interface (visible skills, selected skill, key handling, render) so
the pane is a deep module rather than scattered fields and functions.

**Files:**
- Modify: `internal/skills/{frontmatter,skill,scanner}.go` — keep the raw
  frontmatter block (fences + all fields) on `Skill`
- Modify: `internal/skills/scanner_test.go` — fixture with a non-name/description
  frontmatter field, asserted through `ScanGlobal`
- Create: `internal/actions/update.go` + `update_test.go` — `UpdateCommand()`
- Create: `internal/app/skilllist.go` — Skills-pane module (search + filter +
  selection + render)
- Create: `internal/app/keys.go` — `key.Binding` definitions shared by the key
  handler and the help modal
- Create: `internal/app/help.go` — `?` help modal via Bubbles `help`
- Create: `internal/app/scrollbar.go` — scrollbar from `viewport` metrics
- Modify: `internal/app/{model,update,view}.go` — wire the above, make `r`
  context-dependent (reset in Skills, References in Details), `Details` title,
  drop caret + description, dividers
- Test: `internal/app/{search_test,filter_test,help_test,detail_test,layout_test,browse_test}.go`
  — integration through `Update` (real key messages) and `View` over temp-dir /
  in-memory fixtures

**Interfaces produced:**
- `Skill.Frontmatter string` — raw frontmatter block including both `---` fences
- `func UpdateCommand() *exec.Cmd`
- a Skills-pane type exposing visible/selected/update/view

### RED then GREEN order (one behavior per cycle)

Data + rendering of the SKILL.md frontmatter:

1. **Scanner keeps the raw frontmatter.** Fixture `SKILL.md` whose frontmatter
   has a field that is neither `name` nor `description` (e.g. `license: MIT`).
   Assert (through `ScanGlobal`) the skill's raw frontmatter contains both `---`
   fences and `license: MIT`.
2. **`SKILL.md` tab shows the frontmatter and the body.** Build the model from
   that fixture; on the `SKILL.md` tab assert `View()` shows `---`, `license:
   MIT`, and a known body heading. (Proves frontmatter is no longer stripped.)

Details pane shape:

3. **Pane title is `(3) Details`.** Assert `View()` contains `(3) Details`.
4. **Selected skill row has no caret.** Assert the selected row shows the name
   without a leading `> ` (the elevated band marks selection, tested as in
   Slice 7).
5. **Meta block omits the description.** Fixture skill with a description; assert
   the description text does not appear in the meta block (it still appears in
   the `SKILL.md` content from cycle 2).
6. **Details fills the full terminal width.** Send a `WindowSizeMsg(W, H)`;
   assert the joined frame width equals `W` (equality, not `<= W`, so an
   underfilled layout fails).

Content section:

7. **No `Content (NN%)` header.** Assert `View()` contains neither `Content (`
   nor `%)`.
8. **Scrollbar appears only on overflow.** Long content in a short pane: assert
   a scrollbar glyph column is present. Short content that fits: assert it is
   absent.

Search:

9. **`/` focuses search; typing narrows the list.** Skills `[alpha, beta,
   gamma]`; `/` then type `be`; assert the list shows `beta` and not `alpha` /
   `gamma`.
10. **`enter` keeps the narrowed list and leaves search.** After typing, `enter`;
    assert the list is still narrowed and a following `j` moves the selection
    (not typed into the box).
11. **`esc` clears search.** After typing, `esc`; assert all skills show again.

Filter:

12. **`Remote` shows only lockfile-backed skills.** Fixture: one locked, one
    local; `f`, move to `Remote`, `space`; assert only the locked skill shows.
13. **`Local` shows only on-disk skills; `c` clears to All.** Assert both steps.
14. **Search and filter combine.** With a search term and `Remote` selected,
    assert the list is the intersection.

Reset and key gating:

15. **`r` in the Skills pane resets search and filter.** Set both; focus Skills;
    `r`; assert the full list is restored.
16. **`r` in the Details pane selects the References tab.** Focus Details; `r`;
    assert the References tab is active (confirms `r` is context-dependent: it
    resets in the Skills pane but selects References in the Details pane).

Help modal and update command:

17. **`?` opens a help modal listing bindings.** `?`; assert `View()` shows the
    search, filter, and quit bindings and their keys; `esc` closes it.
18. **`:u` builds `npx skills@latest update`.** Unit-assert `UpdateCommand()`
    argv exactly. App: `:u` runs the injected runner when `npx` is available and
    shows an explanatory message (no run) when it is not.

### Notes / open questions
- Assert on plain substrings that survive Glamour/Lip Gloss, never full-frame
  snapshots (per the standing plan rule).
- The scrollbar glyph choice is a rendering detail; assert its presence/absence
  on overflow, not exact characters.
- Selection must index into the *visible* (searched + filtered) list; when the
  visible set changes, keep the selection on a listed skill and follow it in the
  Details pane.

## Slice 9: Rebuild the add-skill wizard on Huh v2 — DONE

**Status:** Completed 2026-07-03. Verified with `make verify` (fmt-check, vet,
test, lint: 0 issues), a binary build, and rendered smokes of the source and
SSH-key steps at 110×26 (a centered rounded-border modal over the three panes,
themed to the Gruvbox palette). Thirteen RED→GREEN cycles driven through
`Model.Update`/`View`.

**User-observable behavior:** `:a` opens the add wizard as a **centered modal
overlay**, styled like the command palette (the earlier hand-rolled wizard was
appended below the full-height body and rendered off the bottom of the screen).
Step 1 is a source input; an empty source is rejected and stays put. For SSH git
sources with two or more usable keys, step 2 is an SSH-key select; other sources
skip it. `enter` confirms, `esc` cancels the modal, `ctrl+c` quits the app. On
completion Trainer runs `npx skills add <source> -g` (with `GIT_SSH_COMMAND` when
an SSH key was chosen) via the injected runner, then rescans.

### Verification of the original hand-roll rationale (required by this slice)

The prior handoff note claimed Huh forms are not drivable through `Model.Update`
and "own their own event loop." That is **false**, proven by a spike and a
throwaway prototype before any slice code was written:

- A `huh.Form` is a `tea.Model`; embedding it and forwarding messages to
  `form.Update` drives it. The form transitions groups and completes correctly.
- The real requirement the note missed: the parent must **propagate the
  `tea.Cmd`** Huh returns. Group transitions and completion arrive as messages
  produced by Huh's own cmds (`nextGroupMsg`), not synchronously inside one
  `Update`. A driver that discards the cmd never transitions — which is exactly
  what the existing `press` test helper does.

### Implementation decisions & handoff notes for Slice 9

- **The wizard is a single `huh.Form`** (`internal/app/add.go`): a source
  `huh.Input` group plus an SSH-key `huh.Select` group whose `WithHideFunc`
  reads the bound source and hides the step unless `IsSSHGitSource(source) &&
  len(keys) >= 2`. Empty-source rejection is a Huh `Validate`.
- **Message routing:** while the wizard is open, `Model.Update` forwards **every**
  message to `updateWizard` (not just key presses), because Huh's transition and
  completion messages are non-key. `ctrl+c` (quit) and `esc` (cancel) are gated
  before the form sees them — Huh binds quit to `ctrl+c` only, so `esc` would
  otherwise do nothing. A `WindowSizeMsg` is consumed for the app's own layout
  but **not** forwarded to the form: Huh sizes every group to the tallest
  group's height, which padded the short source step up to the SSH step and made
  the modal jump after `form.Init` requested the size. The wizard is a
  fixed-width modal, so it does not need the terminal size.
- **Gotcha:** a *hidden* conditional `Select` still defaults its bound value to
  its first option. So the SSH key is read only when the SSH step applied
  (`sshStepApplies`), or a non-SSH add would attach the first key. Covered by
  `TestNonSSHSourceAttachesNoKey`.
- **Rendering:** `renderWizard` wraps `form.View()` in the same rounded-border
  modal as the palette and is drawn via `overlayCenter`.
- **Theme:** the form is themed to the Gruvbox palette
  (`gruvboxHuhTheme` in `huhtheme.go`) so it matches the rest of the UI; Huh's
  default theme is indigo titles and a fuchsia select selector. The theme is
  built on `huh.ThemeBase` with only the accent-bearing styles overridden from
  the app `Theme`.
- **Test harness (`add_test.go`):** a dedicated wizard driver sends realistic
  keys (`{Text,Code}` for runes, `{Code}` for named keys) and pumps the returned
  cmd with a small bound (the cursor-blink tick would otherwise recur). Typing
  uses single `Update`s (characters insert synchronously); `enter`/navigation
  pump. The shared `press` helper is unchanged — it cannot drive Huh, and
  rewriting it would touch every test file for no gain.

### Files changed in Slice 9
- `internal/app/add.go` — rebuilt `addWizard` on `huh.Form`; `updateWizard`,
  `finishWizard`, `sshStepApplies`; `renderWizard` as a bordered modal
- `internal/app/huhtheme.go` (new) — `gruvboxHuhTheme` maps the app palette to a
  `huh.Theme`
- `internal/app/update.go` — wizard message routing; `:a` returns `form.Init()`
- `internal/app/view.go` — wizard drawn via `overlayCenter` (was appended below)
- `internal/app/add_test.go` — wizard-driving harness + 12 integration cycles
- `go.mod`/`go.sum` — `charm.land/huh/v2 v2.0.3` as a direct dependency

## Final slice: Full verification — DONE

**Status:** Completed 2026-07-03 (v0.99). All slices land; `make verify`
(fmt-check, vet, test, lint) is green with 0 lint issues, the binary builds, and
the app has been smoke-run against the real `~/.agents/skills` including the add
wizard. A full audit reconciled the implementation against the design spec and
the spec was reverse-updated to match the code (see the audit notes below the
checklist). Two known gaps are recorded there rather than fixed for v1.

Was Task 13. After all slices land:

```bash
gofmt -w cmd internal
make vet test lint
go run ./cmd/trainer   # manual smoke against real ~/.agents/skills
```

Manual smoke checklist:
- TUI opens; left pane shows `Global`.
- Skills pane lists skills from `~/.agents/skills`, selected row highlighted by
  the band (no caret), meta shows source or `local`.
- Search box: `/` focuses it, typing narrows the list, `enter` keeps the result,
  `esc` clears it.
- Filter: `f` focuses it, `space` picks `Remote`/`Local`, `c` clears; `r` in the
  Skills pane resets search and filter together.
- Details pane titled `Details`, fills to the terminal's right edge, meta has no
  description; sections separated by dividers.
- `i/r/s/a` switch tabs while Details is focused; file lists + rendering work;
  `SKILL.md` shows the frontmatter (fences + all fields) above the body.
- Content has a scrollbar when it overflows and no percentage header.
- `?` opens the help modal listing all bindings; `esc` closes it.
- `:` opens the command palette (centered); `:a`, `:d`, `:u` behave per
  dependency status.
- Resize: frame always fits; below 60×15 shows the too-small message and restores on grow.

### Audit notes (v0.99, 2026-07-03)

A full implementation-vs-spec audit reconciled the two. The implementation is the
source of truth and the design spec was reverse-updated to match it (the Huh add
wizard, the row `local` label, the search placeholder, `g` not `gg`, the
`pluginName` lock field, the npx-only dependency prompt, the delete-confirm
wording, the architecture file list).

Changes made off the back of the audit:

- **Warnings removed.** The scan-warning machinery (`ScanResult.Warnings`,
  `Skill.Warnings`, `Model.warnings`) was stripped. A malformed or
  frontmatter-less `SKILL.md` is still listed, keeping its directory name and an
  empty body; there is no longer a collected-but-unshown warning.
- **Search (`/`) and filter (`f`) are Skills-pane keys.** They act only while the
  Skills pane is focused, the same way the tab keys act only in the Details pane.
  Previously they worked from any pane.
- **Help modal and key handling are one definition.** `keys.go` holds a `keymap`
  of `key.Binding` values with their real keys and help labels; the handlers
  match with `key.Matches` and the help modal renders the same bindings, so the
  displayed keys and the handled keys are the same source and cannot list
  different keys. The earlier wrong entries were corrected: `g/G` (single `g`)
  not `gg/G`, the `ctrl+f`/`ctrl+b` full-page bindings are shown, and the filter
  keys are labelled "(filter focused)".
- **`j` / `k` are inert in the Scope pane.** They move the skills selection only
  while the Skills pane is focused. The Scope-pane key is reserved for scope
  selection once there is more than one scope.
- **Rendered content is flush left.** The Glamour document margin and the
  code-block margin are both zero, so the frontmatter YAML block and the body sit
  flush under the section dividers instead of indented.

Out of scope for v1 (by choice):

- The Skills-pane search/filter/list logic stays as `Model` methods in
  `search.go`. A separate-type extraction was proposed earlier but carries no
  user-visible change.

---

## Slice 10: Multi-scope browsing — DONE

**Status:** Completed 2026-07-03. Verified with `make verify` (fmt-check, vet,
test, lint: 0 issues), a binary build, and a rendered smoke against the real
home: `ScanAll` found 4 non-empty scopes — Global `.agents` (37 skills, 36
remote from its lock), Global `claude`/`codex`/`pi` (all local, symlinked into
`.agents`), with `opencode`/`cursor` and the Project section correctly omitted
(absent / no project skills). The rendered frame shows the two-level pane with
counts, and moving the scope selection switches the skill list. Ten RED→GREEN
cycles: five scanner cycles (over temp-dir fixtures with real symlinks and both
lock schemas) as pure additions, then a behavior-preserving reshape of the model
to `[]ScanResult` + `selectedScope`, then five view/navigation cycles.

### Implementation decisions & handoff notes for Slice 10 (READ before Slice 11)

> Decisions not dictated by the spec/plan, review before relying on them:
> (1) **The scope list is flat.** `Model.results []skills.ScanResult` +
> `selectedScope int` index the flat slice; the `Global`/`Project` headers are
> render-only grouping (`sectionOrder` + `scopeIndices`), so `j`/`k` never has to
> skip a header row and there is no header off-by-one to get wrong.
> (2) **`NewModel([]skills.ScanResult, opts...)`** is the honest production
> signature (main.go passes all scopes; a one-scope machine is a one-element
> slice). Tests use a `newTestModel(result, opts...)` helper that wraps a single
> scope. `RescanFunc` is now `func() []skills.ScanResult`.
> (3) **`ScanGlobal` is a thin wrapper over the new `Scan(dir, lockPath)`**, kept
> only so the existing `internal/skills` scanner tests still describe one scope;
> `main.go` no longer calls it.
> (4) **Symlink fix:** `Scan` dropped the `entry.IsDir()` guard (which is false
> for a symlink-to-dir, since `os.ReadDir` dirents come from lstat) and relies on
> `os.Stat(SKILL.md)` following the link; a plain-file child fails that stat and
> is skipped.
> (5) **`refreshFromDisk` re-clamps `selectedScope`** because `ScanAll` omits
> empty scopes, so a delete that empties a scope shifts every later index.
> (6) **Actions are unchanged this slice** (add/delete/update still force `-g`),
> so the existing action tests and the app stay green and shippable. Slice 11
> makes them scope-aware.

- **`internal/skills/harness.go`:** `ScopeDef{Label, Section, Dir, Lock}` with
  `Dir`/`Lock` relative to the section base (home for Global, cwd for Project)
  and kept as separate fields — the `.agents` lock sits beside `skills/`, the
  project lock is `skills-lock.json` at the cwd root. `Registry()` is the plain
  slice everything iterates; the Project section lists only `.agents`, `claude`,
  `pi` (codex/opencode/cursor share the project `.agents/skills`). `ScanAll(home,
  cwd)` builds paths from its two args (never `os.UserHomeDir()`, so it is
  drivable over temp dirs), scans each scope, omits empty ones, and tags each
  result's `Scope` with section + label + resolved path.
- **One lock reader parses both schemas** (v3 global, v1 project) because both are
  `{version, skills}` maps keyed by name; the schema differences (`sourceUrl`,
  `computedHash`) are just empty or ignored.
- **`internal/app/scope.go`:** two-level render (`renderScope`, `scopeRow` with a
  right-aligned count and an elevated highlight band on the selected scope),
  `moveScope` (inert with < 2 scopes; resets skill/file selection and re-syncs on
  change), `clampScope`, `scopeIndices`.
- **Zero-scope empty state** renders without panic (`currentSkills` returns nil,
  the detail pane shows `No skill selected`, no scope rows).

### Files created/changed in Slice 10
- `internal/skills/harness.go` (new) — `ScopeDef`, `Registry`, `ScanAll`
- `internal/skills/scanner.go` — `Scan(dir, lockPath)`; `ScanGlobal` wraps it
- `internal/skills/skill.go` — `Section` type + `Scope.Section`
- `internal/app/scope.go` — scope model, navigation, two-level render
- `internal/app/{model,update,view,search,add}.go` — `[]ScanResult` +
  `selectedScope`; `visibleSkills`/`refreshFromDisk` read the selected scope
- `cmd/trainer/main.go` — resolve cwd; `ScanAll(home, cwd)`; rescan over all scopes
- `internal/skills/scanner_test.go`, `internal/app/scope_test.go` (new) — integration
  tests; `internal/app/helpers_test.go` (new) — `newTestModel`; existing app
  tests repointed to `newTestModel` and their fixtures tagged with a section

**Original plan for this slice (seams under test, confirmed before writing tests):**
- `skills.Scan(dir, lockPath string) ScanResult` and `skills.ScanAll(home, cwd string) []ScanResult` — driven over real temp-dir fixtures (symlinks, real copies, several harness dirs, both lock schemas). The harness registry and per-scope lock reading are exercised *through* `ScanAll`, never shape-tested directly.
- `Model.Update` / `Model.View` — scope-pane navigation, the scope-scoped skill list, hidden empty scopes/sections, per-scope counts.

**User-observable behavior:** On launch the Scope pane shows a `Global` section and, when the current directory has project skills, a `Project` section, each listing one row per detected scope that has skills (`.agents`, `claude`, `codex`, `opencode`, `pi`, `cursor`) with a skill count. Absent or empty scopes are hidden; a section with no non-empty scopes is hidden. `j`/`k` move between scope rows while the Scope pane is focused; selecting a scope shows exactly that scope's skills. `.agents` scopes mark skills remote/local from their lock (global `~/.agents/.skill-lock.json` v3, project `<cwd>/skills-lock.json` v1); harness scopes show every skill `local`. Search and the origin filter operate within the selected scope. Add/delete/update are unchanged this slice (still force `-g`), so the app stays green and shippable.

**Files:**
- Create: `internal/skills/harness.go` — scope-definition struct, the registry, `ScanAll(home, cwd)`.
- Modify: `internal/skills/scanner.go` — `Scan(dir, lockPath)` that follows symlinks and copies (the current `ScanGlobal` skips symlinked dirs via `entry.IsDir()`); keep `ScanGlobal` as a thin wrapper or generalize it.
- Modify: `internal/skills/skill.go` — carry section + label on the scan result.
- Create: `internal/app/scope.go` — two-level scope model (sections + scope leaves), navigation, render.
- Modify: `internal/app/{model,update,view}.go` — hold `[]ScanResult` + `selectedScope`; `visibleSkills` reads the selected scope; render the two-level pane with counts.
- Modify: `cmd/trainer/main.go` — resolve cwd; build the registry; `ScanAll(home, cwd)`; rescan closure over all scopes.
- Test: `internal/skills/scanner_test.go`, `internal/app/scope_test.go` (+ updates to browse/layout tests that assume a single `Global` scope).

**Interfaces produced:**
- a harness/scope-definition struct + `func Registry() []...` carrying label, section, skills dir, optional lock path (skills-dir and lock-path are separate fields, not derived from each other)
- `func Scan(dir, lockPath string) ScanResult`
- `func ScanAll(home, cwd string) []ScanResult` — one per non-empty scope, tagged section + label

**RED then GREEN order (one behavior per cycle, integration over temp-dir fixtures):**
1. **Scanner discovers a symlinked skill dir.** Fixture: a scope dir with `foo -> ../store/foo` where `foo/SKILL.md` exists. Assert `Scan` returns `foo`. (Proves the `entry.IsDir()` symlink-skip is fixed.)
2. **Scanner lists a real (copied) skill dir and ignores non-skill dirs.** Fixture: `bar/SKILL.md` (real) + `notaskill/` (no SKILL.md). Assert only the skills return.
3. **`ScanAll` returns one result per non-empty scope, tagged section + label; empty/absent omitted.** Fixture home: `.agents/skills` (1), `.claude/skills` (1 symlink), `.codex/skills` (empty). Fixture cwd: `.agents/skills` (1). Assert scopes = Global/.agents, Global/claude, Project/.agents; Global/codex omitted.
4. **`.agents` scope reads its lock; harness scope does not.** Fixture: `.agents/skills` lock marks `foo` remote; `.claude/skills/foo`. Assert Global/.agents `foo` is remote (source shown); Global/claude `foo` is local.
5. **Project `.agents` scope reads `skills-lock.json` (v1) at the cwd root.** Fixture cwd with the v1 lock. Assert a project skill shows its `source`.
6. **View: Global header + scope rows + counts; first scope selected; its skills listed.** Assert `View()` shows `Global`, `.agents`, a count, and the first scope's skills.
7. **View: Project section appears only when a project scope has skills.** Two fixtures; assert presence/absence of `Project`.
8. **Scope navigation changes the selected scope and the Skills list follows.** Focus Scope, `j`; assert the listed skills are the next scope's.
9. **Harness scope lists every skill `local`.** Select a harness scope; assert its rows show `local`, even for a skill the `.agents` lock lists.
10. **Empty section hidden.** cwd with no project skills → assert no `Project` section.

**Notes / seams:**
- Registry entries carry skills-dir and lock-path separately: the `.agents` lock sits beside `skills/`, but the project lock is at the cwd root (`skills-lock.json`), not inside `.agents`. Do not derive one path from the other.
- One lock reader parses both schemas (v3 global, v1 project); the `computedHash`/`sourceUrl` differences are ignored or empty.
- The registry is a plain slice that the scanner, scope pane, and actions all iterate, so adding a harness is one appended entry with no other change.
- Keep assertions on substrings, never full-frame snapshots.

---

## Slice 11: Scope-aware actions — DONE

**Status:** Done. v1.0. Builds on Slice 10's scope model.

**Seams under test:**
- `actions.AddCommand`, `actions.DeleteCommand` — argv (justified unit tests: pure construction).
- `Model.Update` / `Model.View` — the add wizard runs with no scope flag; delete dispatch by the selected scope; filesystem delete over a real temp dir.

**User-observable behavior:** `:a` runs `npx skills add <source>` with no scope flag, so npx's own prompts choose skills, agents, and Project/Global. `:d` on a lock-listed skill runs `npx skills remove <name>`, adding `--global` when the skill is in a Global-section scope (Trainer sets the flag from the skill's own scope, so the delete is deterministic); on a local/harness skill it deletes that skill's own directory entry at its scope path (removing a symlink leaves the canonical skill). `:u` is unchanged (already omits any scope flag). Every action rescans all scopes.

**Files changed:**
- `internal/actions/add.go` — `AddCommand(source, keyPath)` builds `npx skills add <source>` with no scope flag; `GIT_SSH_COMMAND` still set when a key is given.
- `internal/actions/delete.go` — `DeleteCommand(name, global)` builds `remove <name>` plus `--global` when `global`; strategy stays lock-vs-disk, the filesystem path is the scope-specific `skill.Path`.
- `internal/app/delete.go` — `deleteConfirm` carries the selected skill's `skills.Scope`; `startDelete` captures it via `selectedScopeDef`; `runDelete` passes `scope.Section == SectionGlobal` to `DeleteCommand`; the confirm text names the scope (`<name> (<Section>)`) instead of always saying "global". Refresh runs through `deleteFinishedMsg` → `refreshFromDisk`, which rescans every scope.
- Tests: `internal/actions/{add,delete}_test.go`, `internal/app/{add,delete}_test.go`.

**RED then GREEN order (as built):**
1. **`AddCommand` has no scope flag.** Argv asserted `npx skills add <source>` exactly; `GIT_SSH_COMMAND` still set with a key.
2. **`DeleteCommand(name, global)` argv.** Global → `npx skills remove <name> --global`; project → `npx skills remove <name>`. Both asserted.
3. **Delete of a lock-listed skill dispatches to npx-remove with the right scope flag.** Through `Update` with an injected runner; the command carries `--global` iff the skill's scope is Global, and refresh runs. Asserted for both sections.
4. **Delete of a local/harness skill removes its scope-specific directory.** Regression guards (temp-dir + symlink fixtures from Slice 5) stay green: the entry at `skill.Path` is removed and a symlink delete leaves the canonical skill.
5. **Confirm text no longer claims "global".** A Project-scope delete names "Project" and contains no "global".
6. **After each action, refresh repopulates every scope.** The injected rescan runs and the refreshed list reflects it.

**Implementation decisions & handoff notes for Slice 11:**

> - `-g` and `--global` are the same npx flag (`skills remove --help` confirms `-g, --global`). Delete uses the long form `--global` to match the spec; add omits the flag entirely so npx prompts for scope.
> - The scope flag is derived from the selected skill's own scope, captured into `deleteConfirm` at `startDelete`. `selectedSkill` → `visibleSkills` → `currentSkills` never leaves the selected scope, so the scope is authoritative and the flag is deterministic.
> - `runDelete` reads `scope.Section` into a local before it nils `m.confirm`, so the flag survives the reset.

---

## Slice 12: Context footer + palette-dimmed commands — TODO

**Status:** Not started. Target v1.0. Adds a permanent context keybind footer,
retires the persistent status line, and moves all `npx`-unavailability into the
command palette.

**What the user asked for (design, grilled 2026-07-03):**
- A permanent bottom row like herdr's prefix bar: a leading accent chip naming
  the current context, then the keys available from where you are.
- The footer shows only keys **not already on screen**: omit the pane digits
  `1/2/3` (shown in pane titles) and the Details tab keys `i/r/s/a` (shown in the
  tab bar). Never any error text.
- Contexts and their chips: `SCOPE`, `SKILLS`, `DETAILS` (one chip; the key list
  changes by tab/subfocus), `SEARCH`, `FILTER`. Hidden entirely while an overlay
  modal (palette / confirm / wizard / help) is open.
- No persistent red status line anywhere. `npx` unavailability shows **only** in
  the command palette: the affected command is dimmed with a muted
  `disabled without npx` tag, and a dimmed command does nothing when pressed.
  The tag never mentions lockfiles or any internal mechanism.
- The pre-TUI `Continue? [y/N]` prompt is pointless once the palette carries this,
  so it is removed; the TUI always launches.
- The one genuine on-disk delete failure (`os.RemoveAll` error) shows no message:
  the skill stays in the list after the refresh, so the failure is visible by the
  skill not disappearing.

**Footer contents by context (exact, after omissions):**
- `SCOPE` — `j/k switch scope · h/l move focus · : commands · ? keys · q quit`
- `SKILLS` — `j/k select · / search · f filter · r reset · h/l move focus · : commands · ? keys · q quit`
- `DETAILS`, SKILL.md tab — `j/k scroll · ctrl+d/u half-page · ctrl+f/b page · g/G top/bottom · h/l move focus · : commands · ? keys · q quit`
- `DETAILS`, file tab + list active — `j/k select file · tab focus content · h/l move focus · : commands · ? keys · q quit`
- `DETAILS`, file tab + content active — `j/k scroll · ctrl+d/u half-page · ctrl+f/b page · g/G top/bottom · tab focus files · h/l move focus · : commands · ? keys · q quit`
- `SEARCH` — `type to filter · enter apply · esc clear`
- `FILTER` — `h/l move option · space apply · c clear · esc done`

**Seams under test:**
- `Model.renderFooter() string` — the footer line for the current state (empty
  string when hidden). This is the seam the footer tests assert on directly
  (like `renderConfirm` / `renderWizard`), because asserting footer content
  against the whole `View()` would collide with the tab bar, which legitimately
  contains `i/r/s/a`. Tests drive the model into each context with real key
  presses, then assert substrings of `renderFooter()`.
- `Model.renderPalette() string` and palette dispatch (`Model.Update`) — dimmed
  commands, the `disabled without npx` tag, and that a dimmed key is inert.
- `Model.View()` height — the footer occupies one reserved row and the frame
  still fits the terminal.
- `runtime` package — `ConfirmContinueWithoutNPX` is removed (its test deleted).

**Files:**
- New: `internal/app/footer.go` — `footerContext()` resolving state → context,
  and `renderFooter()` building the chip + hint line; hidden when any overlay is
  open. Chip styled like the palette/help accents (accent background, bold);
  keys in `theme.Secondary`, descriptions in `theme.Muted`; middot separator.
- New: `internal/app/footer_test.go`.
- Modify: `internal/app/view.go` — join `renderFooter()` as the bottom row;
  `paneHeight` subtracts one row for the footer; remove `renderStatus` and the
  `m.status` rendering; `renderPalette` dims `npx`-only commands and appends the
  `disabled without npx` tag.
- Modify: `internal/app/model.go` — remove the `status` field.
- Modify: `internal/app/update.go`, `updateskills.go`, `delete.go` — drop the
  `m.status` assignments; the disabled paths are now unreachable (palette dim),
  and the on-disk delete failure just refreshes (skill remains).
- Modify: `internal/app/view.go` / palette gating — a dimmed command key is inert
  in `handlePaletteKey`.
- Modify: `cmd/trainer/main.go` — remove the `ConfirmContinueWithoutNPX` call and
  the (alt-screen-wiped) version printout; always launch.
- Modify: `internal/runtime/dependencies.go` + `dependencies_test.go` — remove
  `ConfirmContinueWithoutNPX` and its test.
- Modify: `internal/app/{palette,layout,delete}_test.go` — extend as below.
- Bump `minHeight` by one row for the footer if needed.

**RED then GREEN order (one behavior per cycle):**
1. **Footer renders the SKILLS context.** With the Skills pane focused,
   `renderFooter()` contains the `SKILLS` chip and `j/k`, `/`, `f`, `r`; it does
   **not** contain `1/2/3` or `i/r/s/a`. (RED: no footer yet.)
2. **Footer for the SCOPE context.** Focus pane 1; footer shows `j/k switch scope`
   and the global tail, and none of `/`, `f`, `r`.
3. **Footer for DETAILS / SKILL.md tab.** Focus pane 3 on the SKILL.md tab; footer
   shows the scroll keys (`ctrl+d/u`, `ctrl+f/b`, `g/G`), no `tab` toggle, and
   omits `i/r/s/a`.
4. **Footer for DETAILS / file tab, list active.** Select a file tab (subfocus
   list); footer shows `j/k select file` and `tab focus content`.
5. **Footer for DETAILS / file tab, content active.** Toggle subfocus to content;
   footer shows the scroll keys and `tab focus files`.
6. **Footer for SEARCH mode.** Enter search; footer shows the `SEARCH` chip and
   `enter apply · esc clear`, and none of the pane keys.
7. **Footer for FILTER mode.** Focus the filter; footer shows `h/l move option ·
   space apply · c clear · esc done`.
8. **Footer hidden during modals.** With the palette (then confirm, wizard, help)
   open, `renderFooter()` is empty.
9. **Footer reserves one row and the frame fits.** After a `WindowSizeMsg`, the
   joined `View()` height is ≤ the terminal height and the footer line is present
   at the bottom; a pane is one row shorter than without the footer.
10. **Palette dims add/update without npx.** With `npx` unavailable, the palette
    shows `add` and `update` dimmed with the `disabled without npx` tag; pressing
    `a` opens no wizard and produces no status text (asserts no red line anywhere).
11. **Palette dims delete only for an npx-only selection.** With `npx` unavailable
    and a lock-tracked skill selected, `delete` is dimmed with the tag and `d` is
    inert; with an on-disk skill selected, `delete` is enabled and `d` starts the
    confirm.
12. **No status line remains; on-disk delete failure leaves the skill.** The
    `m.status` rendering is gone (no red line in any `View()`); a failed on-disk
    delete refreshes and the skill is still listed, with no message.
13. **Footer truncates right-to-left with `? keys` pinned.** At a width narrower
    than the DETAILS context line, `renderFooter()` fits within the width, keeps
    the chip and the leftmost context keys, shows an ellipsis where items were
    dropped, and still ends with `? keys`. Assert the line width ≤ the frame
    width and that `? keys` and the first context key survive while a
    middle/global item is gone.

**Cleanup (not red-green — deletions):**
- Remove `ConfirmContinueWithoutNPX` (runtime) and its test; remove the main.go
  call and the pre-TUI printout.
- Remove the `m.status` field and `renderStatus` once cycles 10–12 are green.

**Notes / open questions:**
- The footer draws from the same `keymap` the help modal uses, so a key shown is
  a key handled (the existing single-source-of-truth invariant holds).
- Narrow terminals (confirmed 2026-07-04): the footer drops whole `key desc`
  items from the right until it fits, keeps the chip and the leftmost context
  keys, marks the dropped run with an ellipsis, and pins `? keys` as the final
  item so it is never dropped. The global tail (`: commands`, `q quit`,
  `h/l move focus`) is trimmed before the context keys.
- Chip labels are UI copy, not new domain terms; no glossary/ADR change.

---

## Cross-slice risks (read before starting)

- **Bubble Tea v2 API changes.** v2 changed `Update` return types, key messages, and program options vs v1 and most tutorials. Verify against the vendored v2.0.7 source before writing update tests.
- **`go.mod` re-resolution.** Deps re-add as slices import them; run `go mod tidy` at the end of each slice.
- **View test brittleness.** Assert substrings, never full-frame snapshots — Lip Gloss styling and width padding make snapshots fragile and coupled to layout.
- **No real `npx` in tests.** Only filesystem delete of a skill on disk executes for real (temp dir). Everything else is construction + injected runner.
