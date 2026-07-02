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
A top-level grouping of installed skills by where they are discovered, such as global, a specific agent, or a project path.
_Avoid_: tag, label, category

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
