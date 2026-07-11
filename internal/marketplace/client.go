// Package marketplace is Trainer's HTTP client for the skills.sh Marketplace.
// It performs Skill Search (GET /api/search) and, in a later slice, downloads a
// Marketplace Skill's file tree. It is the first HTTP client in the repo; it
// holds no persistent cache and talks to an unofficial skills.sh API surface.
package marketplace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// DefaultBaseURL is the production skills.sh origin, used for both search and
// download when no override is supplied.
const DefaultBaseURL = "https://skills.sh"

// minQueryLen is the shortest query the API accepts; shorter ones are rejected
// with HTTP 400, so the client short-circuits them.
const minQueryLen = 2

// defaultLimit is the number of Marketplace Skills requested per search when
// the caller passes a non-positive limit. The API has no pagination.
const defaultLimit = 25

// errBodyCap bounds how much of an error response body is read into an APIError.
const errBodyCap = 4 << 10

// ErrQueryTooShort is returned by Search when the trimmed query is under two
// characters. No HTTP request is made in that case.
var ErrQueryTooShort = errors.New("marketplace: query must be at least 2 characters")

// APIError reports a non-2xx response from the Marketplace API.
type APIError struct {
	// StatusCode is the HTTP status returned.
	StatusCode int
	// URL is the request URL that produced the error.
	URL string
	// Body is the (capped) response body, for diagnostics.
	Body string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("marketplace: %s: unexpected status %d: %s", e.URL, e.StatusCode, strings.TrimSpace(e.Body))
}

// Client talks to the skills.sh Marketplace over HTTP. The zero value is not
// usable; construct one with New. Its http.Client has a nil Transport so
// http.DefaultTransport resolves at call time, letting gock intercept requests
// in tests.
type Client struct {
	http          *http.Client
	searchBaseURL string
	downloadBase  string
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL sets both the search and download base URLs, mirroring a single
// skills.sh origin. Individual overrides win if applied after it.
func WithBaseURL(base string) Option {
	return func(c *Client) {
		c.searchBaseURL = base
		c.downloadBase = base
	}
}

// WithSearchBaseURL overrides only the search base URL (the CLI's
// SKILLS_API_URL).
func WithSearchBaseURL(base string) Option {
	return func(c *Client) { c.searchBaseURL = base }
}

// WithDownloadBaseURL overrides only the download base URL (the CLI's
// SKILLS_DOWNLOAD_URL).
func WithDownloadBaseURL(base string) Option {
	return func(c *Client) { c.downloadBase = base }
}

// WithHTTPClient supplies the http.Client to use instead of the default.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.http = hc }
}

// New builds a Marketplace client. By default it targets DefaultBaseURL for
// both search and download and uses an http.Client with a nil Transport so
// tests can intercept the default transport.
func New(opts ...Option) *Client {
	c := &Client{
		http:          &http.Client{},
		searchBaseURL: DefaultBaseURL,
		downloadBase:  DefaultBaseURL,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// searchResponse is the decode target for GET /api/search. Only the fields
// Trainer needs are declared; the rest of the payload is ignored.
type searchResponse struct {
	Skills []MarketplaceSkill `json:"skills"`
}

// Search runs a Skill Search against the Marketplace and returns the ranked
// page of Marketplace Skills (metadata only). Queries under two characters
// return ErrQueryTooShort with no request. A non-2xx response yields an
// *APIError. The request honors ctx, so cancelling it aborts the call.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]MarketplaceSkill, error) {
	query = strings.TrimSpace(query)
	if len(query) < minQueryLen {
		return nil, ErrQueryTooShort
	}
	if limit <= 0 {
		limit = defaultLimit
	}

	u := c.searchBaseURL + "/api/search?" + url.Values{
		"q":     {query},
		"limit": {strconv.Itoa(limit)},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, apiError(resp, u)
	}

	var decoded searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("marketplace: decode search response: %w", err)
	}
	return decoded.Skills, nil
}

// Download fetches a Marketplace Skill's full file tree in one call, returning
// every file inline. Each path segment is escaped, so owners, repos and skill
// ids with reserved characters produce a valid URL. A non-2xx response (e.g. a
// 404 for an unknown skill) yields an *APIError. The request honors ctx, so
// cancelling it aborts the call.
func (c *Client) Download(ctx context.Context, owner, repo, skillId string) (SkillFiles, error) {
	u := c.downloadBase + "/api/download/" +
		url.PathEscape(owner) + "/" +
		url.PathEscape(repo) + "/" +
		url.PathEscape(skillId)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return SkillFiles{}, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return SkillFiles{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return SkillFiles{}, apiError(resp, u)
	}

	var files SkillFiles
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return SkillFiles{}, fmt.Errorf("marketplace: decode download response: %w", err)
	}
	return files, nil
}

// apiError reads a capped slice of the error body and builds an *APIError.
func apiError(resp *http.Response, reqURL string) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, errBodyCap))
	return &APIError{StatusCode: resp.StatusCode, URL: reqURL, Body: string(body)}
}
