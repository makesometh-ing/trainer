package skills

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type lockFile struct {
	Version int                   `json:"version"`
	Skills  map[string]*LockEntry `json:"skills"`
}

func DefaultSkillsRoot(home string) string {
	return filepath.Join(home, ".agents", "skills")
}

func DefaultGlobalLockPath(home string) string {
	return filepath.Join(home, ".agents", ".skill-lock.json")
}

func ReadGlobalLock(path string) (map[string]LockEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]LockEntry{}, nil
		}
		return nil, err
	}

	var lf lockFile
	if err := json.Unmarshal(data, &lf); err != nil {
		return nil, err
	}

	out := make(map[string]LockEntry, len(lf.Skills))
	for name, entry := range lf.Skills {
		if entry == nil {
			continue
		}
		out[name] = *entry
	}
	return out, nil
}
