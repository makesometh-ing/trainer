package marketplace

import "testing"

// mktNames returns the Name of each skill in order, for asserting the observable
// ordering a sort produced.
func mktNames(skills []MarketplaceSkill) []string {
	out := make([]string, len(skills))
	for i, s := range skills {
		out[i] = s.Name
	}
	return out
}

// assertOrder asserts the sorted slice's names match want in order. The want
// order is hand-authored, independent of how SortSkills computes the ordering.
func assertOrder(t *testing.T, got []MarketplaceSkill, want []string) {
	t.Helper()
	names := mktNames(got)
	if len(names) != len(want) {
		t.Fatalf("len = %d, want %d (%v)", len(names), len(want), names)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("order = %v, want %v", names, want)
		}
	}
}

// TestSortInstallsDescOrdersHighToLowStableOnTies sorts by Install Count
// descending and asserts high→low order with tied skills keeping their input
// order (stable). The tie is between "alpha" and "gamma" at 100 installs;
// "alpha" precedes "gamma" in the input, so it must precede it in the output.
func TestSortInstallsDescOrdersHighToLowStableOnTies(t *testing.T) {
	in := []MarketplaceSkill{
		{Name: "alpha", Installs: 100},
		{Name: "beta", Installs: 300},
		{Name: "gamma", Installs: 100},
		{Name: "delta", Installs: 200},
	}

	got := SortSkills(in, SortInstalls, Desc)

	assertOrder(t, got, []string{"beta", "delta", "alpha", "gamma"})
}

// TestSortInstallsAscReversesOrdering sorts the same slice by Install Count
// ascending and asserts low→high order, with the tie still resolved in input
// order.
func TestSortInstallsAscReversesOrdering(t *testing.T) {
	in := []MarketplaceSkill{
		{Name: "alpha", Installs: 100},
		{Name: "beta", Installs: 300},
		{Name: "gamma", Installs: 100},
		{Name: "delta", Installs: 200},
	}

	got := SortSkills(in, SortInstalls, Asc)

	assertOrder(t, got, []string{"alpha", "gamma", "delta", "beta"})
}

// TestSortNameIsCaseInsensitive sorts by Name and asserts a case-insensitive
// alphabetical order in both directions: "Banana" sorts between "apple" and
// "cherry" regardless of case (Asc = A→Z, Desc = Z→A).
func TestSortNameIsCaseInsensitive(t *testing.T) {
	in := []MarketplaceSkill{
		{Name: "cherry"},
		{Name: "apple"},
		{Name: "Banana"},
	}

	asc := SortSkills(in, SortName, Asc)
	assertOrder(t, asc, []string{"apple", "Banana", "cherry"})

	desc := SortSkills(in, SortName, Desc)
	assertOrder(t, desc, []string{"cherry", "Banana", "apple"})
}

// TestSortRelevanceUsesApiOrder asserts Relevance ascending returns the input
// (API) order untouched and Relevance descending reverses it. The input is
// deliberately unsorted by both name and installs, so returning input order
// proves Relevance ignores those fields.
func TestSortRelevanceUsesApiOrder(t *testing.T) {
	in := []MarketplaceSkill{
		{Name: "zeta", Installs: 10},
		{Name: "alpha", Installs: 90},
		{Name: "mid", Installs: 50},
	}

	asc := SortSkills(in, SortRelevance, Asc)
	assertOrder(t, asc, []string{"zeta", "alpha", "mid"})

	desc := SortSkills(in, SortRelevance, Desc)
	assertOrder(t, desc, []string{"mid", "alpha", "zeta"})
}

// TestSortSkillsNeverMutatesInput sorts a slice and asserts the caller's
// original slice is left in its input order, so the API order (SortRelevance,
// Asc) stays re-derivable after any other sort.
func TestSortSkillsNeverMutatesInput(t *testing.T) {
	in := []MarketplaceSkill{
		{Name: "cherry"},
		{Name: "apple"},
		{Name: "Banana"},
	}

	_ = SortSkills(in, SortName, Asc)

	assertOrder(t, in, []string{"cherry", "apple", "Banana"})
}
