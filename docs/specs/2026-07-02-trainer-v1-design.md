# Trainer v1 Design Spec

## Goal

Build Trainer, a hyper-focused Go terminal UI for browsing, inspecting, adding, and deleting globally installed agent skills.

## Non-goals

- Do not support project-specific skills in v1.
- Do not support agent-specific skill directories in v1.
- Do not keep a Trainer-owned database, cache, or persistent index.
- Do not embed a true interactive terminal inside the TUI in v1.
- Do not depend on the external `gum` binary.
- Do not preview image or binary assets in v1.

## Source of truth

Trainer rebuilds transient in-memory state from local files at startup and after add/delete actions.

V1 scans only:

```text
~/.agents/skills/*/SKILL.md
~/.agents/.skill-lock.json
```

The filesystem determines which skills exist. The lockfile contributes source metadata when available.

The global lockfile is maintained by `npx skills`; Trainer reads it but does not treat it as its own database.

## Skill format

Trainer follows the Agent Skills specification:

```text
skill-name/
├── SKILL.md
├── references/
├── scripts/
└── assets/
```

`SKILL.md` is required for a valid skill. It contains YAML frontmatter and Markdown body content.

Trainer still lists invalid or partially broken skill directories: the scanner
continues past a malformed or frontmatter-less `SKILL.md`, listing the skill with
its directory name and an empty body. v1 neither collects nor displays warnings
for these.

## Lockfile metadata

Global lockfile path:

```text
~/.agents/.skill-lock.json
```

Relevant schema fields:

```json
{
  "version": 3,
  "skills": {
    "skill-name": {
      "source": "owner/repo",
      "sourceType": "github",
      "sourceUrl": "git@github.com:owner/repo.git",
      "ref": "main",
      "skillPath": "skills/skill-name/SKILL.md",
      "skillFolderHash": "...",
      "installedAt": "...",
      "updatedAt": "...",
      "pluginName": "..."
    }
  }
}
```

Skill list rows show `source` when the skill is present in the lockfile. Otherwise they show the literal label `local`. (The full local filesystem path is shown in the Details meta block, not in the list row.)

Skill detail headers show:

- skill name
- `source`
- `sourceUrl`
- `skillPath`
- local filesystem path

If a locked field is missing, omit that field. Always show the local filesystem path.

## UI layout

Trainer uses a three-pane TUI.

```text
┌ (1) Scope ┐ ┌ (2) Skills ───────────┐ ┌ (3) Details ──────────────────────┐
│ Global    │ │ Search  ▏             │ │ skill-name                         │
│           │ │ Filter ●All ○Remote   │ │ source     owner/repo              │
│           │ │ ○Local                │ │ sourceUrl  https://…               │
│           │ │                       │ │ skillPath  skills/…/SKILL.md       │
│           │ │ ─────────────────     │ │ path       /Users/…                │
│           │ │ skill-name            │ │ ─────────────────────────────     │
│           │ │   source or local     │ │ (i) SKILL.md (r) References …      │
│           │ │ skill-name            │ │ ─────────────────────────────     │
│           │ │   source or local     │ │ file list, when relevant       █  │
│           │ │                       │ │ ─────────────────────────────  █  │
│           │ │                       │ │ rendered content               ▓  │
└───────────┘ └───────────────────────┘ └───────────────────────────────────┘
```

The Skills pane fills the terminal from the Scope pane's right edge to the
Details pane's left edge. The Details pane fills the remaining width all the
way to the terminal's right edge, with no dead space. Within the Details pane,
the meta block, the tab bar, the optional file list, and the content are each
separated by a horizontal divider line so the sections are visually distinct. A
scrollbar is drawn down the right edge of the content when the content is taller
than the visible area.

### Full-screen and resize behavior

Trainer runs full-screen (alt screen) and fills the entire terminal by
default. The three panes reflow to the current terminal width and height on
every resize. Panes are given explicit widths and a shared explicit height so
the whole frame — including a one-row outer margin — always fits within the
terminal and never overflows past the bottom or right edge. Content that would
exceed a pane is windowed (skills list) or scrolled (detail content) rather
than growing the pane.

The scope pane has a fixed width; the remaining width is split between the
skills list and the detail pane.

If the terminal is too small to render the app usefully, Trainer replaces the
three-pane layout with a centered message such as:

```text
[Too small] Resize terminal to view the full app
```

