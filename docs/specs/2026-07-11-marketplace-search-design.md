# Marketplace Skill Search — Design Brief

Status: agreed via grilling session 2026-07-11. This is the single source of decisions for the TDD plan. Read alongside:
- `docs/research/skills-find-api.md` — the skills.sh API (endpoints, params, response shapes).
- `docs/adr/0002-marketplace-search-over-skills-sh-http.md` — why HTTP + skills.sh download + gock.
- `CONTEXT.md` — glossary (Marketplace, Marketplace Skill, Skill Search, Install Count).

## Goal

Add **Skill Search** to the add flow: a step lets the user search the skills.sh **Marketplace**, preview a **Marketplace Skill** in the existing three-pane detail view, and install the chosen skill via `npx skills add`.

## Flow (wizard steps)

1. `:a` opens the add wizard on a **chooser** step: `Enter skill URL or repository` / `Search for skills`.
   - Picking "Enter…" keeps today's Huh input path unchanged (manual entry → optional SSH key → install).
   - Picking "Search" grows the modal (harmonica spring, §Animation) into the full three-pane search browser.
2. **Search browser** (near-full-window three-pane): search box (top) + results list (left) + Skill Detail (right, four tabs).
3. `Enter` on a result installs `source@skillId` via the existing add seam (ExecProcess suspends the TUI; npx runs interactively).
4. **Post-install** step: chooser `Find more skills` / `Finish`.
   - `Find more skills` → back to the search box, cache + results preserved.
   - `Finish` → close overlay, rescan disk once (`refreshFromDisk`), land browser selection on the newly installed skill if findable.

The post-install step is **search-flow-specific**; the manual "Enter reference" path keeps its current single-shot behaviour (do not rewrite already-tested wizard behaviour).

## Decisions (agreed)

1. **Entry point:** chooser as step 1 of the existing add wizard, in-place swap. One entry (`:a`).
2. **Escape ladder (step-back):** in list/detail → search box; search box → chooser; chooser → close. `/` is the fast jump to the search box from anywhere in results.
3. **Context-aware footer is mandatory:** the footer/status helper (`internal/app/footer.go`) must show the live keybindings for the focused zone (search box / results list / detail), same as normal view. Nothing hidden.
4. **Grow animation:** `github.com/charmbracelet/harmonica` spring on modal width+height, driven by a capped `tea.Tick` frame loop that stops when settled. First animation/ticker in the repo. Animates an empty shell (before any search), so no mid-tween Glamour/Chroma cost.
5. **Sorts (client-side; API has none):** `r` Relevance (API fuzzy order), `p` Popularity (installs desc), `n` Name (A–Z). Pressing the same letter again toggles asc/desc. Default = **Popularity desc**. Sort keys live **only in results-list focus** (typing owns letters in the box; detail focus uses those letters for tabs). Sort bar renders at the top, driven from the list.
6. **Debounce:** 300ms after last keystroke; search only when query ≥ 2 chars (API 400s under 2). Under 2 chars → no request, "type to search" hint. New keystroke cancels the in-flight request (context cancel). Spinner while a request is active.
7. **Result limit:** `limit=25` (API has no pagination). Results are a windowed scrollable list.
8. **Download / preview:** metadata (name/source/installs) shows instantly with no fetch. The four content tabs need one download call. On selection dwell (~200ms) fetch `GET /api/download/<owner>/<repo>/<skillId>` (one JSON, all files inline). Constraints:
   - One download in flight at a time.
   - Moving the selection cancels the in-flight download and resets the active tab to **SKILL.md**.
   - Spinner in the content area while fetching.
   - Cache the download JSON per **Skill Search session**; cleared when the overlay closes.
   - **SKILL.md renders first** (fast first view); other files render on tab/file open. Rendering is **full-quality Glamour/Chroma**, per visible file, exactly like the existing browser; memoize rendered output per file.
   - Network staggering is impossible (endpoint is all-or-nothing) — fast-first-view is achieved by lazy *rendering*, not lazy fetching.
9. **Install ref:** `<source>@<skillId>` (e.g. `vercel-labs/agent-skills@vercel-react-best-practices`) — the form `npx skills add` accepts to install that exact skill. No SSH key needed. Reuse `actions.AddCommand` + injected `AddRunner` (prod: `tea.ExecProcess`).
10. **Result row:** two-line, mirroring `skillRow`. Line 1: skill name. Line 2: `source · <installs> installs` with comma thousands separators (e.g. `540,366 installs`).
11. **Empty/error states, inline in the relevant pane:** query < 2 → "Type at least 2 characters…" (results pane); zero results → `No skills found for "<q>"` (results pane); search failure/offline → "Search failed — space to retry" (results pane); download failure → "Couldn't load files — space to retry" (detail pane). Retry key = **`space`** (free in the search overlay context; the existing `space`=`filterApply` binding only applies in local-filter-focus). Footer shows the retry key. No new toast/notification system.
12. **Domain term:** a not-yet-installed found skill is a **Marketplace Skill** (glossary; bans remote/available/catalog/search-result as the noun).

## Control scheme (search overlay), by focus zone — all shown in the footer

