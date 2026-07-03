# Trainer

Trainer is a focused terminal interface for browsing, inspecting, adding, and deleting agent skills installed on a developer machine.

## Language

**Trainer**:
The terminal application being built. Trainer is a single-purpose skill browser and manager.
_Avoid_: dashboard, platform, IDE

**Skill**:
A reusable agent capability packaged as a directory with a `SKILL.md` file and optional supporting files.
_Avoid_: plugin, extension, command

**Installed Skill**:
A skill that Trainer can discover in a supported skill location on the local machine.
_Avoid_: available skill, remote skill

**Global Skill**:
An installed skill available outside a single project workspace.
_Avoid_: user skill, system skill

**Scope**:
A single detected skill location that Trainer scans, shown as one selectable leaf in the Scope pane, grouped under a Global or Project section. Selecting a scope shows exactly that location's skills. Examples: the global `.agents` store, a project's claude skills.
_Avoid_: tag, label, category, folder

**Harness**:
An AI coding agent whose skills Trainer detects in that agent's own directory, such as claude, codex, opencode, pi, or cursor.
_Avoid_: tool, IDE, editor, integration

**`.agents` scope**:
The harness-independent skill store that agents share, labelled `.agents`, as opposed to a single harness's own store. It is the canonical location other harnesses symlink into, and the only scope with a skill lock; harness scopes have none.
_Avoid_: generic scope, default scope, base scope

**Skill Source**:
The origin a skill was installed from, such as a GitHub repository, Git URL, registry source, or local path.
_Avoid_: package, provider, upstream

**Skill Metadata**:
The name, description, source, source URL, source path, and local filesystem path shown to explain what a skill is and where it came from.
_Avoid_: database record, manifest

**Skill Detail**:
The focused view of one selected skill, including metadata and inspectable content.
_Avoid_: preview, inspector

**References**:
Documentation bundled with a skill for agents to read when they need additional context.
_Avoid_: docs, resources

**Scripts**:
Executable or helper files bundled with a skill.
_Avoid_: tools, commands

**Assets**:
Static files bundled with a skill, such as images, templates, schemas, and examples.
_Avoid_: resources

**Skill Lock**:
Installer-maintained source metadata used to explain where a skill came from.
_Avoid_: app database, cache
