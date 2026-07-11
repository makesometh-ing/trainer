package marketplace

import "strings"

// MarketplaceSkill is a skill offered in the Marketplace that Trainer found by
// query but that is not yet installed on the machine. Skill Search carries only
// this metadata; a Marketplace Skill's file contents come from a separate
// download call.
type MarketplaceSkill struct {
	// Name is the display name (the SKILL.md frontmatter name).
	Name string
	// SkillId is the skill's slug within its repo, e.g.
	// "vercel-react-best-practices". It is the last path segment of the
	// download URL.
	SkillId string
	// Source is "<owner>/<repo>", the repository the skill lives in.
	Source string
	// Installs is the Install Count: how many times the skill has been
	// installed, used as its popularity in Skill Search.
	Installs int
}

// InstallRef returns the reference `npx skills add` accepts to install exactly
// this skill: "<source>@<skillId>", e.g.
// "vercel-labs/agent-skills@vercel-react-best-practices".
func (s MarketplaceSkill) InstallRef() string {
	return s.Source + "@" + s.SkillId
}

// OwnerRepo splits Source into its owner and repo halves. When Source is not in
// "owner/repo" form, repo is empty and owner is the whole string.
func (s MarketplaceSkill) OwnerRepo() (owner, repo string) {
	owner, repo, _ = strings.Cut(s.Source, "/")
	return owner, repo
}

// SkillFiles is a Marketplace Skill's full file tree, returned inline by one
// download call. Hash is the server-computed content hash; Files holds every
// file with its contents.
type SkillFiles struct {
	// Hash is the server-side content hash of the file tree.
	Hash string `json:"hash"`
	// Files is the flat list of files, each with a repo-relative path and its
	// inline UTF-8 contents.
	Files []File `json:"files"`
}

// File is one file in a Marketplace Skill's tree. Path is repo-relative with
// "/" separators as returned by the download endpoint; after Classify it is the
// prefix-stripped display path within its tab. Contents is the inline UTF-8
// body. This is deliberately distinct from skills.SkillFile, which is a
// filesystem path with no contents.
type File struct {
	// Path is the file's path (repo-relative from download; display path after
	// Classify).
	Path string `json:"path"`
	// Contents is the file's inline UTF-8 body.
	Contents string `json:"contents"`
}
