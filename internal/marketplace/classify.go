package marketplace

import "strings"

// Classify buckets a downloaded file tree into the four Skill Detail tabs
// exactly as internal/skills/scanner.go classifies an installed skill: SKILL.md
// at the root is returned as skillMD; files under references/, scripts/ and
// assets/ are returned with those prefixes stripped to display paths (nested
// remainders preserved); every other root file (README.md, metadata.json, …)
// is dropped, since the installed-skill browser does not surface them either.
func Classify(files []File) (skillMD string, refs, scripts, assets []File) {
	for _, f := range files {
		switch {
		case f.Path == "SKILL.md":
			skillMD = f.Contents
		case strings.HasPrefix(f.Path, "references/"):
			refs = append(refs, File{Path: strings.TrimPrefix(f.Path, "references/"), Contents: f.Contents})
		case strings.HasPrefix(f.Path, "scripts/"):
			scripts = append(scripts, File{Path: strings.TrimPrefix(f.Path, "scripts/"), Contents: f.Contents})
		case strings.HasPrefix(f.Path, "assets/"):
			assets = append(assets, File{Path: strings.TrimPrefix(f.Path, "assets/"), Contents: f.Contents})
		}
	}
	return skillMD, refs, scripts, assets
}
