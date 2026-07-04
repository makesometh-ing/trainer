# Trainer

![Trainer browsing skills across scopes](docs/screenshot.png)

Trainer is a terminal app for browsing, inspecting, adding, and deleting the
agent skills installed on your machine. A skill is a reusable agent capability:
a directory with a `SKILL.md` file and optional supporting files. Trainer finds
every place skills live, on your machine, the shared `.agents` store and each
harness's own store (claude, codex, opencode, pi, and so on), and lets you look
through them and manage them from one screen.

It does not replace the installer. Adding, deleting, and updating skills call
`npx skills` under the hood, so those actions need Node/`npx` on your `PATH`.
Without `npx`, Trainer still browses and inspects; the add/delete/update
commands are dimmed.

## What it detects

Trainer scans the shared `.agents` skill store and the per-harness stores it
knows about. The harnesses and locations it currently detects:

| Harness | Global location | Project location |
| --- | --- | --- |
| `.agents` (shared store) | `~/.agents/skills` | `./.agents/skills` |
| claude | `~/.claude/skills` | `./.claude/skills` |
| codex | `~/.codex/skills` | (shares `.agents`) |
| opencode | `~/.config/opencode/skills` | (shares `.agents`) |
| pi | `~/.pi/agent/skills` | `./.pi/skills` |
| cursor | `~/.cursor/skills` | (shares `.agents`) |

Empty or absent locations are skipped. Support for the full set of agents that
`npx skills` handles is the first item on the roadmap.

## Install

**Homebrew (macOS and Linux):**

```sh
brew install makesometh-ing/tap/trainer
```

**Debian / Ubuntu:** download the `.deb` for your architecture from the
[latest release](https://github.com/makesometh-ing/trainer/releases/latest) and
install it:

```sh
sudo dpkg -i trainer_*_amd64.deb   # or _arm64.deb
```

**Any other Linux, or manual install:** download the `.tar.gz` for your OS and
architecture from the releases page, extract it, and move `trainer` onto your
`PATH`:

```sh
tar xzf trainer_*_Linux_amd64.tar.gz
sudo mv trainer /usr/local/bin/
```

Check the install:

```sh
trainer --version
```

## Quick start

Run `trainer` in any directory. It scans your global skill stores and, if the
current directory is a project with its own skills, that project too. Pick a
scope on the left, a skill in the middle, and read its detail on the right.

Add, delete, and update act on the scope you have selected: a global scope
targets your global skills, a project scope targets that project. Every action
rescans afterwards so the list stays true to disk.

## Keys

Press `?` at any time for the in-app list. The defaults:

**Global**

| Key | Action |
| --- | --- |
| `1` / `2` / `3` | focus Scope / Skills / Details |
| `h` / `l` | move focus left / right |
| `:` | command palette |
| `?` | toggle help |
| `q` | quit |

**Skills pane**

| Key | Action |
| --- | --- |
| `j` / `k` | move selection |
| `/` | search |
| `f` | focus the origin filter (All / Remote / Local) |
| `r` | reset search and filter |
| `h` / `l` | move filter option (when the filter is focused) |
| `space` | apply filter option (when the filter is focused) |
| `c` | clear filter (when the filter is focused) |

**Details pane**

| Key | Action |
| --- | --- |
| `i` / `r` / `s` / `a` | switch tab: SKILL.md / References / Scripts / Assets |
| `tab` | toggle between file list and content |
| `j` / `k` | move file, or scroll content |
| `ctrl+d` / `ctrl+u` | half-page scroll (from any pane) |
| `g` / `G` | top / bottom of content |

**Command palette** (open with `:`, then the letter)

| Key | Action |
| --- | --- |
| `:a` | add a skill to the selected scope |
| `:d` | delete the selected skill |
| `:u` | update all skills in the selected scope |

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for how to
build, run the tests, and send a change.

## Roadmap

1. Support the same set of agents that `npx skills` supports.
2. Custom keybinding config.
3. Manual skill creator and editor.
4. AI skill creator.

## License

MIT. See [LICENSE](LICENSE).
