package skills

import "time"

// Section groups scopes by where they live: under the user's home (Global) or
// relative to the launch directory (Project).
type Section string

const (
	SectionGlobal  Section = "Global"
	SectionProject Section = "Project"
)

type Scope struct {
	Name    string
	Section Section
	Path    string
}

type Skill struct {
	Name        string
	Description string
	Body        string
	Frontmatter string
	Path        string
	SkillPath   string
	References  []SkillFile
	Scripts     []SkillFile
	Assets      []SkillFile
	Lock        *LockEntry
}

type SkillFile struct {
	Name string
	Path string
}

type LockEntry struct {
	Source          string    `json:"source"`
	SourceType      string    `json:"sourceType"`
	SourceURL       string    `json:"sourceUrl"`
	Ref             string    `json:"ref,omitempty"`
	SkillPath       string    `json:"skillPath,omitempty"`
	SkillFolderHash string    `json:"skillFolderHash"`
	InstalledAt     time.Time `json:"installedAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
	PluginName      string    `json:"pluginName,omitempty"`
}

type ScanResult struct {
	Scope  Scope
	Skills []Skill
}
