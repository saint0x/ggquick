package hooks

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/go-github/v57/github"
	"github.com/saint0x/ggquick/pkg/log"
	"golang.org/x/oauth2"
)

// Manager handles git hooks and GitHub API integration
type Manager struct {
	logger *log.Logger
	github *GitHubClient
	mu     sync.RWMutex
	hooks  sync.Map // map[string]*Hook
}

// GitHubClient wraps GitHub API client
type GitHubClient struct {
	client *github.Client
	owner  string
	repo   string
}

// PullRequestOptions contains options for creating a PR
type PullRequestOptions struct {
	Title       string
	Description string
	Branch      string
	BaseBranch  string
	Labels      []string
}

// Hook represents a git hook
type Hook struct {
	Name     string
	Path     string
	Template string
	Enabled  bool
}

// RepoInfo contains repository information
type RepoInfo struct {
	Path      string
	HooksPath string
}

// New creates a new hooks manager
func New(logger *log.Logger) *Manager {
	return &Manager{
		logger: logger,
	}
}

// InitGitHub initializes GitHub client with token
func (m *Manager) InitGitHub(token, owner, repo string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	m.github = &GitHubClient{
		client: github.NewClient(tc),
		owner:  owner,
		repo:   repo,
	}

	return nil
}

// CreatePullRequest creates a new PR on GitHub
func (m *Manager) CreatePullRequest(ctx context.Context, opts *PullRequestOptions) (*github.PullRequest, error) {
	m.mu.RLock()
	gh := m.github
	m.mu.RUnlock()

	if gh == nil {
		return nil, fmt.Errorf("GitHub client not initialized")
	}

	pr, _, err := gh.client.PullRequests.Create(ctx, gh.owner, gh.repo, &github.NewPullRequest{
		Title:               github.String(opts.Title),
		Body:                github.String(opts.Description),
		Head:                github.String(opts.Branch),
		Base:                github.String(opts.BaseBranch),
		MaintainerCanModify: github.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create PR: %w", err)
	}

	// Add labels if specified
	if len(opts.Labels) > 0 {
		_, _, err = gh.client.Issues.AddLabelsToIssue(ctx, gh.owner, gh.repo, pr.GetNumber(), opts.Labels)
		if err != nil {
			return nil, fmt.Errorf("failed to add labels: %w", err)
		}
	}

	return pr, nil
}

// InstallHooks installs git hooks in the repository
func (m *Manager) InstallHooks(repoPath string) error {
	// Install post-commit hook
	hook := fmt.Sprintf(`#!/bin/sh
# ggquick post-commit hook
if [ -z "$GGQUICK_DISABLED" ]; then
	curl -s -X POST "https://ggquick.fly.dev/push" \
		-H "Content-Type: application/json" \
		-d "{\"ref\":\"$(git rev-parse --abbrev-ref HEAD)\",\"sha\":\"$(git rev-parse HEAD)\"}" >/dev/null || true
fi
`)

	// Write hook file
	if err := writeHook(repoPath, "post-commit", hook); err != nil {
		return fmt.Errorf("failed to install post-commit hook: %w", err)
	}

	// Install post-push hook
	hook = fmt.Sprintf(`#!/bin/sh
# ggquick post-push hook
if [ -z "$GGQUICK_DISABLED" ]; then
	curl -s -X POST "https://ggquick.fly.dev/push" \
		-H "Content-Type: application/json" \
		-d "{\"ref\":\"$(git rev-parse --abbrev-ref HEAD)\",\"sha\":\"$(git rev-parse HEAD)\"}" >/dev/null || true
fi
`)

	if err := writeHook(repoPath, "post-push", hook); err != nil {
		return fmt.Errorf("failed to install post-push hook: %w", err)
	}

	return nil
}

// writeHook writes a git hook file
func writeHook(repoPath, hookName, content string) error {
	hookPath := filepath.Join(repoPath, hookName)
	if err := os.WriteFile(hookPath, []byte(content), 0755); err != nil {
		return err
	}
	return nil
}

// UpdateRepo updates hooks for a repository
func (m *Manager) UpdateRepo(repo *RepoInfo) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Define hooks
	hooks := []*Hook{
		{
			Name: "post-commit",
			Template: `#!/bin/sh
# ggquick post-commit hook
if [ -z "$GGQUICK_DISABLED" ]; then
	curl -s -X POST "https://ggquick.fly.dev/push" \
		-H "Content-Type: application/json" \
		-d "{\"ref\":\"$(git rev-parse --abbrev-ref HEAD)\",\"sha\":\"$(git rev-parse HEAD)\"}" >/dev/null || true
fi
`,
			Enabled: true,
		},
		{
			Name: "post-push",
			Template: `#!/bin/sh
# ggquick post-push hook
if [ -z "$GGQUICK_DISABLED" ]; then
	curl -s -X POST "https://ggquick.fly.dev/push" \
		-H "Content-Type: application/json" \
		-d "{\"ref\":\"$(git rev-parse --abbrev-ref HEAD)\",\"sha\":\"$(git rev-parse HEAD)\"}" >/dev/null || true
fi
`,
			Enabled: true,
		},
	}

	// Install hooks in parallel
	var wg sync.WaitGroup
	errCh := make(chan error, len(hooks))

	for _, hook := range hooks {
		wg.Add(1)
		go func(h *Hook) {
			defer wg.Done()
			if err := writeHook(repo.HooksPath, h.Name, h.Template); err != nil {
				errCh <- fmt.Errorf("failed to install %s: %w", h.Name, err)
			}
			m.hooks.Store(h.Name, h)
		}(hook)
	}

	wg.Wait()
	close(errCh)

	// Check for errors
	for err := range errCh {
		if err != nil {
			return err
		}
	}

	return nil
}

// RemoveHooks removes all hooks from a repository
func (m *Manager) RemoveHooks(hooksPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var wg sync.WaitGroup
	errCh := make(chan error, 2) // Number of hooks

	m.hooks.Range(func(key, value interface{}) bool {
		wg.Add(1)
		go func(hookName string) {
			defer wg.Done()
			hookPath := fmt.Sprintf("%s/%s", hooksPath, hookName)
			if err := os.Remove(hookPath); err != nil && !os.IsNotExist(err) {
				errCh <- err
			}
		}(key.(string))
		return true
	})

	wg.Wait()
	close(errCh)

	// Check for errors
	for err := range errCh {
		if err != nil {
			return err
		}
	}

	return nil
}

// ValidateGitRepo validates a git repository
func (m *Manager) ValidateGitRepo(path string) error {
	gitPath := fmt.Sprintf("%s/.git", path)
	if _, err := os.Stat(gitPath); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository: %w", err)
	}
	return nil
}