The app returns to the normal layout as soon as the terminal is resized large
enough. The minimum threshold is width < 60 or height < 15.

### Pane and tab labels

Each pane and detail tab label includes its keyboard shortcut so shortcuts are
discoverable without opening the help modal:

- Panes: `(1) Scope`, `(2) Skills`, `(3) Details`
- Detail tabs: `(i) SKILL.md`, `(r) References`, `(s) Scripts`, `(a) Assets`

Detail tab shortcuts deliberately avoid `j`/`k` (reserved for vim-style
selection/scroll movement) so tab switching never conflicts with navigation.
The detail tab keys act only while the Details pane is focused, so the same
letters are free in the Skills pane. `r` is context-dependent: in the Details
pane it selects the References tab; in the Skills pane it resets search and
filter.

### Pane 1: Scope

V1 contains exactly one scope:

```text
Global
```

Do not render disabled future placeholders. Future scopes may be agent names such as `claude` or `codex`, or project paths.

`j` / `k` act on the focused pane's list: they move the skills selection only
while the Skills pane is focused. With a single scope they do nothing in the
Scope pane; that key moves the scope selection once there is more than one scope.

### Pane 2: Skills

The Skills pane has three parts stacked top to bottom: a search box, a filter
radio group, and the skill list. The search box and the filter group are always
drawn, even when empty, so the controls are always visible.

#### Search box

A single-line text box that narrows the skill list as the user types. Matching
is fuzzy (a skill matches when the typed characters appear in order anywhere in
its name or source). The search box is unfocused by default and shows the
placeholder `type to filter…`.

- `/` moves focus into the search box.
- Typing narrows the list immediately.
- `enter` leaves the search box, keeping the typed text and the narrowed list.
- `esc` clears the typed text and leaves the search box (the list returns to
  full).

#### Filter radio group

A single-choice radio group that narrows the skill list by where each skill
came from:

- `All` — every skill (the default).
- `Remote` — skills that have a lockfile entry (installed from a source).
- `Local` — skills with no lockfile entry (present only on disk).

The three options are laid out on one horizontal line (`Filter ●All ○Remote
○Local`), wrapping between options only when the pane is too narrow to fit them.
The selected option is filled (`●`); the rest are hollow (`○`).

- `f` moves focus into the filter group.
- `h` / `l` move the cursor left/right between the options while the group is
  focused; `j` / `k` still move the skill selection.
- `space` selects the option under the cursor.
- `c` clears the selection back to `All`.

Search and the filter combine: the list shows skills that match the search text
and the selected origin. When the Skills pane is focused, `r` resets both at
once (clears the search text and sets the filter back to `All`).

#### Skill list

The list shows the skills that pass the current search and filter. Each skill
occupies two lines:

- line 1: the skill name
- line 2: one dim metadata line containing `source` when the skill is in the
  lockfile, otherwise the literal label `local`

Both lines are truncated with an ellipsis (`…`) when they exceed the pane
width. The selected skill's two lines are highlighted with an elevated
background band and an accent-colored name; the band alone marks the selection,
so no caret or pointer is drawn on the row. The list is windowed around the
selection so it always fits the pane height without overflowing the frame.

Moving through the list immediately updates the Details pane. Pressing enter is
not required. When the search or filter changes the set of listed skills, the
selection stays on a listed skill and the Details pane follows it.

### Pane 3: Details

The Details pane stacks its sections top to bottom, each separated from the next
by a horizontal divider line so the sections read as distinct blocks:

```text
Meta block   (skill name + source fields)
─────────────
Tab bar      ((i) SKILL.md (r) References (s) Scripts (a) Assets)
─────────────
File list    (References/Scripts/Assets only, when files exist)
─────────────
Content      (rendered, with a scrollbar down the right edge)
```

#### Meta block

The meta block shows, each on its own line, truncated with an ellipsis when it
exceeds the pane width:

- the skill name
- `source` (when the skill is in the lockfile)
- `sourceUrl` (when present)
- `skillPath` (when present)
- the local filesystem `path` (always)

The description is not shown in the meta block. The full frontmatter, including
the description, appears in the `SKILL.md` content instead (see below), so
repeating it here would be redundant.

#### Tabs

- `i` — `SKILL.md`
- `r` — References
- `s` — Scripts
- `a` — Assets

#### SKILL.md content

