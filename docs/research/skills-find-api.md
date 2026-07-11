# skills.sh marketplace API — raw HTTP details for a native Go search feature

Research target: `npx skills find` and `npx skills add` in [vercel-labs/skills](https://github.com/vercel-labs/skills), and the `skills.sh` API they call.

Primary sources: repo cloned at package version `1.5.15` (`package.json`), plus live `skills.sh` API responses confirmed with `curl` on 2026-07-10. File paths/line numbers below refer to the cloned repo `src/`.

TL;DR for the Go TUI:
- Search: `GET https://skills.sh/api/search?q=<query>&limit=<n>[&owner=<gh-owner>]` → metadata only.
- Full contents (per skill, lazy-loadable): `GET https://skills.sh/api/download/<owner>/<repo>/<skillId>` → full file tree with contents.
- You can build the download URL directly from a search result: `source` (= `owner/repo`) + `skillId`. No `npx` needed.

---

## 1. The `find` command source and its HTTP request

Implementation: `src/find.ts`. The network call is `searchSkillsAPI()` at `src/find.ts:86-115`.

Base URL (`src/find.ts:17`):
```ts
const SEARCH_API_BASE = process.env.SKILLS_API_URL || 'https://skills.sh';
```

The exact request (`src/find.ts:88-91`):
```ts
const params = new URLSearchParams({ q: query, limit: '10' });
if (owner) params.set('owner', owner);
const url = `${SEARCH_API_BASE}/api/search?${params.toString()}`;
const res = await fetch(url);
```

So the request is:

```
GET https://skills.sh/api/search?q=<query>&limit=10[&owner=<owner>]
```

- Method: `GET`, no headers, no auth.
- `q` — the search query (required; server rejects <2 chars, see below).
- `limit` — hardcoded to `10` by the CLI.
- `owner` — only added when `--owner` is passed. Parsed/validated in `parseFindOptions()` (`src/find.ts:45-83`); it is lower-cased and must match `^[a-z0-9](?:[a-z0-9-]{0,38})$` (`src/find.ts:43`), i.e. a GitHub-owner-shaped string.
- No pagination / offset / page-size / sort params are sent. The CLI does a client-side sort by installs (see §3).

Two entry paths in `runFind()` (`src/find.ts:325-419`):
- Non-interactive (a query given on the CLI, or not a TTY / running inside an agent): calls `searchSkillsAPI(query, owner)` once and prints results (`src/find.ts:340-368`). Install hint printed as `npx skills add <owner/repo@skill>`, and each result links to `https://skills.sh/<slug>`.
- Interactive fzf-style prompt (`runSearchPrompt`, `src/find.ts:125-304`): debounced live search (min 2 chars, debounce `Math.max(150, 350 - q.length*50)` ms, `src/find.ts:222`), calls the same `searchSkillsAPI`.

## 2. Response shape (search)

The CLI's declared type (`src/find.ts:95-102`):
```ts
const data = (await res.json()) as {
  skills: Array<{ id: string; name: string; installs: number; source: string }>;
};
```

The CLI only reads `id`, `name`, `installs`, `source`. But the **live response carries more**. Real trimmed response for `GET /api/search?q=react&limit=3`:

```json
{
  "query": "react",
  "searchType": "fuzzy",
  "skills": [
    {
      "id": "vercel-labs/agent-skills/vercel-react-best-practices",
      "skillId": "vercel-react-best-practices",
      "name": "vercel-react-best-practices",
      "installs": 540366,
      "source": "vercel-labs/agent-skills"
    },
    {
      "id": "vercel-labs/agent-skills/vercel-react-native-skills",
      "skillId": "vercel-react-native-skills",
      "name": "vercel-react-native-skills",
      "installs": 162520,
      "source": "vercel-labs/agent-skills"
    }
  ],
  "count": 3,
  "duration_ms": 351
}
```

Field meanings:
- `id` — fully-qualified id: `<owner>/<repo>/<skillId>`.
- `skillId` — the skill's slug within its repo (e.g. `vercel-react-best-practices`). This is what you pass to the download endpoint.
- `name` — display name (equals the frontmatter `name`; here identical to `skillId`).
- `installs` — integer popularity count.
- `source` — `<owner>/<repo>` (the GitHub repo the skill lives in).
- Top-level: `query` (echo), `searchType` (`"fuzzy"`), `count` (number returned), `duration_ms`.

**CRUCIAL — search returns metadata only.** No `description`, no `SKILL.md` body, no scripts/assets/resources, no repo URL beyond `source`, no version. To get any file content you must call the download endpoint (§5). There is not even a description in the search payload.

The CLI maps into its own `SearchSkill` (`src/find.ts:26-31, 104-111`): `{ name, slug: id, source, installs }` — note it aliases the API's `id` into a field it calls `slug`, and everything passes through `sanitizeMetadata()`.

## 3. Filters & sort — server-side vs client-side

Server-side (confirmed live):
- `q` — required, must be ≥2 chars. `GET /api/search?q=&limit=3` returns HTTP `400 {"error":"Query must be at least 2 characters"}`.
- `limit` — honored (`limit=2` returns 2 results, `count:2`).
- `owner` — honored; filters to repos under that owner. `owner=vercel-labs` returned only `vercel-labs/*` sources.
- Search is fuzzy (`searchType: "fuzzy"`), ranked server-side (results already come back highest-installs first for common queries).

NOT supported server-side (confirmed live — parameter present but ignored, first result unchanged):
- `offset` — ignored (no change).
- `page` — ignored (no change).
- `sort` — no effect; there is no documented sort switch. Results appear to come pre-ranked by relevance+installs.

Client-side (CLI does this itself):
- Re-sorts the returned page by `installs` descending (`src/find.ts:111`): `.sort((a, b) => (b.installs || 0) - (a.installs || 0))`.
- Truncates display to 8 (interactive, `src/find.ts:171`) or 6 (non-interactive, `src/find.ts:359`) rows.

Net: there is effectively one sort — a fuzzy-relevance ranking from the server, then a most-installed re-sort applied locally on the (max 10) returned rows. No "recent", no explicit "most-downloaded" server param. If you want most-installed ordering in Go, replicate the client-side sort on `installs`.

## 4. `npx skills add <ref>` resolution

Ref parsing: `parseSource()` in `src/source-parser.ts:239-410`. Accepted ref formats (each yields a `ParsedSource { type, url, ref?, subpath?, skillFilter? }`):

- GitHub shorthand `owner/repo` → `https://github.com/owner/repo.git` (`src/source-parser.ts:388-398`).
- `owner/repo@skill-name` → same repo, with `skillFilter=skill-name` (`src/source-parser.ts:375-386`). **This is the form `find` emits** (`src/find.ts:404`: `parseAddOptions([pkg, '--skill', skillName])`).
- `owner/repo/sub/path` → repo + `subpath` (`src/source-parser.ts:388-398`).
- Fragment ref `owner/repo#branch` or `owner/repo#branch@skill` → `ref`/`skillFilter` (`parseFragmentRef`, `src/source-parser.ts:203-243`).
- Prefix `github:owner/repo`, `gitlab:owner/repo` (`src/source-parser.ts:264-278`).
- Full GitHub URLs: `https://github.com/owner/repo`, `.../tree/<branch>`, `.../tree/<branch>/<path>` (`src/source-parser.ts:281-317`).
- GitLab URLs incl. subgroups and `/-/tree/<branch>/<path>` (`src/source-parser.ts:319-364`).
- SSH git URLs `git@host:owner/repo.git`, `ssh://...` (`getOwnerRepo`, `src/source-parser.ts:16-42`).
- Local filesystem paths (absolute / `./` / `../`) → `type: 'local'` (`src/source-parser.ts:240-249`).
- Arbitrary HTTP(S) URL not on github/gitlab → `type: 'well-known'`, resolved against `/.well-known/agent-skills/index.json` then `/.well-known/skills/index.json` (`src/source-parser.ts:400-407`, provider in `src/providers/wellknown.ts`).
- Anything else → `type: 'git'` (direct git URL, `src/source-parser.ts:409-413`).

HTTP calls made to resolve a GitHub skill — the **blob fast path** in `src/blob.ts` (`tryBlobInstall`, `src/blob.ts:446-585`), tried before any git clone:
1. `GET https://api.github.com/repos/<owner>/<repo>/git/trees/<branch>?recursive=1` (`src/blob.ts:112`), branches tried in order `HEAD, main, master` (`src/blob.ts:184`). Unauthenticated first; only retries with a token on rate-limit 403 or private-repo 401/404 (`src/blob.ts:179-219`). `Accept: application/vnd.github.v3+json`, `User-Agent: skills-cli`.
2. For each discovered `SKILL.md` path: `GET https://raw.githubusercontent.com/<owner>/<repo>/<branch>/<skillMdPath>` to read frontmatter and get the skill `name` (`src/blob.ts:374-389`).
3. `GET https://skills.sh/api/download/<owner>/<repo>/<slug>` for the full file contents (`src/blob.ts:395-413`) — see §5. `slug = toSkillSlug(frontmatter.name)`.

If any part of the blob path fails, `add` falls back to a real `git clone` (`src/git.ts`) and on-disk discovery (`discoverSkills` in `src/skills.ts:144-289`).

Slug computation (must match server), `toSkillSlug()` at `src/blob.ts:64-71`:
```ts
name.toLowerCase()
    .replace(/[\s_]+/g, '-')
    .replace(/[^a-z0-9-]/g, '')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '');
```
In practice this equals the `skillId` returned by the search API, so from a search result you can skip steps 1–2 and hit the download endpoint directly using `source` + `skillId`.

What `add` writes to disk (`src/installer.ts`):
- Canonical install dir (universal): `<base>/.agents/skills/<skill-name>/` — `getCanonicalSkillsDir` (`src/installer.ts:98-101`) joins `AGENTS_DIR='.agents'` + `SKILLS_SUBDIR='skills'` (`src/constants.ts:1-2`). `base` is `process.cwd()` for a project install or `homedir()` with `--global`.
- Per-agent dirs when a specific agent is targeted: `<base>/<agent.skillsDir>/<skill-name>/`, e.g. `.claude/skills/<name>/`, `.cursor/skills/<name>/`, `.codex/skills/<name>/`, etc. (agent table in `src/agents.ts`; dir resolution `getAgentBaseDir` `src/installer.ts:121-149`, `getInstallPath` `src/installer.ts:565-587`). Universal agents share the canonical `.agents/skills` dir and other agent dirs are symlinked to it (`src/installer.ts:174-249`).
- Inside a skill dir, the snapshot's files are written preserving their **relative paths** (`writeSkillFiles`, `src/installer.ts:977-990`): `SKILL.md`, `metadata.json`, `README.md`, `rules/*.md`, `scripts/*`, etc. The skill name is sanitized to prevent path traversal (`getInstallPath` + `isPathSafe`, `src/installer.ts:572-585`).
- The dir is cleaned and recreated on each install (`cleanAndCreateDirectory`, `src/installer.ts:163-169`).
- Lockfiles are maintained: `skills-lock.json` (`src/skill-lock.ts`) and a local lock (`src/local-lock.ts`).

Note on root-level skills: if the skill's `SKILL.md` is at the repo root, the installer keeps only `SKILL.md` (not the whole repo) to avoid dumping thousands of files (`src/blob.ts:560-568`).

## 5. Single-skill full-contents endpoint (for lazy-loading files per tab)

Yes. `fetchSkillDownload()` at `src/blob.ts:395-413`, URL built at `src/blob.ts:401`:

```
GET https://skills.sh/api/download/<owner>/<repo>/<slug>
```
(env override `SKILLS_DOWNLOAD_URL`, default `https://skills.sh`; `src/blob.ts:45`. `owner`/`repo` come from splitting `source`; each path segment is `encodeURIComponent`-ed.)

Response type (`src/blob.ts:20-28`):
```ts
interface SkillDownloadResponse {
  files: Array<{ path: string; contents: string }>;
  hash: string; // server-side skillsComputedHash
}
```

Confirmed live — `GET https://skills.sh/api/download/vercel-labs/agent-skills/vercel-react-best-practices` → HTTP 200, top-level keys exactly `["files","hash"]`, `hash` = `ca7b0c0c...`, `files` = 76 entries. The tree includes `SKILL.md`, `AGENTS.md`, `metadata.json`, `README.md`, and a `rules/` directory (`rules/_sections.md`, `rules/async-parallel.md`, …). `files[].contents` holds the full text of each file (e.g. `AGENTS.md` ~108 KB). Real trimmed example:

```json
{
  "files": [
    { "path": "SKILL.md",       "contents": "---\nname: vercel-react-best-practices\n..." },
    { "path": "metadata.json",  "contents": "{\n  \"version\": \"1.0.0\",\n  \"organization\": \"Vercel Engineering\", ... }" },
    { "path": "README.md",      "contents": "# ..." },
    { "path": "rules/async-parallel.md", "contents": "..." }
  ],
  "hash": "ca7b0c0c6e5f2750043f7f0cd72d16ac4e2abc48f9b5500d047a4b77a2506212"
}
```

`metadata.json` for that skill (full, real):
```json
{
  "version": "1.0.0",
  "organization": "Vercel Engineering",
  "date": "January 2026",
  "abstract": "Comprehensive performance optimization guide for React and Next.js ...",
  "references": ["https://react.dev", "https://nextjs.org", "..."]
}
```

So a UI can lazy-load a skill's whole file tree (SKILL.md body, scripts/, assets/, resources/, README, metadata) with one download call, keyed by `owner` + `repo` + `skillId` — all present in the search result (`source` = `owner/repo`, `skillId` = the slug). The response is flat (`path` is a repo-relative path with `/` separators); build the tree client-side by splitting on `/`.

Self-hosted exception: `zapier/connectors` serves its own download URL `https://connectors-skills.zapier.com/download/<slug>/snapshot.json` (`BLOB_ALLOWED_REPOS`, `src/blob.ts:48-53`). Everything else uses `skills.sh/api/download/...`.

## Endpoint summary (for a Go client, no npx)

| Purpose | Method + URL | Params | Returns |
|---|---|---|---|
| Search | `https://skills.sh/api/search` | `q` (≥2 chars, required), `limit`, `owner` (optional) | `{query, searchType, skills:[{id, skillId, name, installs, source}], count, duration_ms}` — metadata only |
| Full skill contents | `https://skills.sh/api/download/<owner>/<repo>/<skillId>` | path segments only | `{files:[{path, contents}], hash}` — full file tree |
| Repo tree (add fast-path) | `https://api.github.com/repos/<owner>/<repo>/git/trees/<branch>?recursive=1` | `recursive=1` | GitHub tree; used to discover SKILL.md paths |
| Raw SKILL.md (add fast-path) | `https://raw.githubusercontent.com/<owner>/<repo>/<branch>/<path>` | — | raw file text |
| Repo privacy check | `https://api.github.com/repos/<owner>/<repo>` | — | `{private: bool}` |

Notes for the Go implementation:
- Reuse the search result's `skillId` directly as the download slug (it equals `toSkillSlug(name)`); no need to re-derive it. If you ever derive it yourself, replicate `toSkillSlug` (`src/blob.ts:64-71`).
- `limit` is server-honored but `offset`/`page` are not — there is no pagination; you get one ranked page. The CLI caps at `limit=10`.
- Apply your own `installs`-descending sort if you want most-popular ordering (that is all the CLI does).
- `web page` for a skill is `https://skills.sh/<id>` (i.e. `https://skills.sh/<owner>/<repo>/<skillId>`), used for the "View the skill" link (`src/find.ts:365,412`).

---

Notes file location: `docs/research/skills-find-api.md` (matches the existing `docs/{adr,plans,specs}` convention in the trainer repo; added a `research/` sibling).
