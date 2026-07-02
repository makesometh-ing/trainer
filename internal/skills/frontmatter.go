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

// ParseSkillMarkdown splits a SKILL.md into its parsed frontmatter, the raw
// frontmatter block (verbatim, including both `---` fence lines), and the
// Markdown body below the frontmatter. When there is no valid frontmatter it
// returns an empty raw block, the whole content as the body, and errNoFrontmatter.
func ParseSkillMarkdown(content []byte) (SkillFrontmatter, string, string, error) {
	trimmed := bytes.TrimLeft(content, " \t\r\n")
	if !bytes.HasPrefix(trimmed, []byte("---")) {
		return SkillFrontmatter{}, "", string(content), errNoFrontmatter
	}

	rest := trimmed[len("---"):]
	rest = bytes.TrimLeft(rest, "\r")
	if !bytes.HasPrefix(rest, []byte("\n")) {
		return SkillFrontmatter{}, "", string(content), errNoFrontmatter
	}
	rest = rest[1:]

	end := findClosingFence(rest)
	if end.start < 0 {
		return SkillFrontmatter{}, "", string(content), errNoFrontmatter
	}

	yamlPart := rest[:end.start]
	body := rest[end.after:]

	// raw is the whole frontmatter block: the opening fence, the YAML, and the
	// closing fence, exactly as written. len(trimmed)-len(rest) is the length of
	// the opening fence consumed to reach rest; end.after is the offset in rest
	// just past the closing fence line.
	raw := string(trimmed[:(len(trimmed)-len(rest))+end.after])

	var fm SkillFrontmatter
	if err := yaml.Unmarshal(yamlPart, &fm); err != nil {
		return SkillFrontmatter{}, raw, string(bytes.TrimLeft(body, "\n")), err
	}

	return fm, raw, string(bytes.TrimLeft(body, "\n")), nil
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