The `SKILL.md` tab shows the whole `SKILL.md` file: its YAML frontmatter first,
then the rendered Markdown body. The frontmatter is shown verbatim, including
its opening and closing `---` fence lines and every field, not just name and
description. It is shown as a fenced YAML block so the Markdown renderer treats
the `---` lines as literal text rather than as horizontal rules (rendering them
as horizontal rules is what previously dropped the fence lines). The body below
it renders as Markdown as normal. `SKILL.md` has no file list.

#### File list and content for References / Scripts / Assets

References, Scripts, and Assets always show a file list above the content; it
reads `No files` when the skill bundles none of that kind. The selected file is
marked with an elevated highlight band and accent name, the same cue the skill
list uses, not a caret. `tab` toggles which of the two, the file list or the
content, `j` / `k` act on. The section that `j` / `k` currently act on is marked
by drawing its divider line in the accent color; the other divider is dim. There
is no separate `Files` or `Content` text header and no scroll percentage; the
divider lines and the scrollbar carry that information instead.

Assets are list-only in v1. The selected asset content area shows:

```text
No preview available
```

#### Content scrollbar

When the rendered content is taller than the visible content area, a scrollbar
is drawn down the right edge of the content. The scrollbar is a vertical bar the
height of the content area, with a solid segment whose length is the fraction of
the content that is visible and whose position tracks how far the content is
scrolled. When all the content fits, no scrollbar is drawn.

## Navigation

Global keys:

- `1` — focus Scope pane
- `2` — focus Skills pane
- `3` — focus Details pane
- `?` — show the help modal
- `:` — open command palette
- `q` — quit

Skills pane:

- `j` / `k` — move selection down/up
- `h` — move focus to the pane on the left
- `l` / `enter` — move focus to the pane on the right
- `/` — focus the search box (acts only while the Skills pane is focused)
- `f` — focus the filter radio group (acts only while the Skills pane is focused)
- `r` — reset the search text and the filter to `All`

`/` and `f` are Skills-pane keys, the same way the tab keys `i` / `r` / `s` / `a`
are Details-pane keys: they do nothing from the Scope or Details pane. Focus the
Skills pane first (`2`) to use them.

While the search box is focused:

- typing narrows the list
- `enter` — leave the search box, keeping the text and the narrowed list
- `esc` — clear the text and leave the search box

While the filter group is focused:

- `h` / `l` — move the cursor between `All` / `Remote` / `Local`
- `j` / `k` — move the skill selection (unchanged)
- `space` — select the option under the cursor
- `c` — clear back to `All`
- `esc` / `enter` — leave the filter group

Pane focus moves one pane at a time and clamps at the edges (`h` on the Scope
pane and `l` on the Details pane are no-ops).

Details pane (all of these keys act only while the Details pane is focused):

- `i` — show `SKILL.md` tab
- `r` — show References tab (in the Skills pane, `r` resets search and filter)
- `s` — show Scripts tab
- `a` — show Assets tab
- `tab` — toggle whether `j` / `k` act on the file list or the content, for tabs
  with a file list
- `j` / `k` — move the selected file when the file list is active
- `j` / `k` — scroll content one line when the content is active
- `ctrl+d` / `ctrl+u` — scroll content half-page down/up
- `ctrl+f` / `ctrl+b` — scroll content full-page down/up
- `g` / `G` — jump content to top/bottom (single `g`, not a `gg` sequence)
- `h` / `l` — navigate panes, not detail subfocus

Command palette:

- `:` — open command palette
- `a` — add skill
- `d` — delete selected skill
- `u` — update all skills (runs `npx skills@latest update`)
- `esc` — close command palette or modal

### Help modal

`?` opens a modal that lists the key bindings grouped by context (global, Skills
pane, Details pane, command palette). The bindings are defined once in `keys.go`
as a `keymap` of `key.Binding` values (each carrying its real keys and its help
label). The handlers match against them with `key.Matches` and the modal renders
the same bindings, so the keys shown and the keys handled are one definition and
cannot list different keys. `esc` or `?` closes it.

## Startup dependency check

Before starting the TUI, Trainer checks whether `node`, `npm`, and `npx` are available on `PATH`.

If all are available, Trainer prints the detected versions before launching the app. Example:

```text
node 26.4.0
npm 11.13.0
npx 11.13.0
```

If `npx` is missing, Trainer warns that adding skills is unavailable and asks whether to continue. (A missing `node` or `npm` is printed as `<name> not found` but does not prompt, since the add / update / delete actions only shell out to `npx`.) The prompt:

