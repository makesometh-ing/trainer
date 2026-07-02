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

Trainer should still show invalid or partially broken skill directories when possible, with warnings in the UI. This is low priority for v1.

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
      "updatedAt": "..."
    }
  }
}
```

Skill list rows show `source` when the skill is present in the lockfile. Otherwise they show the local filesystem path.

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
┌ Scope ┐ ┌ Skills ───────────────┐ ┌ Detail ───────────────────────────┐
│Global │ │ skill-name            │ │ skill-name                         │
│       │ │ source/path metadata  │ │ source / sourceUrl / skillPath/path│
│       │ │ ...                   │ │ [a SKILL] [b Refs] [c Scripts] [d Assets]
│       │ │                       │ │ file list, when relevant           │
│       │ │                       │ │ rendered content                   │
└───────┘ └───────────────────────┘ └───────────────────────────────────┘
```

### Pane 1: Scope

V1 contains exactly one scope:

```text
Global
```

Do not render disabled future placeholders. Future scopes may be agent names such as `claude` or `codex`, or project paths.

### Pane 2: Skills

The skill list shows discovered global skills.

Each row includes:

- skill name
- one dim metadata line containing `source` when the skill is in the lockfile, otherwise the local path
- local path when space allows

Moving through the list immediately updates the detail pane. Pressing enter is not required.

### Pane 3: Detail

The detail pane is vertical:

```text
Skill title
Source metadata
Tabs
File list, for References/Scripts/Assets
Content
```

Tabs:

- `a` — `SKILL.md`
- `b` — References
- `c` — Scripts
- `d` — Assets

`SKILL.md` has no file list. References, Scripts, and Assets show a file list above content when files exist.

Assets are list-only in v1. The selected asset content area shows:

```text
No preview available
```

## Navigation

Global keys:

- `1` — focus scope pane
- `2` — focus skill list pane
- `3` — focus detail pane
- `/` — show shortcuts modal
- `:` — open command palette
- `q` — quit

Skill list pane:

- `j` / `k` — move selection down/up
- `h` — move to scope pane
- `l` — move to detail pane

Detail pane:

- `a` — show `SKILL.md` tab
- `b` — show References tab
- `c` — show Scripts tab
- `d` — show Assets tab
- `tab` — toggle subfocus between file list and content for tabs with file lists
- `j` / `k` — move selected file when file-list subfocus is active
- `j` / `k` — scroll content one line when content subfocus is active
- `ctrl+d` / `ctrl+u` — scroll content half-page down/up
- `ctrl+f` / `ctrl+b` — scroll content full-page down/up
- `gg` / `G` — jump content to top/bottom
- `h` / `l` — navigate panes, not detail subfocus

Command palette:

- `:` — open command palette
- `a` — add skill
- `d` — delete selected skill
- `esc` — close command palette or modal

## Startup dependency check

Before starting the TUI, Trainer checks whether `node`, `npm`, and `npx` are available on `PATH`.

If all are available, Trainer prints the detected versions before launching the app. Example:

```text
node 26.4.0
npm 11.13.0
npx 11.13.0
```

If any dependency is missing, Trainer warns that adding skills is unavailable and asks whether to continue:

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

The confirmation explains:

- the selected skill will be removed
- symlinks may break
- this action affects the installed global skill directory

If the selected skill has lockfile metadata, run:

```bash
npx skills remove -g <skill-name>
```

If the selected skill is not present in the lockfile, remove the selected skill directory directly after confirmation.

After deletion, Trainer refreshes from disk.

## Rendering

Markdown rendering:

- Use Glamour for `SKILL.md` and Markdown references.
- Use word wrapping based on detail pane width.

Script rendering:

- Use Chroma for syntax highlighting by file extension.
- Fall back to plain text for unknown extensions.

Asset rendering:

- List files only.
- Show `No preview available` for selected asset content.

## Technology stack

- Go 1.26.4
- golangci-lint 2.12.2 as the project linter
- Bubble Tea v2 for the application model and update loop
- Bubbles for viewport, text input, list, and help primitives
- Lip Gloss v2 for layout and styling
- Glamour v2 for Markdown rendering
- Huh v2 for modal forms, select prompts, and confirmations
- Chroma for syntax highlighting

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
internal/app/theme.go
internal/skills/scanner.go
internal/skills/skill.go
internal/skills/lockfile.go
internal/skills/frontmatter.go
internal/render/markdown.go
internal/render/code.go
internal/actions/add.go
internal/actions/delete.go
internal/runtime/dependencies.go
internal/ssh/keys.go
```

Responsibilities:

- `cmd/trainer/main.go`: program entrypoint
- `internal/app`: Bubble Tea model, input handling, rendering layout, modal state
- `internal/skills`: filesystem scan, frontmatter parsing, lockfile merge
- `internal/render`: Markdown and code rendering
- `internal/actions`: add/delete command construction and execution
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
