# Marketplace Skill Search Implementation Plan (vertical slices)

> **For agentic workers:** implement slice-by-slice using TDD. Each slice is a
> vertical tracer bullet proven by **integration tests** — the marketplace
> client over recorded real skills.sh fixtures replayed with gock, the UI
> through `Model.Update`/`Model.View` with real key/messages. Do RED then GREEN
> one behavior at a time; never write all tests up front. Each slice must leave
> the app compiling and `make verify` green. Never commit with git.

**Goal:** Add **Skill Search** to the add flow: a chooser step lets the user
search the skills.sh **Marketplace**, preview a **Marketplace Skill** in the
existing three-pane detail view, and install the chosen skill via
`npx skills add <source>@<skillId>` through the existing add seam.

**Authoritative inputs:** `docs/specs/2026-07-11-marketplace-search-design.md`
(agreed decisions), `docs/research/skills-find-api.md` (skills.sh API),
`docs/adr/0002-marketplace-search-over-skills-sh-http.md`, `CONTEXT.md`
(glossary — Marketplace, Marketplace Skill, Skill Search, Install Count).

**Architecture:** a new `internal/marketplace` package is the HTTP seam to the
Marketplace (first HTTP client in the repo); the Skill Search overlay is Bubble
Tea state in `internal/app` that injects that client, reuses the existing
render/layout primitives, and installs through the existing
`actions.AddCommand` + `AddRunner` seam. The download JSON lives in a
session-scoped in-memory cache cleared when the overlay closes (ADR-0001/0002:
no persistent cache).

**Tech stack additions:** `github.com/h2non/gock` (test-only HTTP replay),
`github.com/charmbracelet/harmonica` (spring for the grow animation — first
animation/ticker in the repo). Everything else reuses installed dependencies.

---

## Why this plan is sliced vertically

A horizontal cut (all types, then the whole client, then the whole overlay)
would produce shape-tests of imagined structures. Instead the feature is one
end-to-end build broken into ordered tracer bullets: the first slices stand up
`internal/marketplace` and prove the real HTTP code path against recorded
fixtures; each later slice wires one user-visible behavior of the overlay —
chooser, animation, debounce, results, detail, install, post-install, errors,
escape ladder, footer — through the public `Model.Update`/`Model.View` seams.
Private request builders, decode structs, epochs, and memo maps are
collaborators exercised *through* those seams, never tested directly.

## Testing philosophy for this plan

- **Integration over fakes (ADR-0002).** Marketplace client tests replay
  responses captured **once** from the live skills.sh API through gock, over
  the real HTTP code path. No hand-written fake servers or fake clients.
- **UI tests drive public entry points.** Build the model with the real
  `*marketplace.Client` injected (`WithMarketplace`), stub the network with
  gock, send real key messages and the overlay's own async messages through
  `Update`, assert on user-observable `View()` text and on captured
  `exec.Cmd` args/env via the injected `AddRunner`. Stdlib `testing` only
  (plus gock).
- **No tautological tests.** Expected values are literals lifted from the
  recorded real payloads (`540,366 installs`, `vercel-react-best-practices`,
  the download `hash`) or hand-authored sort inputs/outputs — independent
  sources of truth, never recomputed the way the code computes them.
- **No implementation-coupled tests.** Never reach into private decode
  structs, request builders, epoch counters, or the render memo map. Cache
  behavior is asserted by what is observable: gock reports whether a second
  request was made.
- **Fixtures are recorded once, by hand.** `record_test.go` carries
  `//go:build record` so it is invisible to `go test ./...` and CI; run
  `go test -tags record ./internal/marketplace` once against the live API,
  commit the JSON, never run it again unless the API changes.

## Global constraints

- Never commit with git; `make verify` (fmt-check, vet, test, lint) gates
  every slice; golangci-lint must pass with 0 issues.
- Use `CONTEXT.md` vocabulary everywhere (code identifiers, test names, UI
  copy): **Marketplace**, **Marketplace Skill**, **Skill Search**, **Install
  Count**. Banned nouns: remote/available/catalog skill, search result (as the
  noun), registry, store, downloads.
- No Trainer-owned database, cache, or persistent index. The download cache is
  per-Skill-Search-session, in memory, cleared when the overlay closes.
