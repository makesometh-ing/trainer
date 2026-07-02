package skills

import (
	"bytes"
	"errors"
	"strings"

	"gopkg.in/yaml.v3"
)

type SkillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

var errNoFrontmatter = errors.New("missing frontmatter")

func ParseSkillMarkdown(content []byte) (SkillFrontmatter, string, error) {
	trimmed := bytes.TrimLeft(content, " \t\r\n")
	if !bytes.HasPrefix(trimmed, []byte("---")) {
		return SkillFrontmatter{}, string(content), errNoFrontmatter
	}

	rest := trimmed[len("---"):]
	rest = bytes.TrimLeft(rest, "\r")
	if !bytes.HasPrefix(rest, []byte("\n")) {
		return SkillFrontmatter{}, string(content), errNoFrontmatter
	}
	rest = rest[1:]

	end := findClosingFence(rest)
	if end.start < 0 {
		return SkillFrontmatter{}, string(content), errNoFrontmatter
	}

	yamlPart := rest[:end.start]
	body := rest[end.after:]

	var fm SkillFrontmatter
	if err := yaml.Unmarshal(yamlPart, &fm); err != nil {
		return SkillFrontmatter{}, string(bytes.TrimLeft(body, "\n")), err
	}

	return fm, string(bytes.TrimLeft(body, "\n")), nil
}

type fencePos struct {
	start int
	after int
}

func findClosingFence(b []byte) fencePos {
	offset := 0
	for len(b) > 0 {
		line, rest, found := bytes.Cut(b, []byte("\n"))
		if strings.TrimRight(string(line), "\r") == "---" {
			return fencePos{start: offset, after: offset + len(line) + boolToInt(found)}
		}
		if !found {
			break
		}
		consumed := len(line) + 1
		offset += consumed
		b = rest
	}
	return fencePos{start: -1, after: -1}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
