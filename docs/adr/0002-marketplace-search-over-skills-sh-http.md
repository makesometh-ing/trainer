# Trainer searches the Marketplace over skills.sh HTTP, not by shelling out to `npx skills find`

Skill Search talks to the skills.sh Marketplace directly over HTTP — `GET /api/search?q=&limit=[&owner=]` for results and `GET /api/download/<owner>/<repo>/<skillId>` for a Marketplace Skill's full file tree — rather than executing `npx skills find`. Install still delegates to `npx skills add <source>@<skillId>` through the existing add seam. This is the first HTTP client, first debounce, and first animation (harmonica) introduced into the codebase.

## Considered Options

- Shell out to `npx skills find` and parse its stdout.
- Fetch each skill's files individually from GitHub (`git/trees` + `raw.githubusercontent.com`) for true per-file lazy loading.
- Call the skills.sh JSON API directly and use its monolithic download endpoint.

## Decision

Call the skills.sh JSON API directly. Search returns metadata only (`skillId`, `name`, `installs`, `source`); a single download call returns every file inline as JSON (`{files:[{path,contents}],hash}`), so it works for GitHub, GitLab, and self-hosted sources alike and avoids GitHub's 60-request/hour unauthenticated rate limit. The API offers no server-side sort or pagination, so ordering (Relevance / Popularity / Name) is done client-side over the returned page. Fast first view comes from rendering SKILL.md first and rendering other files only when their tab is opened — lazy rendering, not lazy fetching, because the endpoint cannot stagger.

Tests exercise the real HTTP code path against recorded fixtures replayed with `github.com/h2non/gock`, captured once from the live API — integration tests over hand-written fakes.

## Consequences

- Trainer depends on an unofficial, undocumented skills.sh API surface; if it changes, search breaks and the recorded fixtures go stale. Install is unaffected because it stays on `npx skills add`.
- The download JSON is held in a transient in-memory cache scoped to a single Skill Search session and cleared when the overlay closes. This is consistent with ADR-0001: it is not a persistent database, cache, or index — it is transient state rebuilt per session.
- The HTTP client must use a transport that `gock` can intercept (default transport / no bespoke connection pool), so the client seam is shaped around testability rather than an injected runner like the add/delete actions.
