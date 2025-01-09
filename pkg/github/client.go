package github

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/google/go-github/v57/github"
	"github.com/saint0x/ggquick/pkg/log"
	"golang.org/x/oauth2"
)

// Client handles GitHub operations
type Client struct {
	client *github.Client
	logger *log.Logger
}

// New creates a new GitHub client
func New(logger *log.Logger) *Client {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		logger.Error("GITHUB_TOKEN environment variable not set")
		return nil
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.Background(), ts)

	return &Client{
		client: github.NewClient(tc),
		logger: logger,
	}
}

// CreatePR creates a new pull request
func (c *Client) CreatePR(ctx context.Context, owner, repo, title, body, head, base string) (*github.PullRequest, error) {
	pr, _, err := c.client.PullRequests.Create(ctx, owner, repo, &github.NewPullRequest{
		Title: github.String(title),
		Body:  github.String(body),
		Head:  github.String(head),
		Base:  github.String(base),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create PR: %w", err)
	}

	return pr, nil
}

// GetDefaultBranch gets the default branch for a repository
func (c *Client) GetDefaultBranch(ctx context.Context, owner, repo string) (string, error) {
	repository, _, err := c.client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return "", fmt.Errorf("failed to get repository: %w", err)
	}

	return repository.GetDefaultBranch(), nil
}

// ParseRepoURL parses a GitHub URL into owner and repo
func (c *Client) ParseRepoURL(repoURL string) (owner, repo string, err error) {
	// Handle different URL formats
	repoURL = strings.TrimSuffix(repoURL, ".git")

	// Handle SSH URLs (git@github.com:owner/repo)
	if strings.HasPrefix(repoURL, "git@github.com:") {
		parts := strings.Split(strings.TrimPrefix(repoURL, "git@github.com:"), "/")
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid SSH repository URL format")
		}
		return parts[0], parts[1], nil
	}

	// Handle HTTPS URLs
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid URL: %w", err)
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid repository URL format")
	}

	return parts[0], parts[1], nil
}

// GetContributingGuide gets the contributing guide content
func (c *Client) GetContributingGuide(ctx context.Context, owner, repo string) (string, error) {
	// Try common contributing guide paths
	paths := []string{
		"CONTRIBUTING.md",
		".github/CONTRIBUTING.md",
		"docs/CONTRIBUTING.md",
		"CONTRIBUTING",
		".github/CONTRIBUTING",
	}

	for _, path := range paths {
		content, _, _, err := c.client.Repositories.GetContents(
			ctx,
			owner,
			repo,
			path,
			&github.RepositoryContentGetOptions{},
		)
		if err == nil && content != nil {
			decoded, err := content.GetContent()
			if err != nil {
				continue
			}
			return decoded, nil
		}
	}

	return "", fmt.Errorf("no contributing guide found")
}

// GetBranches gets all branches for a repository
func (c *Client) GetBranches(ctx context.Context, owner, repo string) ([]*github.Branch, error) {
	var allBranches []*github.Branch
	opts := &github.BranchListOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		branches, resp, err := c.client.Repositories.ListBranches(ctx, owner, repo, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list branches: %w", err)
		}

		allBranches = append(allBranches, branches...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allBranches, nil
}

// GetPRs gets recent pull requests
func (c *Client) GetPRs(ctx context.Context, owner, repo string, limit int) ([]*github.PullRequest, error) {
	opts := &github.PullRequestListOptions{
		State: "all",
		ListOptions: github.ListOptions{
			PerPage: limit,
		},
	}

	prs, _, err := c.client.PullRequests.List(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list PRs: %w", err)
	}

	return prs, nil
}

// GetDiff gets the diff for a branch
func (c *Client) GetDiff(ctx context.Context, owner, repo, base, head string) (string, error) {
	comp, _, err := c.client.Repositories.CompareCommits(
		ctx,
		owner,
		repo,
		base,
		head,
		&github.ListOptions{},
	)
	if err != nil {
		return "", fmt.Errorf("failed to get diff: %w", err)
	}

	return comp.GetDiffURL(), nil
}

// GetCommitMessage gets the commit message for a SHA
func (c *Client) GetCommitMessage(ctx context.Context, owner, repo, sha string) (string, error) {
	commit, _, err := c.client.Git.GetCommit(ctx, owner, repo, sha)
	if err != nil {
		return "", fmt.Errorf("failed to get commit: %w", err)
	}

	return commit.GetMessage(), nil
}