- Search hits `GET {base}/api/search?q=<q>&limit=25`; download hits
  `GET {base}/api/download/<owner>/<repo>/<skillId>`. No auth, no retries, no
  custom headers. `limit=25`, no pagination. Base URLs configurable for tests
  (mirrors the CLI's `SKILLS_API_URL`/`SKILLS_DOWNLOAD_URL`).
- Queries under 2 characters never produce a request (API 400s under 2);
  debounce is 300ms; one search and one download in flight at a time; a new
  keystroke or selection move cancels the in-flight request via context.
- Sorting is client-side over the returned page: Relevance (API order),
  Popularity (Install Count), Name. Default **Popularity desc**. Same sort key
  pressed again toggles direction. Sort keys act only in results-list focus.
- Install ref is `<source>@<skillId>`; reuse `actions.AddCommand` +
  injected `AddRunner` (prod: `tea.ExecProcess`), no SSH key.
- The manual "Enter skill URL or repository" path is **not rewritten**: it is
  reached through the chooser and is otherwise the already-tested Huh wizard.
  The post-install chooser is search-flow-only.
- The context footer must show live keybindings for every Skill Search zone
  (spec decision 3). Keys are pane-scoped exactly as the app already does.
- Rendering reuses the existing primitives (`render.Markdown`, `render.Code`,
  `renderTabs`, `renderFileList`, `windowBounds`, scrollbar) — the Marketplace
  detail differs only in sourcing file contents from memory, not
  `os.ReadFile`. SKILL.md renders first; other files render lazily on open,
  memoized per file (lazy *rendering*, not lazy fetching — the download
  endpoint is all-or-nothing).

### Decisions resolved by this plan (were open in the drafts)

1. **Two base URLs.** The client keeps `WithSearchBaseURL` /
   `WithDownloadBaseURL` plus `WithBaseURL` setting both, mirroring the real
   CLI's two env overrides. Default `https://skills.sh` for both.
2. **Client-side `< 2` guard stays.** `Search` returns `ErrQueryTooShort`
   without a request for queries under 2 characters, so the client is safe
   even if a caller skips the UI debounce.
3. **`testdata/` convention.** First `testdata/` in the repo lands at
   `internal/marketplace/testdata/` (recorded JSON fixtures).
4. **Fixture fidelity.** Search fixtures are verbatim live responses. The
   download fixture is the live response with the `files` array trimmed to a
   structurally-real subset (SKILL.md plus a few real files, each kept
   byte-for-byte) so the repo stays small. `Classify` tests use hand-authored
   input covering all four tabs plus root extras.
5. **No `Searcher` interface.** `WithMarketplace(*marketplace.Client)` injects
   the concrete client; tests use the real client with gock (never a fake), so
   an interface would have no second implementation. `nil` disables the
   "Search for skills" option.
6. **Install Count comma formatting is a local helper** (`commaInt`, ~10
   lines); `dustin/go-humanize` stays an indirect dependency.
7. **File names.** The overlay lives in `internal/app/skillsearch.go` (the
   glossary term), not in the existing `search.go` (which owns the local
   Skills-pane search); the chooser in `internal/app/chooser.go`.

## Target file structure

Files land as their owning slice needs them.

```
internal/marketplace/
  client.go          Client, options, New, Search, Download, ErrQueryTooShort, APIError
  types.go           MarketplaceSkill (+ InstallRef, OwnerRepo), SkillFiles, File
  classify.go        Classify: bucket a downloaded tree into the four tabs
  sort.go            SortField, SortDir, SortSkills
  client_test.go     Search/Download integration tests over fixtures via gock
  classify_test.go   pure classification tests (hand-authored input)
  sort_test.go       pure sort tests (no gock, no network)
  record_test.go     //go:build record — captures live fixtures ONCE, never in CI
  testdata/
    search_react.json
    search_empty.json
    search_too_short_400.json
    download_vercel-react-best-practices.json
    download_unknown_404.json

internal/app/
  chooser.go         addChooser (entry + post-install steps), update + render
  skillsearch.go     searchOverlay state, zone key handling, async msgs, render
  (modified)         model.go, update.go, add.go, view.go, keys.go, footer.go, help.go

cmd/trainer/main.go  (modified) WithMarketplace(marketplace.New())
go.mod               + gock (slice 1, test-only), + harmonica (slice 5)
```

---

## Slice map

| Slice | User-observable behavior | Seams | Status |
|-------|--------------------------|-------|--------|
| 1 | Skill Search over HTTP: `Client.Search` returns Marketplace Skills from the real API shape | `marketplace.Client.Search` (gock) | DONE |
| 2 | One download call returns a Marketplace Skill's full file tree; files bucket into the four tabs | `marketplace.Client.Download`, `marketplace.Classify` | DONE |
| 3 | Deterministic client-side ordering by Relevance / Install Count / Name | `marketplace.SortSkills` | DONE |
| 4 | `:a` opens a chooser: manual entry (unchanged Huh wizard) or Skill Search | `Model.Update`/`View` | DONE |
| 5 | Choosing Search grows the overlay shell (harmonica spring) and lands focus in the search box | `Model.Update`/`View` | DONE |
| 6 | Typing searches the Marketplace: 300ms debounce, ≥2-char gate, stale results dropped, in-flight cancel, spinner | `Model.Update`/`View` + `Client.Search` via gock | DONE |
| 7 | Results list: two-line rows with Install Count, windowed j/k nav, r/p/n sorts with direction toggle | `Model.Update`/`View` | DONE |
| 8 | Selection dwell downloads once per skill and renders SKILL.md first; moves cancel + reset | `Model.Update`/`View` + `Client.Download` via gock | DONE |
| 9 | Skill Detail tabs: i/r/s/a, file/content subfocus, Glamour/Chroma over in-memory contents, memoized | `Model.Update`/`View` | DONE |
| 10 | `Enter` installs `<source>@<skillId>` through the existing add seam | `Model.Update` + injected `AddRunner` | DONE |
| 11 | Post-install chooser: Find more skills (state preserved) / Finish (one rescan, land on the new skill) | `Model.Update`/`View` | DONE |
| 12 | Inline empty/error states with `space` retry in results and detail panes | `Model.Update`/`View` (+ gock) | DONE |
| 13 | Escape ladder detail→list→box→chooser→closed; `/` jumps to the box | `Model.Update`/`View` | DONE |
| 14 | Context footer chips + live keys for every Skill Search zone; help modal lists the overlay group | `Model.renderFooter()`, `Model.View()` | DONE |
| Final | Full verification + manual smoke of the whole flow | all | DONE |

---

## Slice 1: Marketplace client — Skill Search over HTTP

**User-observable behavior (package seam):** `marketplace.New()` produces a
client whose `Search(ctx, query, limit)` returns the Marketplace's ranked page
as `[]MarketplaceSkill` (metadata only: `Name`, `SkillId`, `Source`,
`Installs`), errors typed and inspectable, cancellable via context.

**Files:**
- Create: `internal/marketplace/types.go` — `MarketplaceSkill` (+ `InstallRef()
  string` = `Source + "@" + SkillId`, `OwnerRepo()` splitting `Source`)
- Create: `internal/marketplace/client.go` — `Client`, `New(opts...)`,
  `Option` (`WithBaseURL`, `WithSearchBaseURL`, `WithDownloadBaseURL`,
  `WithHTTPClient`), `Search`, `ErrQueryTooShort`, `APIError{StatusCode, URL,
  Body}`
- Create: `internal/marketplace/record_test.go` — `//go:build record`; run once
  by hand to write verbatim live responses into `testdata/`
- Create: `internal/marketplace/client_test.go`
- Create: `internal/marketplace/testdata/{search_react.json,search_empty.json,search_too_short_400.json}`
- Modify: `go.mod` — `github.com/h2non/gock` (test-only)

**Interfaces produced:**
- `func New(opts ...Option) *Client`
- `func (c *Client) Search(ctx context.Context, query string, limit int) ([]MarketplaceSkill, error)`
- `var ErrQueryTooShort error`; `type APIError struct`

**Seams under test:** `Client.Search` only. The private `searchResponse`
decode struct and URL builder are collaborators.

**gock wiring (applies to every gock test in this plan):** `New()` builds an
`&http.Client{}` with nil Transport, so `http.DefaultTransport` resolves at
call time and `gock.New(base)` intercepts with no extra wiring. Tests `defer
gock.Off()` and may `gock.DisableNetworking()` so no real request can escape.
Fixture capture precedes cycle 1: run `go test -tags record -run TestRecord
./internal/marketplace` once, commit the JSON verbatim.

### RED then GREEN order (one behavior per cycle)

1. **Search decodes a recorded page into `[]MarketplaceSkill`.** Replay
   `search_react.json`; assert length equals the fixture's `count` and the
   first skill's `Name`/`SkillId`/`Source`/`Installs` equal the captured
   literals (`vercel-react-best-practices`, `vercel-labs/agent-skills`,
   `540366`). Drives the types + decode into existence.
2. **Search sends `q` and `limit`.** Strict
   `MatchParam("q","react").MatchParam("limit","25")`; a wrong or missing
   param produces no gock match and an error.
3. **Search uses a configurable base URL.** `New(WithBaseURL("http://mock.local"))`;
   `gock.New("http://mock.local")`; assert the request lands there.
4. **Empty results → empty slice, nil error.** Replay `search_empty.json`.
5. **Query under 2 chars short-circuits.** `Search(ctx, "a", 25)` returns
   `ErrQueryTooShort` and makes no request (a pending gock mock stays
   unconsumed; `gock.IsDone()` is false).
6. **Non-2xx → `*APIError`.** Replay the recorded 400 body with `.Reply(400)`
   against a valid-length query; assert `errors.As(err, &apiErr)`,
   `apiErr.StatusCode == 400`, nil result.
7. **Context cancellation aborts Search.** Delayed gock reply + cancelled ctx;
   assert `errors.Is(err, context.Canceled)`.

### Notes
- `limit <= 0` defaults to 25. Trim the query before the length check.
- Decode only what is needed (`skills`, `count`); ignore `id`, `query`,
  `searchType`, `duration_ms`.
- Read error bodies through an `io.LimitReader` cap.

### Caveats / gotchas discovered during Slice 1
- `harmonica` was deliberately NOT added here — it belongs to slice 5. Only
  `gock` (test-only, with its indirect `h2non/parth`) landed in `go.mod`.
- `New()` builds an `&http.Client{}` with a nil `Transport` so `gock` intercepts
  the default transport with no extra wiring.
- The recorded fixture's `installs` literal (`540366`) and `skillId`
  (`vercel-react-best-practices`) are the independent source of truth for decode
  assertions.

---

## Slice 2: Marketplace client — Download + tab classification

**User-observable behavior (package seam):** `Download(ctx, owner, repo,
skillId)` returns a Marketplace Skill's full file tree in one call
(`SkillFiles{Hash, Files []File{Path, Contents}}`); `Classify(files)` buckets
that tree into the four tabs exactly as `internal/skills/scanner.go` classifies
installed skills (SKILL.md at root; `references/`, `scripts/`, `assets/`
prefixes stripped for display; other root files not surfaced).

**Files:**
- Modify: `internal/marketplace/types.go` — `SkillFiles`, `File`
- Modify: `internal/marketplace/client.go` — `Download`
- Create: `internal/marketplace/classify.go` — `func Classify(files []File)
  (skillMD string, refs, scripts, assets []File)`
- Modify: `internal/marketplace/record_test.go` — capture the download
  fixtures (files array trimmed to a structurally-real subset, per-file bytes
  verbatim)
- Create: `internal/marketplace/testdata/{download_vercel-react-best-practices.json,download_unknown_404.json}`
- Modify: `internal/marketplace/client_test.go`; create `classify_test.go`

**Interfaces produced:**
- `func (c *Client) Download(ctx context.Context, owner, repo, skillId string) (SkillFiles, error)`
- `func Classify(files []File) (skillMD string, refs, scripts, assets []File)`

**Seams under test:** `Client.Download` (gock), `Classify` (pure).

### RED then GREEN order

1. **Download decodes the file tree.** Replay the recorded fixture; assert
   `Hash` equals the captured literal and `Files` contains `SKILL.md` whose
   `Contents` starts with the captured `---\nname:` prefix; assert file count.
2. **Download builds the path from owner/repo/skillId.** Strict
   `Get("/api/download/vercel-labs/agent-skills/vercel-react-best-practices")`;
   each segment `url.PathEscape`d.
3. **Download base URL is independent of search.** `New(WithDownloadBaseURL("http://dl.local"))`;
   the download lands on `dl.local` while the search base is untouched.
4. **Unknown skill → `*APIError` 404.** Replay `download_unknown_404.json`
   with `.Reply(404)`.
5. **Context cancellation aborts Download.** Delayed reply + cancel.
6. **Classify buckets like the scanner.** Hand-authored input: `SKILL.md`,
   `references/guide.md`, `references/deep/nested.md`, `scripts/run.sh`,
   `assets/logo.png`, plus root `README.md`/`metadata.json`. Assert SKILL.md
   contents returned; refs/scripts/assets populated with prefix-stripped
   display paths (nested path keeps its remainder); root extras absent.

### Notes
- `marketplace.File{Path, Contents}` is deliberately distinct from
  `skills.SkillFile{Name, Path}` (a filesystem path with no contents).
- Trimming the download fixture: keep the real SKILL.md entry byte-for-byte;
  drop the bulk of the 76 files. Record the trim rule in `record_test.go` so a
  re-record reproduces it.

### Caveats / gotchas discovered during Slice 2
- The real `vercel-react-best-practices` skill has no `references/`, `scripts/`,
  or `assets/` directories — it uses a `rules/` dir, which `Classify` drops as an
  unsurfaced root prefix. So the download fixture only exercises `Download` decode
  and the SKILL.md bucket; the other three tabs are covered by the hand-authored
  `Classify` test input, and later UI slices seed a hand-authored tree.
- `marketplace.File{Path, Contents}` is deliberately distinct from
  `skills.SkillFile{Name, Path}` (a filesystem path, no contents).

---

## Slice 3: Sort helpers

**User-observable behavior (package seam):** `SortSkills(in, field, dir)`
returns a new deterministically ordered slice: Relevance = API order (Desc =
reversed), Installs = Install Count, Name = case-insensitive; stable on ties;
never mutates its input so the caller can always re-derive the API order.

**Files:**
- Create: `internal/marketplace/sort.go` — `SortField`
  (`SortRelevance`/`SortInstalls`/`SortName`), `SortDir` (`Asc`/`Desc`),
  `SortSkills`
- Create: `internal/marketplace/sort_test.go`

**Seams under test:** `SortSkills` (pure; no gock, no network).

### RED then GREEN order

1. **`SortInstalls, Desc` orders high→low, stable on ties.** Hand-authored
   slice with a tie; tied elements keep input order.
2. **`SortInstalls, Asc` reverses the ordering.**
3. **`SortName, Asc` is case-insensitive A–Z; `SortName, Desc` is Z–A.**
4. **`SortRelevance, Asc` returns input order; `SortRelevance, Desc` reverses it.**
5. **`SortSkills` never mutates its input.** Sort by Name; assert the original
   slice is unchanged.

### Notes
- The direction toggle and the Popularity-desc default live in the UI model
  (slice 7); this package supplies only the ordering.

### Caveats / gotchas discovered during Slice 3
- `SortRelevance` is `iota` value 0, but the UI default is Popularity (Installs)
  desc per the design — the field constants carry no default, so slice 7 sets the
  default sort itself.
- `SortSkills` returns a fresh slice and never mutates its input, so Relevance is
  always re-derivable from the untouched API page.

---

## Slice 4: Add-flow entry chooser

**User-observable behavior:** `:a` opens a chooser modal (styled like the
palette) with two options: `Enter skill URL or repository` and `Search for
skills`. `j/k`/`↑↓` move, `Enter` picks, `Esc` closes. Picking "Enter…" opens
the existing Huh add wizard, byte-for-byte the already-tested manual path.
When no Marketplace client is injected, the Search option renders dimmed and
is inert (mirrors the palette's `disabled without npx` pattern).

**Files:**
- Create: `internal/app/chooser.go` — `addChooser{kind, cursor}`
  (`chooserEntry` now; `chooserPostInstall` arrives in slice 11),
  `updateChooser`, `renderChooser`
- Modify: `internal/app/model.go` — `chooser *addChooser`, `market
  *marketplace.Client`, `WithMarketplace(*marketplace.Client)` option
- Modify: `internal/app/update.go` — `handlePaletteKey` `:a` opens the chooser
  (not the wizard); route to `updateChooser` while `m.chooser != nil` (after
  the wizard block)
- Modify: `internal/app/view.go` — draw the chooser via `overlayCenter`
- Modify: `internal/app/add.go` — wizard construction moves behind the chooser
- Test: `internal/app/chooser_test.go`; update `add_test.go` helpers
  (`openWizard`) to advance through the chooser

**Interfaces produced:**
- `func WithMarketplace(c *marketplace.Client) Option`

**Seams under test:** `Model.Update`/`Model.View`.

### RED then GREEN order

1. **`:a` opens the chooser listing both options.** Assert `View()` shows
   `Enter skill URL or repository` and `Search for skills`.
2. **Picking "Enter…" reaches the Huh source prompt.** Advance the chooser
   with `enter`; assert the source input renders; the whole existing manual
   suite (SSH step, empty-source rejection, completion) passes re-driven
   through the chooser.
3. **`Esc` closes the chooser** with no wizard and no overlay.
4. **Without a Marketplace client the Search option is dimmed and inert.**
   Build the model without `WithMarketplace`; assert the dim tag renders and
   `enter` on it does nothing.

### Notes
- The chooser intercept must sit after `m.wizard != nil` in `Update` so Huh
  keeps owning all its messages.
- `updateChooser` on "Enter…" must return `m.wizard.form.Init()` (Slice 9 v1
  gotcha: Huh transitions arrive via its own cmds).

### Caveats / gotchas discovered during Slice 4
- Picking the Search option no-ops until a `*marketplace.Client` is injected —
  `chooserPick`'s `entrySearch` case returns inert without one (the overlay is
  wired in slice 5).
- The chooser intercept sits after `m.wizard != nil` in `Update` so Huh keeps
  owning all its own messages; `openWizard` in `add_test.go` had to advance
  through the chooser so the existing add suite still drives the Huh form.

---

## Slice 5: Skill Search overlay + grow animation

**User-observable behavior:** picking "Search for skills" swaps the chooser
for the Skill Search overlay, which grows from the chooser's modal size to
near-full-window on a harmonica spring — an empty shell while growing (no
Glamour/Chroma mid-tween) — then settles, stops ticking, and focuses the
search box. `Esc` from the box returns to the chooser. `cmd/trainer/main.go`
injects the real client.

**Files:**
- Create: `internal/app/skillsearch.go` — `searchOverlay` (box `textinput`,
  spinner, spring state `w/h/wVel/hVel/targetW/targetH/growing`, `zone`,
  epochs, caches), `newSearchOverlay`, `updateSearch` routing,
  `renderSearchOverlay` (shell while growing)
- Modify: `internal/app/model.go` — `search *searchOverlay`
- Modify: `internal/app/update.go` — `if m.search != nil { return
  m.updateSearch(msg) }` between the wizard and chooser blocks;
  `animFrameMsg` handling
- Modify: `internal/app/view.go` — draw the overlay via `overlayCenter`
- Modify: `internal/app/chooser.go` — Search option opens the overlay
- Modify: `cmd/trainer/main.go` — `app.WithMarketplace(marketplace.New())`
- Modify: `go.mod` — `github.com/charmbracelet/harmonica`
- Test: `internal/app/skillsearch_test.go`

**Seams under test:** `Model.Update`/`Model.View` (animation driven by sending
`animFrameMsg` through `Update`; assert on rendered size and returned cmd).

### RED then GREEN order

1. **Picking Search opens the overlay shell.** Chooser gone; a rounded shell
   with the Skill Search title renders; a frame-tick cmd is returned.
2. **Frames grow the shell and the loop stops when settled.** Feed
   `animFrameMsg`s; assert the rendered shell width/height increase between
   frames, and once within threshold the overlay snaps to target, the returned
   cmd is nil (ticker stops), and the search box renders focused.
3. **`WindowSizeMsg` mid-grow re-aims the spring.** Resize during the tween;
   assert the settled size tracks the new terminal size.
4. **`Esc` from the search box returns to the entry chooser.** Overlay cleared,
   chooser rendered.

### Notes
- Spring: `harmonica.NewSpring(harmonica.FPS(60), 6.0, 0.7)`; tick
  `tea.Tick(time.Second/60, …)`; targets `m.width-2` ×
  `m.height-frameMargin-footerHeight-1`.
- Every overlay async msg must be dropped when `m.search == nil` (guards
  messages arriving after close).

### Caveats / gotchas discovered during Slice 5
- The overlay route `if m.skillSearch != nil { return m.updateSkillSearch(msg) }`
  had to sit between the wizard and the main switch so `animFrameMsg` (a non-key
  message) reaches the overlay rather than the main key switch.
- A resize that arrives *after* the grow has settled needs its own handling: with
  `growing == false` nothing drives the spring, so the `WindowSizeMsg` case snaps
  `w/h` straight to the re-aimed target (velocities zeroed) instead of leaving the
  overlay frozen at the old size.

---

## Slice 6: Debounced Skill Search from the box

**User-observable behavior:** typing in the search box runs Skill Search
against the Marketplace: under 2 characters shows `Type at least 2
characters…` in the results pane with no request; at 2+ characters, 300ms
after the last keystroke, the ranked page renders as results. A newer
keystroke supersedes and cancels the in-flight request; a spinner shows while
a request is active and stops when results land.

**Files:**
- Modify: `internal/app/skillsearch.go` — query lifecycle (`epoch`,
  `ctxCancel`, `searchState idle|tooShort|loading|ok|empty|error`),
  `searchDebounceMsg{epoch}` → `searchCmd` → `searchResultsMsg{epoch, skills,
  err}`, spinner ticks only while loading
- Modify: `internal/app/update.go` — async msg cases
- Test: `internal/app/skillsearch_test.go` — real `marketplace.New()` client
  injected, gock replaying the slice-1 fixtures

**Seams under test:** `Model.Update`/`Model.View`; the request itself crosses
the real `Client.Search` path under gock. Debounce is proven by delivering the
`searchDebounceMsg` the returned `tea.Tick` cmd would produce.

### RED then GREEN order

1. **Under 2 chars: hint, no request.** Type `a`; assert the hint text and
   that a pending gock mock stays unconsumed.
2. **2+ chars, after debounce: results render from the fixture.** Type
   `react`, deliver the debounce msg, pump the search cmd; assert result
   names from `search_react.json` render.
3. **A superseding keystroke drops the stale cycle.** Type, then type again
   before delivering the first debounce msg; deliver the stale msg (old
   epoch); assert no request/results for the stale query.
4. **In-flight cancel.** A keystroke while a request is active cancels its
   context (the stale `searchResultsMsg` carrying `context.Canceled` is
   dropped; the newer query's results win).
5. **Spinner while loading.** Between debounce firing and results landing the
   spinner frame renders; after results it is gone.

### Notes
- Every async msg carries the epoch it was issued under; `Update` drops
  mismatches. This is behavior (stale results never render), not internals —
  assert through `View()`.
- `searchResultsMsg` handling sets `apiOrder`, applies the default sort, and
  resets `selected` to 0.

### Caveats / gotchas discovered during Slice 6
- `update.go` needed no change: the slice-5 `m.skillSearch != nil →
  updateSkillSearch(msg)` route already carries the new async messages, so the
  debounce/result/spinner `case`s live inside `updateSkillSearch`, not the
  top-level `Update` switch (the plan listed `update.go`, but the seam was
  already there).
- Every async message carries the epoch it was issued under; `updateSkillSearch`
  drops mismatches, so stale results never render — asserted through `View()`.

---

## Slice 7: Results list — rows, navigation, sorts

**User-observable behavior:** results render as two-line rows mirroring
`skillRow` — line 1 the Marketplace Skill's name, line 2
`<source> · <Install Count> installs` with comma separators
(`vercel-labs/agent-skills · 540,366 installs`) — windowed with the selected
row highlighted. `Enter`/`↓` from the box jumps to the list; `j/k` move.
Default order is Popularity desc; `r`/`p`/`n` switch sorts, the same key again
toggles direction; sort keys act only in list focus.

**Files:**
- Modify: `internal/app/skillsearch.go` — `zoneList` keys, `mktRow` +
  `commaInt`, `applySort` over `SortSkills`, sort-state (`sortKey`,
  `sortAsc`), windowing via `windowBounds`
- Modify: `internal/app/keys.go` — `mktSort` (`r,p,n`) binding
- Test: `internal/app/skillsearch_test.go`

**Seams under test:** `Model.Update`/`Model.View`.

### RED then GREEN order

1. **Row shows name and `source · 540,366 installs`.** Comma formatting
   asserted against the fixture literal.
2. **`Enter`/`↓` from the box focuses the list; `j/k` move the selection**
   (highlight band follows; windowing keeps it visible).
3. **Default order is Popularity desc.** Fixture with out-of-order installs;
   assert rendered row order high→low.
4. **`n` sorts Name A–Z; `n` again toggles Z–A; `p` returns to Popularity;
   `r` restores API order.** Verified via rendered row order; switching keys
   resets to that sort's natural direction (relevance/name asc, popularity
   desc).
5. **Sort letters are list-scoped.** In the box, `n` types into the query; the
   result order does not change.

### Notes
- `applySort` sorts a copy of `apiOrder` into the view slice and clamps
  `selected`; relevance is always re-derivable because `apiOrder` is never
  mutated (slice 3 guarantee).

### Caveats / gotchas discovered during Slice 7
- Focus became a real field `m.skillSearch.zone` (`zoneBox` default, `zoneList`
  after Enter/↓); slice 9's `zoneDetail` slots in here.
- Sort keys (`r/p/n`) are list-scoped: in the box those letters type into the
  query, in the detail they are tab keys. This zone-scoping is the reason the
  slice-1-review space-retry finding matters — a retry key must not fire in the
  box either (fixed post-review: `mktRetry` is gated on `zone != zoneBox`).
- `applySearchResults` resets to Popularity-desc with selection 0 on each landed
  page; `apiOrder` is never mutated so Relevance re-derives.

---

## Slice 8: Skill Detail — dwell download, SKILL.md first

**User-observable behavior:** resting the selection on a Marketplace Skill for
~200ms fetches its file tree with one download call and renders SKILL.md in
the detail pane (full Glamour, same look as the installed-skill tab). A
spinner shows in the content area while fetching. Moving the selection cancels
the in-flight download and resets the active tab to SKILL.md. A previously
downloaded skill re-renders from the session cache with no second request.

**Files:**
- Modify: `internal/app/skillsearch.go` — `dwellMsg{dlEpoch}` → `downloadCmd`
  → `downloadResultsMsg{dlEpoch, files, err}`; `dlState`, `dlCancel`,
  `files map[string]marketplace.SkillFiles` keyed `source@skillId`; SKILL.md
  render via `Classify` + `skills.ParseSkillMarkdown` + the existing
  `skillMarkdown`/`render.Markdown` path
- Test: `internal/app/skillsearch_test.go` — download fixture via gock

**Seams under test:** `Model.Update`/`Model.View` + real `Client.Download`
under gock.

### RED then GREEN order

1. **Dwell downloads and renders SKILL.md.** Select a result, deliver the
   dwell msg, pump the download cmd; assert a known SKILL.md substring from
   the fixture renders in the detail pane.
2. **Spinner while the download is in flight.** Between dwell and results the
   content area shows the spinner; gone after.
3. **Moving the selection cancels and resets.** Move before the download
   lands; assert the stale `downloadResultsMsg` is dropped (old skill's
   content never renders) and the tab is SKILL.md for the new selection.
4. **Cache hit makes no second request.** Re-select an already-downloaded
   skill; only one gock mock is consumed (`gock.IsDone()` with a single mock).

### Notes
- One download in flight at a time: `moveMktSelection` calls `dlCancel`, bumps
  `dlEpoch`, issues a fresh dwell tick.
- The cache is discarded with the overlay (`m.search = nil`) — never persisted.

### Caveats / gotchas discovered during Slice 8
- SKILL.md is rendered by reusing the installed-skill path exactly:
  `marketplace.Classify(sf.Files)` → `skills.ParseSkillMarkdown` →
  `skillMarkdown(...)` → `render.Markdown(md, width)`, so the Marketplace detail
  looks byte-for-byte like the on-disk browser but sources contents from memory.
- One download in flight at a time: a selection move calls `cancelDownload`,
  bumps `dlEpoch`, and issues a fresh dwell tick, so a stale `downloadResultsMsg`
  is dropped by the epoch check. The SKILL.md tab has no file list, which the
  post-review j/k-scroll fix relies on (`moveMktDetail` scrolls the viewport
  directly on `tabSkill`).

---

## Slice 9: Skill Detail tabs — lazy render + memo

**User-observable behavior:** `l` moves focus into the Skill Detail;
`i/r/s/a` switch the SKILL.md/References/Scripts/Assets tabs with the file
list and content laid out exactly like the installed-skill browser; `tab`
toggles file-list/content subfocus; `j/k` move the file selection or scroll.
Markdown references render via Glamour, scripts via Chroma, assets show `No
preview available`. Rendering is lazy (a file renders when opened) and
memoized per file. `h` returns to the list.

**Files:**
- Modify: `internal/app/skillsearch.go` — `zoneDetail` keys, overlay tab/file
  state, its own `viewport.Model`, `rendered map[string]string` memo keyed
  `source@skillId|tab|fileIdx|width`; render paths over `File.Contents`
  reusing `render.Markdown`/`render.Code` and the existing layout helpers
  (`renderTabs`, `renderFileList`, `windowBounds`, `divider`,
  `renderContentWithScrollbar`)
- Modify: `internal/app/keys.go` — `mktToDetail` (`l`), `mktToList` (`h`)
- Test: `internal/app/skillsearch_test.go`

**Seams under test:** `Model.Update`/`Model.View`.

### RED then GREEN order

1. **`l` enters the detail; `r` shows the References tab with the downloaded
   file list** (prefix-stripped names from `Classify`).
2. **A `.md` reference renders via Glamour; a script renders via Chroma;
   Assets show `No preview available`.** Assert known substrings from the
   fixture contents.
3. **`tab` toggles subfocus: `j/k` move the file selection in list subfocus
   and scroll the content in content subfocus.**
4. **Switching tabs resets the file selection and subfocus; `h` returns to
   the list zone.**
5. **Re-opening a file renders identically with no re-fetch.** Navigate away
   and back; assert the same rendered output and that gock still reports a
   single consumed download mock. (The memo map itself is internal; the
   observable is identical output and no network activity.)

### Notes
- The overlay reuses the on-disk browser's render helpers but feeds
  `File.Contents` from memory — never `os.ReadFile`, never temp files
  (ADR-0001/0002).
- Detail letters (`i/r/s/a`) are detail-zone-scoped; in the list zone `r` is a
  sort key (slice 7), in the box those letters type.

### Caveats / gotchas discovered during Slice 9
- SKILL.md (`tabSkill`) is rendered full-width with no file list and no
  scrollbar (matching the slice-8 SKILL.md pane); the file tabs get the file
  list + divider + scrollbar layout.
- Rendering is memoized per file keyed `ref|tab|fileIdx|width`; re-opening a file
  is proven by identical `View()` output and gock reporting no second download,
  never by reaching into the memo map.
- `view.go` helpers were refactored to be shared (`renderTabsFor`,
  `renderFileNames`, a free `scrollbar`) so the overlay reuses the on-disk
  browser's rendering rather than duplicating it.

---

## Slice 10: Install through the existing add seam

**User-observable behavior:** `Enter` on a Marketplace Skill (from the list or
the detail) installs it: Trainer runs `npx skills@latest add
<source>@<skillId>` via the injected `AddRunner` (prod `tea.ExecProcess`
suspends the TUI; npx runs interactively). No SSH key, no `GIT_SSH_COMMAND`.
The overlay stays open while the install runs.

**Files:**
- Modify: `internal/app/skillsearch.go` — `runAddFromSearch(ref)` building
  `actions.AddCommand(sel.InstallRef(), "")` and returning the runner's cmd
  without clearing the overlay
- Modify: `internal/app/keys.go` — `mktInstall` (`enter`)
- Test: `internal/app/skillsearch_test.go` — captured `exec.Cmd` via the
  injected `AddRunner`

**Seams under test:** `Model.Update` + injected `AddRunner` (argv/env
asserted, exactly as the existing add tests do).

### RED then GREEN order

1. **`Enter` in the list runs the add command with the exact ref.** Assert
   `cmd.Args` ends `add vercel-labs/agent-skills@vercel-react-best-practices`
   and no `GIT_SSH_COMMAND` in env.
2. **`Enter` in the detail installs the same skill.**
3. **The overlay is not cleared by the install** (unlike the manual wizard's
   `runAdd`); it remains rendered until the post-install Finish (slice 11).

### Notes
- `MarketplaceSkill.InstallRef()` (slice 1) is the single source of the ref
  format — decision 9 in the design brief.

### Caveats / gotchas discovered during Slice 10
- `runAddFromSearch` deliberately does NOT clear `m.skillSearch` (unlike the
  wizard's `runAdd`, which nils `m.wizard`); the overlay stays open so the
  post-install chooser can take over when the install command exits.
- It returns `addFinishedMsg` on completion — the same message the manual path
  uses. Post-review this message now carries the install `err` so the chooser can
  tell success from failure (the callback was previously discarding the error and
  the chooser always claimed "Skill installed").
- `AddCommand(ref, "")` passes an empty keyPath, so no `GIT_SSH_COMMAND` is set —
  asserted via the captured `exec.Cmd` env.

---

## Slice 11: Post-install chooser

**User-observable behavior:** when the install command exits with the overlay
open, a post-install chooser appears: `Find more skills` / `Finish`.
"Find more skills" returns to the search box with the query, results, and
download cache intact (no re-request for a previously downloaded skill).
"Finish" (or `Esc`) closes the overlay, rescans the disk once
(`refreshFromDisk`), and lands the browser selection on the newly installed
skill when it is findable. The manual-entry path keeps its existing
single-shot refresh.

**Files:**
- Modify: `internal/app/chooser.go` — `chooserPostInstall` kind + handling
- Modify: `internal/app/update.go` — `addFinishedMsg` branches: overlay open →
  post-install chooser (no refresh yet); otherwise existing behavior
- Modify: `internal/app/skillsearch.go` — Finish path: `refreshFromDisk`, land
  selection by name, clear overlay + caches
- Test: `internal/app/skillsearch_test.go`, `chooser_test.go`

**Seams under test:** `Model.Update`/`Model.View` with injected `AddRunner` +
rescan.

### RED then GREEN order

1. **Install completion with the overlay open shows the post-install
   chooser** and does not rescan yet (injected rescan not called).
2. **"Find more skills" returns to the box with state preserved.** The
   previous results still render; re-selecting the downloaded skill makes no
   new request (single gock mock still the only one consumed).
3. **"Finish" rescans once and closes.** Injected rescan returns a scope
   containing the installed skill; assert the overlay is gone, the browser
   renders, and the selection sits on the new skill by name (clamped
   best-effort otherwise).
4. **The manual wizard path is untouched.** `addFinishedMsg` with no overlay
   refreshes immediately, as the existing add tests already assert (they stay
   green).

### Caveats / gotchas discovered during Slice 11
- Render order was swapped in `view.go` so the overlay draws before the chooser —
  the post-install chooser sits on top of the preserved overlay.
- The post-install chooser coexists with the overlay (both non-nil). It must own
  key presses ahead of the overlay route in `update.go`. Post-review it must also
  forward *non-key* messages (async results/ticks, resizes) to the overlay
  instead of dropping them, or an in-flight download that lands during the
  chooser is lost and the detail is stranded on a frozen spinner after Find more.
- Finish rescans once via `refreshFromDisk` and lands on the installed skill by
  name; post-review it also calls `cancelSearch`/`cancelDownload` before clearing
  the overlay so no request outlives the session, and rescan/land only runs on a
  successful install.

---

## Slice 12: Empty and error states + retry

**User-observable behavior:** inline states in the owning pane, no toasts:
zero results → `No skills found for "<q>"`; search failure → `Search failed —
space to retry`; download failure → `Couldn't load files — space to retry`
(detail pane). `space` retries the failed request; it does nothing outside an
error state.

**Files:**
- Modify: `internal/app/skillsearch.go` — `searchState empty|error`,
  `dlState error`, retry handling in list and detail zones
- Modify: `internal/app/keys.go` — `mktRetry` (`space`)
- Test: `internal/app/skillsearch_test.go` — gock fixtures:
  `search_empty.json`, replies with 500/404

**Seams under test:** `Model.Update`/`Model.View` + gock.

### RED then GREEN order

1. **Zero-result query renders `No skills found for "<q>"`.** Replay
   `search_empty.json`.
2. **Search failure renders the retry hint; `space` re-fires the search.**
   First mock replies 500, second replies the good fixture; after `space` the
   results render (both mocks consumed).
3. **Download failure renders the detail retry hint; `space` retries.** Same
   two-mock pattern on the download URL.
4. **`space` is inert outside an error state** (no request made; existing
   `filterApply` binding is unaffected because it only applies in
   local-filter focus).

### Caveats / gotchas discovered during Slice 12
- `retryFailed` (space) re-fires the search on `searchError` and the download on
  `dlError`, and reports `ok=false` otherwise so space falls through. Post-review
  the space intercept is gated on `zone != zoneBox`: retry is a results-list /
  detail key per the control scheme, and in the box a space must type into the
  query (the slice-12 retry tests were updated to focus the list before pressing
  space).
- `searchEmpty` names the query (`No skills found for "<q>"`) and fires no dwell;
  `searchError` and `dlError` render the "space to retry" hints in their panes.

---

## Slice 13: Escape ladder + `/` jump

**User-observable behavior:** `Esc` steps back one level at a time: detail →
results list → search box → entry chooser → closed. `/` jumps straight to the
search box from the results list or the detail. Backing out to the chooser
cancels any in-flight request.

**Files:**
- Modify: `internal/app/skillsearch.go` — zone transitions
- Test: `internal/app/skillsearch_test.go`

**Seams under test:** `Model.Update`/`Model.View`.

### RED then GREEN order

1. **The full ladder.** Drive detail → `esc` → list → `esc` → box → `esc` →
   chooser → `esc` → closed; assert the rendered state at each step.
2. **`/` jumps to the box from the list and from the detail** (box focused,
   typing edits the query).
3. **Backing out of the overlay cancels in-flight work.** With a delayed gock
   reply pending, `esc` to the chooser; the late result never renders.

### Caveats / gotchas discovered during Slice 13
- `esc` is matched by the raw string `"esc"` at the top of the
  `tea.KeyPressMsg` case (no keymap binding), and the `/` jump reuses the
  existing Skills-pane `m.keys.search` binding rather than a new one.
- `escapeSkillSearch` only cancels in-flight work on the box→chooser rung
  (leaving the overlay); the detail→list and list→box rungs preserve state and
  any in-flight work. Backing all the way out cancels the search + download so a
  late result never renders.

---

## Slice 14: Context footer for every Skill Search zone

**User-observable behavior:** the footer stays live in the whole add flow
(spec decision 3): chip + keys per zone, overflow/pinning behavior unchanged,
`? keys` pinned. The manual Huh wizard keeps the footer hidden as today. The
`?` help modal gains a Skill Search group.

Footer contents by context:
- `ADD` (entry chooser) — `j/k select · enter choose · esc cancel`
- `SEARCH` (box) — `type to search · enter/↓ results · esc back`
- `RESULTS` (list) — `j/k select · r/p/n sort · l detail · enter install ·
  / search · esc back` (+ `space retry` only in the error state) + global tail
- `DETAIL` — `i/r/s/a tabs · tab files/content · j/k move · h list ·
  enter install · / search · esc list` (+ `space retry` in download error)
- `ADDED` (post-install chooser) — `j/k select · enter choose · esc finish`

**Files:**
- Modify: `internal/app/footer.go` — new `footerCtx` values + `footerParts`
  entries; `footerContext()` keeps `ctxHidden` for palette/help/confirm/wizard
  but resolves the chooser and overlay zones
- Modify: `internal/app/help.go` — overlay binding group
- Test: `internal/app/footer_test.go` — assert on `renderFooter()` directly
  (the established footer seam)

**Seams under test:** `Model.renderFooter()` (drive the model into each zone
with real keys first), `Model.View()` for the help modal group.

### RED then GREEN order

1. **Entry chooser footer.** `ADD` chip + its keys; none of the browse keys.
2. **Search box footer.** `SEARCH` chip; no sort keys.
3. **Results list footer.** `RESULTS` chip with sort/install/detail keys;
   `space retry` appears only in the error state.
4. **Detail footer.** `DETAIL` chip with tab/subfocus keys.
5. **Post-install footer.** `ADDED` chip.
6. **The Huh wizard still hides the footer; `?` lists the Skill Search
   bindings.**

### Notes
- All new keys come from `keys.go` bindings (single source of truth — footer
  and help render the same `key.Binding` values, the v1 invariant).

### Caveats / gotchas discovered during Slice 14
- The Skill Search zones (ADD/SEARCH/RESULTS/DETAIL/ADDED) do **not** carry the
  global tail (`move focus · commands · keys · quit`). The overlay swallows every
  key, so none of those globals function inside it; showing them would violate
  design decision 3 ("live keybindings for the focused zone"). This deviates from
  the plan's `RESULTS … + global tail` note deliberately, and matches the
  existing SEARCH/FILTER input-mode footers, which also omit the tail.
- `footerContext` checks `m.chooser` **before** `m.skillSearch` so the
  post-install chooser (both non-nil at once) names ADDED, not the overlay zone.
  The palette/help/confirm/manual-wizard still force `ctxHidden`.
- Search-error retry is reachable in the RESULTS zone: `/` jumps box→list, a
  failed query sets `searchError` in the box, and `enter/↓` carries that state
  into `zoneList`. The retry chip keys off `o.state == searchError` (RESULTS) and
  `o.dlError` (DETAIL); the SEARCH box footer omits retry (a keystroke re-searches).
- The help group lives in `keys.go` `helpGroups()`, not `help.go` — `renderHelp`
  renders whatever `helpGroups` returns, so `help.go` needed no change. The
  `TestFooterHiddenDuringModals` "wizard" case now drives `openWizard` (through
  the chooser to the Huh form), since bare `:a` opens the chooser, which is no
  longer hidden.

---

## Final slice: Full verification + manual smoke

After all slices land:

```bash
make verify            # fmt-check vet test lint (0 issues)
go build ./...
go run ./cmd/trainer   # manual smoke against the real Marketplace
```

Manual smoke checklist:
- `:a` opens the chooser; "Enter skill URL or repository" behaves exactly as
  before (manual wizard, SSH step, install, refresh).
- "Search for skills" grows the overlay smoothly and lands in the search box;
  the animation stops (no CPU churn when idle).
- One character shows the type-more hint with no network traffic; `react`
  populates results after the debounce; fast typing never flashes stale
  results.
- Rows show `source · N,NNN installs`; `p/n/r` re-order; the same key toggles
  direction; default order is Popularity desc.
- Dwelling on a result loads SKILL.md with a spinner; moving the selection
  quickly never shows the wrong skill's content; tabs render references/
  scripts/assets like the installed-skill browser.
- `Enter` suspends the TUI and runs `npx skills@latest add
  <source>@<skillId>` interactively; on exit the post-install chooser appears;
  "Find more skills" keeps results; "Finish" rescans and lands on the new
  skill.
- Kill the network: search shows `Search failed — space to retry`; retry
  recovers; download failure shows its own retry in the detail pane.
- The escape ladder walks back one level per `Esc`; `/` jumps to the box.
- The footer chip and keys track every zone; `?` lists the Skill Search group.
- Resize mid-animation and mid-browse: the frame always fits.

### Final verification result (executed)

`make verify` green (fmt-check, vet, test, lint — 0 issues) and `go build ./...`
succeeds. The binary was driven against the LIVE Marketplace (skills.sh API
reachable, HTTP 200) and the manual smoke checklist above passed:

- `:a` opens the chooser; "Enter skill URL or repository" behaves exactly as
  before (manual Huh wizard, SSH step, install, refresh).
- "Search for skills" grows the overlay smoothly and lands in the search box; the
  animation stops when settled (no idle CPU churn).
- One character shows the type-more hint with no network; `react` populates real
  results after the debounce; fast typing never flashes stale results.
- Rows show `source · N,NNN installs` (e.g. `vercel-labs/agent-skills · 540,651
  installs`); `p/n/r` re-order, the same key toggles direction, default is
  Popularity desc.
- Dwell loads SKILL.md with a spinner; moving the selection never shows the wrong
  skill's content; the detail tabs render like the installed-skill browser.
- `Enter` suspends the TUI and runs `npx skills@latest add <source>@<skillId>`;
  the post-install chooser appears on exit; Find more keeps results; Finish
  rescans and lands on the new skill.
- Killing the network shows `Search failed — space to retry` / `Couldn't load
  files — space to retry`; retry (from the list/detail) recovers.
- The escape ladder walks back one level per `Esc`; `/` jumps to the box; the
  footer chip + keys track every zone; `?` lists the Skill Search group.
- Resize mid-animation and mid-browse keeps the frame fitting.

### Caveats / gotchas discovered during Final verification

- Post-review acceptance-criteria fixes landed after the first full smoke (space
  is box-scoped-typing not retry; the post-install chooser forwards async
  messages + resizes to the preserved overlay; SKILL.md scrolls with j/k without
  a tab press; a fresh overlay shows the type-more hint; Finish cancels in-flight
  work; a failed install is titled "Install failed" with Try again / Back and
  does not rescan). Each is covered by a `Model.Update`/`Model.View` test.
