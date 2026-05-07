package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	fetchMaxBodyBytes = 1 << 20
	fetchTimeout      = 20 * time.Second
	fetchMaxRetries   = 3
)

// RealFetchTool makes actual HTTP GET requests. It replaces the mock FetchTool and maps to task type "fetch".
type RealFetchTool struct {
	client *http.Client
}

func NewRealFetchTool() *RealFetchTool {
	return &RealFetchTool{
		client: &http.Client{
			Timeout: fetchTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("fetch: too many redirects")
				}
				return nil
			},
		},
	}
}

func (t *RealFetchTool) Name() string { return "fetch" }

func (t *RealFetchTool) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	url, err := requireString(input, "url")
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}

	var lastErr error
	for attempt := 1; attempt <= fetchMaxRetries; attempt++ {
		result, err := t.doFetch(ctx, url)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if attempt < fetchMaxRetries {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * 300 * time.Millisecond):
			}
		}
	}
	return nil, fmt.Errorf("fetch: %w", lastErr)
}

func (t *RealFetchTool) doFetch(ctx context.Context, url string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "llm-orchestrator/1.0")
	req.Header.Set("Accept", "text/html,application/json,text/plain;q=0.9,*/*;q=0.8")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, fetchMaxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}

	contentType := resp.Header.Get("Content-Type")
	wordCount := len(strings.Fields(string(body)))

	return map[string]any{
		"url":          url,
		"status_code":  resp.StatusCode,
		"content_type": contentType,
		"body":         string(body),
		"word_count":   wordCount,
	}, nil
}

type GitHubIssueTool struct {
	client *http.Client
	token  string
}

func NewGitHubIssueTool(token string) *GitHubIssueTool {
	return &GitHubIssueTool{
		token:  token,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (t *GitHubIssueTool) Name() string { return "github_issue" }

func (t *GitHubIssueTool) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	owner, err := requireString(input, "owner")
	if err != nil {
		return nil, fmt.Errorf("github_issue: %w", err)
	}
	repo, err := requireString(input, "repo")
	if err != nil {
		return nil, fmt.Errorf("github_issue: %w", err)
	}
	number, err := requireString(input, "number")
	if err != nil {
		return nil, fmt.Errorf("github_issue: %w", err)
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%s", owner, repo, number)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("github_issue: build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if t.token != "" {
		req.Header.Set("Authorization", "Bearer "+t.token)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github_issue: http do: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 512<<10))
	if err != nil {
		return nil, fmt.Errorf("github_issue: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github_issue: HTTP %d: %s", resp.StatusCode, truncateBytes(raw, 200))
	}

	return parseGitHubIssue(raw)
}

func parseGitHubIssue(raw []byte) (map[string]any, error) {
	var issue struct {
		Number    int    `json:"number"`
		Title     string `json:"title"`
		Body      string `json:"body"`
		State     string `json:"state"`
		HTMLURL   string `json:"html_url"`
		Comments  int    `json:"comments"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
		Labels    []struct {
			Name string `json:"name"`
		} `json:"labels"`
		User struct {
			Login string `json:"login"`
		} `json:"user"`
	}
	if err := json.Unmarshal(raw, &issue); err != nil {
		return nil, fmt.Errorf("github_issue: parse response: %w", err)
	}

	labels := make([]string, 0, len(issue.Labels))
	for _, l := range issue.Labels {
		labels = append(labels, l.Name)
	}

	return map[string]any{
		"number":         issue.Number,
		"title":          issue.Title,
		"body":           issue.Body,
		"state":          issue.State,
		"labels":         labels,
		"comments_count": issue.Comments,
		"url":            issue.HTMLURL,
		"author":         issue.User.Login,
		"created_at":     issue.CreatedAt,
		"updated_at":     issue.UpdatedAt,
	}, nil
}

func truncateBytes(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "…"
}
