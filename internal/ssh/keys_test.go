package ssh

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsSSHGitSource(t *testing.T) {
	cases := []struct {
		source string
		want   bool
	}{
		{"git@github.com:owner/repo.git", true},
		{"ssh://git@host/owner/repo.git", true},
		{"owner/repo", false},
		{"https://github.com/owner/repo.git", false},
		{"/local/path/to/skill", false},
		{"./relative/path", false},
	}
	for _, c := range cases {
		if got := IsSSHGitSource(c.source); got != c.want {
			t.Errorf("IsSSHGitSource(%q) = %v, want %v", c.source, got, c.want)
		}
	}
}

func TestFindKeyPairsReturnsOnlyValidPairs(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "id_ed25519", "PRIVATE")
	writeFile(t, dir, "id_ed25519.pub", "ssh-ed25519 AAAA")
	writeFile(t, dir, "id_rsa", "PRIVATE")
	writeFile(t, dir, "id_rsa.pub", "ssh-rsa AAAA")
	writeFile(t, dir, "known_hosts", "host key")
	writeFile(t, dir, "config", "Host *")
	writeFile(t, dir, "orphan.pub", "ssh-ed25519 BBBB")
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	pairs, err := FindKeyPairs(dir)
	if err != nil {
		t.Fatalf("FindKeyPairs returned error: %v", err)
	}

	got := map[string]bool{}
	for _, p := range pairs {
		got[filepath.Base(p.PrivatePath)] = true
	}
	if len(got) != 2 || !got["id_ed25519"] || !got["id_rsa"] {
		t.Errorf("expected id_ed25519 and id_rsa pairs, got %v", got)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
