package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/MortalArena/Musketeers/pkg/capability"
	"github.com/MortalArena/Musketeers/pkg/policy"
)

type GitHubCapability struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

func NewGitHubCapability(token string) *GitHubCapability {
	return &GitHubCapability{BaseURL: "https://api.github.com", Token: token, HTTPClient: &http.Client{Timeout: 30 * time.Second}}
}

func (c *GitHubCapability) Name() string { return "github" }

func (c *GitHubCapability) Execute(ctx context.Context, principal policy.Principal, cmd capability.Command) (*capability.Result, error) {
	switch v := cmd.(type) {
	case ListReposCommand:
		return c.listRepos(ctx, v)
	case CreateIssueCommand:
		return c.createIssue(ctx, v)
	case ReadFileCommand:
		return c.readFile(ctx, v)
	case GetPRCommand:
		return c.getPR(ctx, v)
	case CreateCommentCommand:
		return c.createComment(ctx, v)
	default:
		return nil, fmt.Errorf("unsupported github command: %s", cmd.Name())
	}
}

type ListReposCommand struct {
	Owner string `json:"owner"`
	Type  string `json:"type,omitempty"`
}

func (ListReposCommand) Name() string { return "github.list_repos" }
func (c ListReposCommand) Args() map[string]any {
	return map[string]any{"owner": c.Owner, "type": c.Type}
}

type CreateIssueCommand struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
	Title string `json:"title"`
	Body  string `json:"body,omitempty"`
}

func (CreateIssueCommand) Name() string { return "github.create_issue" }
func (c CreateIssueCommand) Args() map[string]any {
	return map[string]any{"owner": c.Owner, "repo": c.Repo, "title": c.Title, "body": c.Body}
}

type GetPRCommand struct {
	Owner      string `json:"owner"`
	Repo       string `json:"repo"`
	PullNumber int    `json:"pull_number"`
}

func (GetPRCommand) Name() string { return "github.get_pr" }
func (c GetPRCommand) Args() map[string]any {
	return map[string]any{"owner": c.Owner, "repo": c.Repo, "pull_number": c.PullNumber}
}

type ReadFileCommand struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
	Path  string `json:"path"`
}

func (ReadFileCommand) Name() string { return "github.read_file" }
func (c ReadFileCommand) Args() map[string]any {
	return map[string]any{"owner": c.Owner, "repo": c.Repo, "path": c.Path}
}

type CreateCommentCommand struct {
	Owner   string `json:"owner"`
	Repo    string `json:"repo"`
	IssueNo int    `json:"issue_number"`
	Body    string `json:"body"`
}

func (CreateCommentCommand) Name() string { return "github.create_comment" }
func (c CreateCommentCommand) Args() map[string]any {
	return map[string]any{"owner": c.Owner, "repo": c.Repo, "issue_number": c.IssueNo, "body": c.Body}
}

func (c *GitHubCapability) listRepos(ctx context.Context, cmd ListReposCommand) (*capability.Result, error) {
	if cmd.Owner == "" {
		return nil, fmt.Errorf("owner is required")
	}
	path := fmt.Sprintf("/users/%s/repos", cmd.Owner)
	if cmd.Type != "" {
		path += "?type=" + cmd.Type
	}
	var repos []map[string]any
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &repos); err != nil {
		return nil, err
	}
	return capability.NewResult(cmd.Name(), map[string]any{"repos": repos}), nil
}

func (c *GitHubCapability) createIssue(ctx context.Context, cmd CreateIssueCommand) (*capability.Result, error) {
	if cmd.Owner == "" || cmd.Repo == "" || cmd.Title == "" {
		return nil, fmt.Errorf("owner, repo and title are required")
	}
	path := fmt.Sprintf("/repos/%s/%s/issues", cmd.Owner, cmd.Repo)
	var issue map[string]any
	if err := c.doJSON(ctx, http.MethodPost, path, map[string]any{"title": cmd.Title, "body": cmd.Body}, &issue); err != nil {
		return nil, err
	}
	return capability.NewResult(cmd.Name(), map[string]any{"issue": issue}), nil
}

func (c *GitHubCapability) readFile(ctx context.Context, cmd ReadFileCommand) (*capability.Result, error) {
	if cmd.Owner == "" || cmd.Repo == "" || cmd.Path == "" {
		return nil, fmt.Errorf("owner, repo and path are required")
	}
	path := fmt.Sprintf("/repos/%s/%s/contents/%s", cmd.Owner, cmd.Repo, cmd.Path)
	var content map[string]any
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &content); err != nil {
		return nil, err
	}
	return capability.NewResult(cmd.Name(), map[string]any{"content": content}), nil
}

func (c *GitHubCapability) getPR(ctx context.Context, cmd GetPRCommand) (*capability.Result, error) {
	if cmd.Owner == "" || cmd.Repo == "" || cmd.PullNumber == 0 {
		return nil, fmt.Errorf("owner, repo and pull_number are required")
	}
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d", cmd.Owner, cmd.Repo, cmd.PullNumber)
	var pr map[string]any
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &pr); err != nil {
		return nil, err
	}
	return capability.NewResult(cmd.Name(), map[string]any{"pull_request": pr}), nil
}

func (c *GitHubCapability) createComment(ctx context.Context, cmd CreateCommentCommand) (*capability.Result, error) {
	if cmd.Owner == "" || cmd.Repo == "" || cmd.IssueNo == 0 || cmd.Body == "" {
		return nil, fmt.Errorf("owner, repo, issue_number and body are required")
	}
	path := fmt.Sprintf("/repos/%s/%s/issues/%d/comments", cmd.Owner, cmd.Repo, cmd.IssueNo)
	var comment map[string]any
	if err := c.doJSON(ctx, http.MethodPost, path, map[string]any{"body": cmd.Body}, &comment); err != nil {
		return nil, err
	}
	return capability.NewResult(cmd.Name(), map[string]any{"comment": comment}), nil
}

func (c *GitHubCapability) doJSON(ctx context.Context, method, path string, body any, out any) error {
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	baseURL := strings.TrimRight(c.BaseURL, "/")
	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, reader)
	if err != nil {
		return err
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("github request failed: %s: %s", resp.Status, string(data))
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}
