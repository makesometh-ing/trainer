package marketplace

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/h2non/gock"
)

// defaultBase is the production skills.sh base URL. Tests intercept it with
// gock; nothing here ever reaches the network.
const defaultBase = "https://skills.sh"

// TestSearchDecodesRecordedPage replays the recorded react page and asserts the
// decoded Marketplace Skills match the captured literals. The expected values
// are lifted verbatim from testdata/search_react.json — an independent source
// of truth, not recomputed the way the client decodes.
func TestSearchDecodesRecordedPage(t *testing.T) {
	defer gock.Off()
	gock.New(defaultBase).
		Get("/api/search").
		Reply(200).
		File("testdata/search_react.json")

	c := New()
	skills, err := c.Search(context.Background(), "react", 25)
	if err != nil {
		t.Fatalf("Search: unexpected error: %v", err)
	}

	if len(skills) != 25 {
		t.Fatalf("len(skills) = %d, want 25", len(skills))
	}

	got := skills[0]
	if got.Name != "vercel-react-best-practices" {
		t.Errorf("Name = %q, want %q", got.Name, "vercel-react-best-practices")
	}
	if got.SkillId != "vercel-react-best-practices" {
		t.Errorf("SkillId = %q, want %q", got.SkillId, "vercel-react-best-practices")
	}
	if got.Source != "vercel-labs/agent-skills" {
		t.Errorf("Source = %q, want %q", got.Source, "vercel-labs/agent-skills")
	}
	if got.Installs != 540366 {
		t.Errorf("Installs = %d, want %d", got.Installs, 540366)
	}
}

// TestSearchSendsQueryAndLimit asserts the request carries the exact q and
// limit params. gock only matches when both are present with these values, so a
// consumed (done) mock proves they were sent.
func TestSearchSendsQueryAndLimit(t *testing.T) {
	defer gock.Off()
	gock.New(defaultBase).
		Get("/api/search").
		MatchParam("q", "react").
		MatchParam("limit", "25").
		Reply(200).
		File("testdata/search_react.json")

	c := New()
	if _, err := c.Search(context.Background(), "react", 25); err != nil {
		t.Fatalf("Search: unexpected error: %v", err)
	}
	if !gock.IsDone() {
		t.Errorf("expected the q=react&limit=25 mock to be consumed; pending mocks remain")
	}
}

// TestSearchUsesConfigurableBaseURL points the client at a non-default origin
// and asserts the request lands there.
func TestSearchUsesConfigurableBaseURL(t *testing.T) {
	defer gock.Off()
	gock.New("http://mock.local").
		Get("/api/search").
		Reply(200).
		File("testdata/search_react.json")

	c := New(WithBaseURL("http://mock.local"))
	skills, err := c.Search(context.Background(), "react", 25)
	if err != nil {
		t.Fatalf("Search: unexpected error: %v", err)
	}
	if len(skills) != 25 {
		t.Fatalf("len(skills) = %d, want 25", len(skills))
	}
	if !gock.IsDone() {
		t.Errorf("expected the mock.local mock to be consumed")
	}
}

