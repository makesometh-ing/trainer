package runtime

import (
	"slices"
	"strings"
	"testing"
)

func fakeLookup(present map[string]string) LookPathFunc {
	return func(name string) (string, bool) {
		path, ok := present[name]
		return path, ok
	}
}

func fakeVersion(versions map[string]string) VersionFunc {
	return func(name string) string {
		return versions[name]
	}
}

func TestCheckReportsPresentToolsWithVersions(t *testing.T) {
	look := fakeLookup(map[string]string{
		"node": "/usr/bin/node",
		"npm":  "/usr/bin/npm",
		"npx":  "/usr/bin/npx",
	})
	version := fakeVersion(map[string]string{
		"node": "v26.4.0",
		"npm":  "11.13.0",
		"npx":  "11.13.0",
	})

	status := Check(look, version)

	if !status.NPXAvailable {
		t.Error("expected NPXAvailable when npx is on PATH")
	}
	if len(status.Missing) != 0 {
		t.Errorf("expected no missing tools, got %v", status.Missing)
	}
	if status.Node.Path != "/usr/bin/node" {
		t.Errorf("expected node path, got %q", status.Node.Path)
	}
	if status.Node.Version != "26.4.0" {
		t.Errorf("expected node version normalized to 26.4.0, got %q", status.Node.Version)
	}
	if status.NPM.Version != "11.13.0" {
		t.Errorf("expected npm version 11.13.0, got %q", status.NPM.Version)
	}
	if status.NPX.Version != "11.13.0" {
		t.Errorf("expected npx version 11.13.0, got %q", status.NPX.Version)
	}
}

func TestMissingNPXMarksAddUnavailable(t *testing.T) {
	look := fakeLookup(map[string]string{
		"node": "/usr/bin/node",
		"npm":  "/usr/bin/npm",
	})
	version := fakeVersion(map[string]string{
		"node": "26.4.0",
		"npm":  "11.13.0",
	})

	status := Check(look, version)

	if status.NPXAvailable {
		t.Error("expected NPXAvailable to be false when npx is absent")
	}
	if !slices.Contains(status.Missing, "npx") {
		t.Errorf("expected npx in Missing, got %v", status.Missing)
	}
}

func TestMissingNodeAndNPMRecordedAsMissing(t *testing.T) {
	look := fakeLookup(map[string]string{
		"npx": "/usr/bin/npx",
	})
	version := fakeVersion(map[string]string{
		"npx": "11.13.0",
	})

	status := Check(look, version)

	if !slices.Contains(status.Missing, "node") {
		t.Errorf("expected node in Missing, got %v", status.Missing)
	}
	if !slices.Contains(status.Missing, "npm") {
		t.Errorf("expected npm in Missing, got %v", status.Missing)
	}
	if !status.NPXAvailable {
		t.Error("expected NPXAvailable true even when node/npm are missing")
	}
}

func TestConfirmContinueWithoutNPX(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty defaults to no", "\n", false},
		{"n is no", "n\n", false},
		{"y is yes", "y\n", true},
		{"uppercase Y is yes", "Y\n", true},
		{"yes word is yes", "yes\n", true},
		{"junk defaults to no", "maybe\n", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			got := ConfirmContinueWithoutNPX(strings.NewReader(tc.input), &out)
			if got != tc.want {
				t.Errorf("input %q: got %v, want %v", tc.input, got, tc.want)
			}
			if !strings.Contains(out.String(), "Continue?") {
				t.Errorf("expected prompt to be written, got %q", out.String())
			}
		})
	}
}