- **Chooser step:** `j/k`/`↑↓` select; `Enter` pick; `Esc` cancel wizard.
- **Search box:** type (debounced); `Enter` or `↓` → jump to results list; `Esc` → chooser.
- **Results list (pane 1):** `j/k`/`↑↓` navigate; `r`/`p`/`n` sort (same key again toggles asc/desc); `l` → detail; `Enter` → install; `/` → search box; `Esc` → search box; `space` → retry (error state).
- **Detail (pane 2):** `i`/`r`/`s`/`a` tabs (SKILL.md/References/Scripts/Assets); `tab` toggles file-list/content subfocus; `j/k` file-nav/scroll; `h` → list; `Enter` → install; `/` → search box; `Esc` → list; `space` → retry (download error).
- **Post-install step:** `j/k` select; `Enter` pick `Find more skills` / `Finish`; `Esc` = Finish.
- Keys are **pane-scoped** as the app already does — a key only acts in the pane that owns it.

## skills.sh API (summary — full detail in docs/research/skills-find-api.md)

- **Search:** `GET https://skills.sh/api/search?q=<query>&limit=25`. `q` ≥ 2 chars (else 400). Response: `{query, searchType, skills:[{id, skillId, name, installs, source}], count, duration_ms}` — **metadata only**.
- **Download (one call, all files):** `GET https://skills.sh/api/download/<owner>/<repo>/<skillId>` → `{files:[{path, contents}], hash}`. `contents` is inline UTF-8 (not base64, not a zip). Build the tab file-tree client-side by splitting `path` on `/`: SKILL.md (root), References (`references/**`), Scripts (`scripts/**`), Assets (`assets/**`) — match the existing scanner's file classification.
- No server-side sort or pagination. Ordering is client-side.
- Env overrides in the CLI: `SKILLS_API_URL`, `SKILLS_DOWNLOAD_URL` — the Go client should allow a configurable base URL for tests.

## Codebase integration points (from the map)

- Wizard: `internal/app/add.go` (`addWizard`, `newAddWizard`, `updateWizard`, `finishWizard`, `runAdd`, `refreshFromDisk`, `renderWizard`); routed while `m.wizard != nil` in `internal/app/update.go:12-24`.
- Palette entry: `handlePaletteKey` (`internal/app/update.go:261-289`), `:` then `a`.
- Add command: `internal/actions/add.go` (`AddCommand(source, keyPath)` → `exec.Command("npx","skills@latest","add",source)`); `AddRunner` seam `internal/app/model.go:63-66`; prod wiring `cmd/trainer/main.go:51-53` (`tea.ExecProcess`); completion `addFinishedMsg` → `refreshFromDisk`.
- Browser to mirror: `internal/app/view.go` (`renderSkillList` 111-136, `skillRow` 159-189, `windowBounds` 140-157, `renderDetail` 198-229, `renderTabs` 408-427, `currentFilesAndRenderer` 473-488, viewport+scrollbar 301-358), focus/subfocus (`update.go:234-252`), tab switching (`applyDetailKey` 138-153, `setTab` 254-259).
- Keymap: `internal/app/keys.go` (single source of truth). Footer renders from bindings (`internal/app/footer.go`); help modal too.
- Modal sizing: `overlayCenter` (`view.go:60-74`) centers/clips; pane geometry `view.go:572-643`; terminal size `m.width/m.height`.
- File classification at scan time: `internal/skills/scanner.go:84-109` (`collectFiles` over `references/`, `scripts/`, `assets/`).
- Render: `internal/render/{markdown,code}.go` (Glamour Gruvbox, Chroma).

## Testing strategy

- **Integration over fakes.** New dependency `github.com/h2non/gock`. Capture real skills.sh responses once into recorded fixtures; replay through gock in tests. The HTTP client must use a gock-interceptable transport (default transport / `http.DefaultClient` or a client whose `Transport` gock can attach to) and a configurable base URL.
- Follow the repo's convention: drive public entry points (`Model.Update`, `Model.View`) against the rendered output; assert on user-observable text and on captured `exec.Cmd` args/env via the injected `AddRunner`. Stdlib `testing` only (plus gock). `make verify` gates.
- No `testdata/` convention exists today; fixtures for HTTP may introduce one — decide fixture location in the plan (e.g. `internal/marketplace/testdata/`).

## Seams to test (candidate — confirm in the plan)

- Marketplace HTTP client: `Search(ctx, query) → []MarketplaceSkill` and `Download(ctx, owner, repo, skillId) → SkillFiles` (gock-backed).
- `Model.Update`/`Model.View` for: chooser step, search debounce → results, sort keys + direction toggle, dwell → download + spinner, tab render (SKILL.md first), install ref passed to AddRunner, post-install chooser, empty/error states, Escape ladder, footer contents per zone.

## Non-goals / deferred

- Owner filter (`&owner=`) — deferred; global search only this slice.
- "✓ installed" badge on results during the Find-more loop — optional follow-on.
- Unifying the manual-entry path with the post-install step — out of scope.
- Fetching files individually from GitHub — rejected (rate limit + non-GitHub sources), see ADR-0002.