// TestSearchEmptyResultsYieldEmptySlice replays a zero-result page and asserts
// an empty slice with no error.
func TestSearchEmptyResultsYieldEmptySlice(t *testing.T) {
	defer gock.Off()
	gock.New(defaultBase).
		Get("/api/search").
		Reply(200).
		File("testdata/search_empty.json")

	c := New()
	skills, err := c.Search(context.Background(), "zzqqxxnonexistentskillzz", 25)
	if err != nil {
		t.Fatalf("Search: unexpected error: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("len(skills) = %d, want 0", len(skills))
	}
}

// TestSearchShortQueryMakesNoRequest asserts a one-character query returns
// ErrQueryTooShort without touching the network: the registered mock stays
// pending.
func TestSearchShortQueryMakesNoRequest(t *testing.T) {
	defer gock.Off()
	gock.New(defaultBase).
		Get("/api/search").
		Reply(200).
		File("testdata/search_react.json")

	c := New()
	skills, err := c.Search(context.Background(), "a", 25)
	if !errors.Is(err, ErrQueryTooShort) {
		t.Fatalf("err = %v, want ErrQueryTooShort", err)
	}
	if skills != nil {
		t.Errorf("skills = %v, want nil", skills)
	}
	if gock.IsDone() {
		t.Errorf("expected no request; the mock should remain pending")
	}
}

// TestSearchNon2xxYieldsAPIError replays the recorded 400 body against a
// valid-length query and asserts a typed *APIError carrying the status.
func TestSearchNon2xxYieldsAPIError(t *testing.T) {
	defer gock.Off()
	gock.New(defaultBase).
		Get("/api/search").
		Reply(400).
		File("testdata/search_too_short_400.json")

	c := New()
	skills, err := c.Search(context.Background(), "react", 25)
	if skills != nil {
		t.Errorf("skills = %v, want nil", skills)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want *APIError", err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("StatusCode = %d, want 400", apiErr.StatusCode)
	}
}

// TestSearchContextCancellationAborts cancels the context while a reply is
// delayed and asserts the call fails with context.Canceled.
func TestSearchContextCancellationAborts(t *testing.T) {
	defer gock.Off()
	gock.New(defaultBase).
		Get("/api/search").
		Reply(200).
		Delay(200 * time.Millisecond).
		File("testdata/search_react.json")

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	c := New()
	_, err := c.Search(ctx, "react", 25)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

// TestDownloadDecodesFileTree replays the recorded (trimmed) download fixture
// and asserts the decoded SkillFiles carry the captured hash and the SKILL.md
// entry. Expected values are lifted verbatim from
// testdata/download_vercel-react-best-practices.json — an independent source of
// truth, not recomputed the way the client decodes.
func TestDownloadDecodesFileTree(t *testing.T) {
	defer gock.Off()
	gock.New(defaultBase).
		Get("/api/download/vercel-labs/agent-skills/vercel-react-best-practices").
		Reply(200).
		File("testdata/download_vercel-react-best-practices.json")

	c := New()
	files, err := c.Download(context.Background(), "vercel-labs", "agent-skills", "vercel-react-best-practices")
	if err != nil {
		t.Fatalf("Download: unexpected error: %v", err)
	}

	if files.Hash != "ca7b0c0c6e5f2750043f7f0cd72d16ac4e2abc48f9b5500d047a4b77a2506212" {
		t.Errorf("Hash = %q, want the captured literal", files.Hash)
	}
	if len(files.Files) != 4 {
		t.Fatalf("len(Files) = %d, want 4", len(files.Files))
	}

	var skillMD *File
	for i := range files.Files {
		if files.Files[i].Path == "SKILL.md" {
			skillMD = &files.Files[i]
			break
		}
	}
	if skillMD == nil {
		t.Fatalf("no SKILL.md entry in the downloaded tree")
	}
	if !strings.HasPrefix(skillMD.Contents, "---\nname: vercel-react-best-practices") {
		t.Errorf("SKILL.md Contents = %q…, want the captured frontmatter prefix", firstN(skillMD.Contents, 40))
	}
}

// TestDownloadBuildsPathFromSegments asserts the request path is
// /api/download/<owner>/<repo>/<skillId>. gock only matches when the path is
// exactly that, so a consumed mock proves the URL was built correctly.
func TestDownloadBuildsPathFromSegments(t *testing.T) {
	defer gock.Off()
	gock.New(defaultBase).
		Get("/api/download/vercel-labs/agent-skills/vercel-react-best-practices").
		Reply(200).
		File("testdata/download_vercel-react-best-practices.json")

	c := New()
	if _, err := c.Download(context.Background(), "vercel-labs", "agent-skills", "vercel-react-best-practices"); err != nil {
		t.Fatalf("Download: unexpected error: %v", err)
	}
	if !gock.IsDone() {
		t.Errorf("expected the /api/download/vercel-labs/agent-skills/vercel-react-best-practices mock to be consumed")
	}
}

// TestDownloadUsesIndependentBaseURL points only the download base at a
// non-default origin and asserts the download lands there.
func TestDownloadUsesIndependentBaseURL(t *testing.T) {
	defer gock.Off()
	gock.New("http://dl.local").
		Get("/api/download/vercel-labs/agent-skills/vercel-react-best-practices").
		Reply(200).
		File("testdata/download_vercel-react-best-practices.json")

	c := New(WithDownloadBaseURL("http://dl.local"))
	files, err := c.Download(context.Background(), "vercel-labs", "agent-skills", "vercel-react-best-practices")
	if err != nil {
		t.Fatalf("Download: unexpected error: %v", err)
	}
	if len(files.Files) != 4 {
		t.Fatalf("len(Files) = %d, want 4", len(files.Files))
	}
	if !gock.IsDone() {
		t.Errorf("expected the dl.local download mock to be consumed")
	}
}

// TestDownloadUnknownSkillYieldsAPIError replays the recorded 404 body and
// asserts a typed *APIError carrying the status.
func TestDownloadUnknownSkillYieldsAPIError(t *testing.T) {
	defer gock.Off()
	gock.New(defaultBase).
		Get("/api/download/vercel-labs/agent-skills/definitely-not-a-real-skill-xyz").
		Reply(404).
		File("testdata/download_unknown_404.json")

	c := New()
	files, err := c.Download(context.Background(), "vercel-labs", "agent-skills", "definitely-not-a-real-skill-xyz")
	if files.Files != nil {
		t.Errorf("Files = %v, want nil", files.Files)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want *APIError", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}

// TestDownloadContextCancellationAborts cancels the context while a reply is
// delayed and asserts the call fails with context.Canceled.
func TestDownloadContextCancellationAborts(t *testing.T) {
	defer gock.Off()
	gock.New(defaultBase).
		Get("/api/download/vercel-labs/agent-skills/vercel-react-best-practices").
		Reply(200).
		Delay(200 * time.Millisecond).
		File("testdata/download_vercel-react-best-practices.json")

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	c := New()
	_, err := c.Download(ctx, "vercel-labs", "agent-skills", "vercel-react-best-practices")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

// firstN returns up to n runes of s, for readable failure messages.
func firstN(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
