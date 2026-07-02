package actions

import (
	"strings"
	"testing"
)

func TestUpdateCommandBuildsNpxUpdate(t *testing.T) {
	got := strings.Join(UpdateCommand().Args, " ")
	want := "npx skills@latest update"
	if got != want {
		t.Errorf("UpdateCommand args = %q, want %q", got, want)
	}
}
