package marketplace

import (
	"sort"
	"strings"
)

// SortField is the field Skill Search orders Marketplace Skills by.
type SortField int

const (
	// SortRelevance keeps the Marketplace's own ranking (the API order the
	// skills arrived in); Desc reverses it.
	SortRelevance SortField = iota
	// SortInstalls orders by Install Count.
	SortInstalls
	// SortName orders by Name, case-insensitively.
	SortName
)

// SortDir is the direction an ordering runs in.
type SortDir int

const (
	// Asc is ascending order.
	Asc SortDir = iota
	// Desc is descending order.
	Desc
)

// SortSkills returns a new slice of the given Marketplace Skills ordered by the
// field and direction. It never mutates its input, so the caller can always
// re-derive the original API order. The ordering is stable: skills that compare
// equal keep their input order.
func SortSkills(in []MarketplaceSkill, field SortField, dir SortDir) []MarketplaceSkill {
	out := make([]MarketplaceSkill, len(in))
	copy(out, in)

	switch field {
	case SortRelevance:
		if dir == Desc {
			for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
				out[i], out[j] = out[j], out[i]
			}
		}
	case SortInstalls:
		sort.SliceStable(out, func(i, j int) bool {
			if dir == Desc {
				return out[i].Installs > out[j].Installs
			}
			return out[i].Installs < out[j].Installs
		})
	case SortName:
		sort.SliceStable(out, func(i, j int) bool {
			a := strings.ToLower(out[i].Name)
			b := strings.ToLower(out[j].Name)
			if dir == Desc {
				return a > b
			}
			return a < b
		})
	}

	return out
}
