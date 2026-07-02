# Trainer v1 Implementation Plan (vertical slices)

> **For agentic workers:** implement slice-by-slice using TDD. Each slice is a
> vertical tracer bullet: it cuts end-to-end (disk to logic to UI) and is proven
> by **integration tests over real temp-dir fixtures**, not unit tests of
> internal shapes. Do RED then GREEN one behavior at a time; never write all
> tests up front. Each slice must leave the app compiling and tests passing.

**Goal:** Build Trainer, a Go TUI for browsing, inspecting, adding, and deleting globally installed agent skills.

**Architecture:** Trainer rebuilds transient in-memory state from `~/.agents/skills/*/SKILL.md` and `~/.agents/.skill-lock.json`. Bubble Tea owns interaction state; skill scanning, rendering, add/delete actions, and SSH key detection live in focused internal packages behind small interfaces.

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
- V1 scans only `~/.agents/skills/*/SKILL.md` and `~/.agents/.skill-lock.json`.
- V1 has one scope: `Global`.
- Add uses `npx skills add <source> -g` and does not pass agent flags.
- Delete uses `npx skills remove -g <skill-name>` for skills in the lockfile and direct directory removal for skills only on disk.
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
internal/skills/{skill,frontmatter,lockfile,scanner}.go
internal/ssh/keys.go
internal/actions/{add,delete}.go
internal/runtime/dependencies.go
internal/render/{markdown,code}.go
internal/app/{theme,keymap,model,update,view}.go
```

---

## Slice map

| Slice | User-observable behavior | Absorbs old tasks | Status |
|-------|--------------------------|-------------------|--------|
| Foundation | Module + core data types | 1 | DONE |
| 1 (tracer) | Browse: scope pane, skill list, `j/k` selection, detail header, `q` quit | 2, 3, 4, browse parts of 9/10/11 | DONE |
| 2 | Inspect: tabs `a/b/c/d`, file lists, Glamour/Chroma render, asset no-preview | 8, detail parts of 9/10/11 | NOT DONE |
| 3 | Startup dependency check + continue prompt, disable `:a` when no `npx` | 6, part of 12 | NOT DONE |
| 4 | Add: `:a` wizard, SSH key select, suspend + run, refresh | 5, 7(add), part of 12 | NOT DONE |
| 5 | Delete: `:d` confirm, lockfile vs on-disk strategy, refresh | 7(delete), part of 12 | NOT DONE |
| Final | Full verification + manual smoke | 13 | NOT DONE |

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

## Slice 2: Inspect skill content — NOT DONE

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

---

## Slice 3: Startup dependency check — NOT DONE

**User-observable behavior:** Before the TUI opens, Trainer prints detected `node`/`npm`/`npx` versions. If `npx` is missing, it warns that adding is disabled and prompts `Continue? [y/N]`; declining exits before the TUI. If the user continues without `npx`, `:a` is disabled with an explanatory message and lockfile-backed delete is disabled.

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

## Slice 4: Add skill — NOT DONE

**User-observable behavior:** `:a` opens the add wizard. Step 1 asks for a source. Step 2 (only for SSH Git sources when 2+ usable key pairs exist) asks which SSH key to use. Step 3 suspends the TUI and runs interactive `npx skills add <source> -g` (prefixed with `GIT_SSH_COMMAND` when a key is chosen). On exit, Trainer rescans and refreshes regardless of exit code.

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

## Slice 5: Delete skill — NOT DONE

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

---

## Final slice: Full verification — NOT DONE

Was Task 13. After all slices land:

```bash
gofmt -w cmd internal
make vet test lint
go run ./cmd/trainer   # manual smoke against real ~/.agents/skills
```

Manual smoke checklist:
- TUI opens; left pane shows `Global`.
- Middle pane lists skills from `~/.agents/skills`.
- Detail pane updates as selection changes.
- `a/b/c/d` switch tabs; file lists + rendering work.
- `/` opens shortcuts; `:` opens the command palette.
- `:a` and `:d` behave per dependency status.

---

## Cross-slice risks (read before starting)

- **Bubble Tea v2 API drift.** v2 changed `Update` return types, key messages, and program options vs v1 and most tutorials. Verify against the vendored v2.0.7 source before writing update tests.
- **`go.mod` re-resolution.** Deps re-add as slices import them; run `go mod tidy` at the end of each slice.
- **View test brittleness.** Assert substrings, never full-frame snapshots — Lip Gloss styling and width padding make snapshots fragile and coupled to layout.
- **No real `npx` in tests.** Only filesystem delete of a skill on disk executes for real (temp dir). Everything else is construction + injected runner.
