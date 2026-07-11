package marketplace

import "testing"

// TestClassifyBucketsLikeScanner feeds a hand-authored file tree covering all
// four tabs plus root extras and asserts Classify buckets it exactly as the
// installed-skill scanner does: SKILL.md at the root, references/, scripts/ and
// assets/ prefixes stripped for display (nested remainders kept), and other
// root files dropped. The expected display paths are hand-authored, independent
// of how Classify strips prefixes.
func TestClassifyBucketsLikeScanner(t *testing.T) {
	in := []File{
		{Path: "SKILL.md", Contents: "---\nname: demo\n---\nbody"},
		{Path: "references/guide.md", Contents: "guide"},
		{Path: "references/deep/nested.md", Contents: "nested"},
		{Path: "scripts/run.sh", Contents: "#!/bin/sh"},
		{Path: "assets/logo.png", Contents: "PNGDATA"},
		{Path: "README.md", Contents: "readme"},
		{Path: "metadata.json", Contents: "{}"},
	}

	skillMD, refs, scripts, assets := Classify(in)

	if skillMD != "---\nname: demo\n---\nbody" {
		t.Errorf("skillMD = %q, want the SKILL.md contents", skillMD)
	}

	assertDisplayPaths(t, "references", refs, []string{"guide.md", "deep/nested.md"})
	assertDisplayPaths(t, "scripts", scripts, []string{"run.sh"})
	assertDisplayPaths(t, "assets", assets, []string{"logo.png"})
}

// TestClassifyPreservesContents asserts the bucketed files keep their byte-for-byte
// contents alongside the stripped display path.
func TestClassifyPreservesContents(t *testing.T) {
	in := []File{
		{Path: "SKILL.md", Contents: "skill body"},
		{Path: "scripts/run.sh", Contents: "#!/bin/sh\necho hi"},
	}

	_, _, scripts, _ := Classify(in)
	if len(scripts) != 1 {
		t.Fatalf("len(scripts) = %d, want 1", len(scripts))
	}
	if scripts[0].Contents != "#!/bin/sh\necho hi" {
		t.Errorf("Contents = %q, want the script contents", scripts[0].Contents)
	}
}

// TestClassifyMissingSkillMD returns an empty SKILL.md string when the tree has
// no root SKILL.md, without dropping the other buckets.
func TestClassifyMissingSkillMD(t *testing.T) {
	in := []File{
		{Path: "references/guide.md", Contents: "guide"},
	}
	skillMD, refs, _, _ := Classify(in)
	if skillMD != "" {
		t.Errorf("skillMD = %q, want empty", skillMD)
	}
	if len(refs) != 1 {
		t.Errorf("len(refs) = %d, want 1", len(refs))
	}
}

func assertDisplayPaths(t *testing.T, bucket string, got []File, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: len = %d, want %d (%v)", bucket, len(got), len(want), displayPaths(got))
	}
	set := map[string]bool{}
	for _, f := range got {
		set[f.Path] = true
	}
	for _, w := range want {
		if !set[w] {
			t.Errorf("%s: missing display path %q; got %v", bucket, w, displayPaths(got))
		}
	}
}

func displayPaths(files []File) []string {
	out := make([]string, len(files))
	for i, f := range files {
		out[i] = f.Path
	}
	return out
}
