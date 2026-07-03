package app

import "github.com/makesometh-ing/trainer/internal/skills"

// newTestModel builds a Model over a single scope. Most app tests exercise
// behavior within one scope; multi-scope tests call NewModel with a slice
// directly.
func newTestModel(result skills.ScanResult, opts ...Option) Model {
	return NewModel([]skills.ScanResult{result}, opts...)
}