```text
npx is not available. Adding skills will be disabled.
Continue? [y/N]
```

If the user declines, Trainer exits before opening the TUI. If the user continues, the app opens in browse/delete mode and `:a` is disabled with an explanatory message. Delete of lockfile-backed skills also requires `npx`; if `npx` is unavailable, Trainer should disable lockfile-backed deletion and explain why. Direct deletion of skills not present in the lockfile can still work after confirmation.

## Add flow

`:a` starts an add wizard.

Step 1 asks for a skill source. Supported source formats are delegated to `npx skills`, including:

- GitHub shorthand: `owner/repo`
- full GitHub URL
- direct path to a skill in a repo
- GitLab URL
- arbitrary Git URL
- SSH Git URL
- local path

Step 2 appears only when the source looks like an SSH Git URL and there are two or more usable SSH key pairs in `~/.ssh`.

Usable SSH key pairs are detected by scanning `~/.ssh` for private key files with matching `.pub` files, excluding non-key files such as `known_hosts`, `config`, and public keys themselves.

If an SSH key is selected, Trainer prefixes the add command with `GIT_SSH_COMMAND`.

Step 3 suspends the TUI and runs interactive `npx skills` directly in the terminal.

Without selected SSH key:

```bash
npx skills add <source> -g
```

With selected SSH key:

```bash
GIT_SSH_COMMAND="ssh -i <key-path>" npx skills add <source> -g
```

Trainer does not pass agent flags. The user can make all standard `npx skills` choices during execution. After the command exits, Trainer resumes and refreshes from disk. Exit code does not prevent refresh.

## Delete flow

`:d` starts delete confirmation for the selected skill.

The confirmation asks `Delete <skill-name>?` and explains that this removes the
skill from the global skills directory and may leave broken symlinks. `y`
confirms; any other key cancels.

If the selected skill has lockfile metadata, run:

```bash
npx skills remove -g <skill-name>
```

If the selected skill is not present in the lockfile, remove the selected skill directory directly after confirmation.

After deletion, Trainer refreshes from disk.

## Update flow

`:u` updates all installed skills. It suspends the TUI and runs interactive
`npx skills@latest update` directly in the terminal, the same suspend-and-run
mechanism the add flow uses, so the user sees and can answer any prompts. After
the command exits, Trainer resumes and refreshes from disk. Exit code does not
prevent the refresh.

Update requires `npx`. When `npx` is unavailable, `:u` is disabled and shows an
explanatory message, the same way `:a` is.

## Rendering

Markdown rendering:

- Use Glamour for `SKILL.md` and Markdown references.
- Show the `SKILL.md` frontmatter, do not strip it. The scanner keeps the raw
  frontmatter block (both `---` fence lines and every field, verbatim) alongside
  the body. The `SKILL.md` tab wraps that raw block in a fenced YAML code block
  and places it above the Markdown body, then renders the whole thing through
  Glamour. Wrapping it in a code block keeps Glamour from treating the `---`
  lines as horizontal rules, so the fences and all fields are shown.
- Use a custom Gruvbox Dark Hard Glamour style built from the theme palette
  (Glamour ships no Gruvbox style, so it is configured via a custom
  `ansi.StyleConfig`). Do not use Glamour's built-in `dark` style.
- Use word wrapping based on the detail content width (pane width minus its
  border and padding).
- Render content with no left margin, so it is flush with the left edge of the
  content area and aligned with the section dividers above it. The Glamour
  document margin and the code-block margin are both zero; the frontmatter YAML
  block therefore sits flush left rather than indented.
- Trim the leading and trailing blank lines that renderers frame content with,
  so the content sits flush under its divider (no gap above the frontmatter) and
  the scrollbar reaches the bottom when the last line of real text is in view.

Script rendering:

- Use Chroma for syntax highlighting by file extension.
- Use the `gruvbox` Chroma style with the `terminal256` formatter.
- Fall back to plain text for unknown extensions.

Asset rendering:

- List files only.
- Show `No preview available` for selected asset content.

## Technology stack

- Go 1.26.4
- golangci-lint 2.12.2 as the project linter
- Bubble Tea v2 for the application model and update loop
- Bubbles v2 for the interactive primitives:
  - `viewport` for the scrollable Details content
  - `textinput` for the search box
  - `help` and `key` for the `?` help modal and its binding definitions
