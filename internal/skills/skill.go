package skills

import "time"

type Scope struct {
	Name string
	Path string
}

type Skill struct {
	Name        string
	Description string
	Body        string
	Path        string
	SkillPath   string
	References  []SkillFile
	Scripts     []SkillFile
	Assets      []SkillFile
	Lock        *LockEntry
	Warnings    []string
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
	Scope    Scope
	Skills   []Skill
	Warnings []string
}
