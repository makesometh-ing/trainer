//go:build record

// Package marketplace fixture recorder.
//
// This file is excluded from normal builds and CI by the `record` build tag,
// so `go test ./...` never runs it and never touches the network. Run it once,
// by hand, to (re)capture the skills.sh responses that the gock-backed tests in
// client_test.go replay:
//
//	go test -tags record -run TestRecord ./internal/marketplace
//
// It writes verbatim live response bodies into testdata/. Commit the JSON and
// do not run this again unless the skills.sh API shape changes.
package marketplace

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// recordCase names a fixture file and the search query that produces it.
type recordCase struct {
	file  string
	query string
}

// TestRecordSearchFixtures captures the three Slice 1 search fixtures verbatim
// from the live skills.sh API. Each response body is written byte-for-byte.
func TestRecordSearchFixtures(t *testing.T) {
	const base = "https://skills.sh"

	cases := []recordCase{
		// A common query with a ranked, non-empty page.
		{file: "search_react.json", query: "react"},
		// A query that matches nothing: an empty skills array, count 0.
		{file: "search_empty.json", query: "zzqqxxnonexistentskillzz"},
		// A one-character query: the API rejects it with HTTP 400.
		{file: "search_too_short_400.json", query: "a"},
	}

	for _, tc := range cases {
		u := base + "/api/search?" + url.Values{
			"q":     {tc.query},
			"limit": {"25"},
		}.Encode()

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			cancel()
			t.Fatalf("%s: build request: %v", tc.file, err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			cancel()
			t.Fatalf("%s: request: %v", tc.file, err)
		}
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel()
		if err != nil {
			t.Fatalf("%s: read body: %v", tc.file, err)
		}

		path := filepath.Join("testdata", tc.file)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatalf("%s: write: %v", tc.file, err)
		}
		t.Logf("wrote %s (%d bytes, HTTP %d)", path, len(data), resp.StatusCode)
	}
}

// keepDownloadPaths is the Slice 2 trim rule. The live download response for
// vercel-react-best-practices is 76 files (~243 KB, one 108 KB AGENTS.md). To
// keep the repo small we keep only this structurally-real subset — SKILL.md
// (the frontmatter tests assert on) plus a few real files — each byte-for-byte,
// and preserve the server hash verbatim. Editing this set re-trims on the next
// record run.
var keepDownloadPaths = map[string]bool{
	"SKILL.md":                true,
	"metadata.json":           true,
	"README.md":               true,
	"rules/async-parallel.md": true,
}

// TestRecordDownloadFixtures captures the Slice 2 download fixtures from the
// live skills.sh API. The 200 fixture is the real response with its files array
// trimmed to keepDownloadPaths (contents kept byte-for-byte, hash verbatim);
// the 404 fixture is the real not-found body. Run once by hand; commit the JSON.
func TestRecordDownloadFixtures(t *testing.T) {
	const base = "https://skills.sh"

	// The 200 case: fetch, trim files to keepDownloadPaths, keep hash verbatim.
	{
		u := base + "/api/download/vercel-labs/agent-skills/vercel-react-best-practices"
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			cancel()
			t.Fatalf("build request: %v", err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			cancel()
			t.Fatalf("request: %v", err)
		}
		var full SkillFiles
		decErr := json.NewDecoder(resp.Body).Decode(&full)
		resp.Body.Close()
		cancel()
		if decErr != nil {
			t.Fatalf("decode: %v", decErr)
		}

		trimmed := SkillFiles{Hash: full.Hash}
		for _, f := range full.Files {
			if keepDownloadPaths[f.Path] {
				trimmed.Files = append(trimmed.Files, f)
			}
		}
		data, err := json.Marshal(trimmed)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		path := filepath.Join("testdata", "download_vercel-react-best-practices.json")
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		t.Logf("wrote %s (%d files, %d bytes, HTTP %d)", path, len(trimmed.Files), len(data), resp.StatusCode)
	}

	// The 404 case: an unknown skill id, body kept verbatim.
	{
		u := base + "/api/download/vercel-labs/agent-skills/definitely-not-a-real-skill-xyz"
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			cancel()
			t.Fatalf("build request: %v", err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			cancel()
			t.Fatalf("request: %v", err)
		}
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel()
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		path := filepath.Join("testdata", "download_unknown_404.json")
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		t.Logf("wrote %s (%d bytes, HTTP %d)", path, len(data), resp.StatusCode)
	}
}
