# Trainer v1 reads local skill sources without keeping its own database

Trainer v1 discovers skills from the global skill directory at startup and after add/delete actions, and reads installer-maintained lock metadata only to display source information. Trainer does not maintain its own database, cache, or persistent index; this keeps the application explainable and prevents divergence from the filesystem and `npx skills` state.

## Considered Options

- Keep an application database of scanned skills for faster rendering.
- Treat `npx skills` lock metadata as the source of truth.
- Rebuild transient in-memory state from the filesystem and lockfile whenever Trainer starts or refreshes.

## Decision

Use filesystem discovery as the source of truth for v1 and merge lockfile metadata only when present. The only persistent files Trainer reads are installed skill directories and the installer-maintained global lockfile.

## Consequences

Trainer always reflects what is actually installed. Startup and refresh perform disk reads, which is acceptable for v1 because the expected skill count is small and the UI is single-purpose.
