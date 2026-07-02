package ssh

import (
	"os"
	"path/filepath"
	"strings"
)

// KeyPair is a usable SSH key pair: a private key file with a matching .pub.
type KeyPair struct {
	Name        string
	PrivatePath string
	PublicPath  string
}

// IsSSHGitSource reports whether source looks like an SSH Git URL, either the
// scp-like form (git@host:owner/repo.git) or the ssh:// scheme.
func IsSSHGitSource(source string) bool {
	if strings.HasPrefix(source, "ssh://") {
		return true
	}
	at := strings.Index(source, "@")
	colon := strings.Index(source, ":")
	return at > 0 && colon > at
}

// FindKeyPairs scans dir for private key files that have a matching .pub file,
// returning them sorted by name. Files without a private counterpart (a lone
// .pub), and non-key files such as known_hosts and config, are ignored.
func FindKeyPairs(dir string) ([]KeyPair, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	present := map[string]bool{}
	for _, e := range entries {
		if !e.IsDir() {
			present[e.Name()] = true
		}
	}

	var pairs []KeyPair
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".pub") {
			continue
		}
		if !present[name+".pub"] {
			continue
		}
		pairs = append(pairs, KeyPair{
			Name:        name,
			PrivatePath: filepath.Join(dir, name),
			PublicPath:  filepath.Join(dir, name+".pub"),
		})
	}
	return pairs, nil
}