- Huh v2 for the add-skill wizard form (a source input and a conditional
  SSH-key select), embedded in the Bubble Tea model
- `github.com/sahilm/fuzzy` for fuzzy search matching (the same library the
  Bubbles `list` component uses internally)
- Lip Gloss v2 for layout and styling
- Glamour v2 for Markdown rendering
- Chroma for syntax highlighting

The command palette and delete confirmation are built with plain key handling
over Lip Gloss. The add-skill wizard is a Huh v2 form embedded in the Bubble Tea
model and driven through `Model.Update`; it is rendered as a centered modal
overlay (like the command palette) and themed to the Gruvbox palette from the
app's theme struct (Huh's default theme is indigo and fuchsia, which the theme
override replaces).

Prefer an existing, well-maintained dependency over a hand-built widget whenever
one fits. Where no off-the-shelf component fits (the filter radio group and the
content scrollbar), build the minimum on top of the components' public data
(for example the `viewport`'s line counts and scroll fraction) rather than
reimplementing what a component already does.

The Go module path is:

```text
github.com/makesometh-ing/trainer
```

## Theme

Default theme is Gruvbox Dark Hard using colors from `morhetz/gruvbox`.

Palette:

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

Initial mapping:

- background: `#1d2021`
- panel background: `#282828`
- elevated panel background: `#3c3836`
- foreground: `#ebdbb2`
- muted: `#928374`
- active/accent: `#fabd2f`
- secondary accent: `#83a598`
- warning/error: `#fb4934`
- success: `#b8bb26`
- border: `#504945`
- active border: `#fe8019`

Theme should be represented by a small internal struct so future config can swap palettes, but v1 has no user-facing theme configuration.

## Proposed architecture

```text
cmd/trainer/main.go
internal/app/model.go
internal/app/update.go
internal/app/view.go
internal/app/keymap.go
internal/app/keys.go          # help-modal binding definitions
internal/app/help.go          # ? help modal
internal/app/theme.go
internal/app/search.go        # search box + origin filter + visible-skills
internal/app/add.go           # add wizard (Huh form) + run/refresh
internal/app/huhtheme.go      # Gruvbox theme for the Huh form
internal/app/delete.go        # delete confirm + strategy dispatch
internal/app/updateskills.go  # :u update flow
internal/skills/scanner.go
internal/skills/skill.go
internal/skills/lockfile.go
internal/skills/frontmatter.go
internal/render/markdown.go
internal/render/code.go
internal/render/gruvbox.go    # Gruvbox Glamour StyleConfig
internal/render/trim.go       # trim surrounding blank lines
internal/actions/add.go
internal/actions/update.go
internal/actions/delete.go
internal/runtime/dependencies.go
internal/ssh/keys.go
```

Responsibilities:

- `cmd/trainer/main.go`: program entrypoint, dependency check, scan, runner wiring
- `internal/app`: Bubble Tea model, input handling, rendering layout, modal state
  (command palette, add wizard, delete confirm, `?` help), search/filter
- `internal/skills`: filesystem scan, frontmatter parsing, lockfile merge
- `internal/render`: Markdown, code, and Gruvbox styling
- `internal/actions`: add / update / delete command construction and execution
- `internal/runtime`: startup dependency detection for `node`, `npm`, and `npx`
- `internal/ssh`: SSH key-pair detection

## Verification

Run:

```bash
go test ./...
go vet ./...
golangci-lint run
```

Test coverage should include:

- scanning temporary skill directories
- parsing valid and invalid `SKILL.md` files
- reading and merging `.skill-lock.json`
- SSH key-pair detection
- startup dependency detection for `node`, `npm`, and `npx`
- add command construction with and without selected SSH key
- delete strategy selection for skills in the lockfile vs. skills only on disk
- basic Bubble Tea update behavior for pane focus, tab selection, and skill selection
- search narrows the skill list and `esc` restores it in full
- the origin filter narrows the list to `Remote` or `Local` and `c` clears it
- `r` in the Skills pane resets both search and filter
- the `SKILL.md` tab shows the frontmatter fences and a frontmatter field that
  is neither name nor description
- the Details pane fills the full terminal width (the joined frame width equals
  the terminal width)
- the content scrollbar appears only when content overflows the visible area
- the `?` help modal lists the key bindings
- `:u` builds the `npx skills@latest update` command and is disabled without `npx`
