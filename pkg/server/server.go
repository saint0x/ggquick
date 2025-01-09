package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/go-github/v57/github"
	"github.com/saint0x/ggquick/pkg/ai"
	"github.com/saint0x/ggquick/pkg/hooks"
	"github.com/saint0x/ggquick/pkg/log"
)

// AIGenerator interface for generating PR content
type AIGenerator interface {
	GeneratePRTitle(diff ai.DiffAnalysis) (string, error)
	GeneratePRDescription(diff ai.DiffAnalysis) (string, error)
	AnalyzeCommit(msg string) (*ai.Analysis, error)
	AnalyzeCode(code string) (*ai.Analysis, error)
}

// GitHubClient interface for GitHub operations
type GitHubClient interface {
	CreatePR(ctx context.Context, owner, repo, title, body, head, base string) (*github.PullRequest, error)
	GetDefaultBranch(ctx context.Context, owner, repo string) (string, error)
	ParseRepoURL(url string) (owner, repo string, err error)
	GetContributingGuide(ctx context.Context, owner, repo string) (string, error)
	GetBranches(ctx context.Context, owner, repo string) ([]*github.Branch, error)
	GetPRs(ctx context.Context, owner, repo string, limit int) ([]*github.PullRequest, error)
	GetDiff(ctx context.Context, owner, repo, base, head string) (string, error)
}

// HooksManager interface for git hooks
type HooksManager interface {
	InstallHooks(string) error
	InitGitHub(token, owner, repo string) error
	CreatePullRequest(ctx context.Context, opts *hooks.PullRequestOptions) (*github.PullRequest, error)
	UpdateRepo(repo *hooks.RepoInfo) error
	RemoveHooks(string) error
	ValidateGitRepo(string) error
}

// Server handles webhook events and PR creation
type Server struct {
	logger *log.Logger
	ai     AIGenerator
	github GitHubClient
	hooks  HooksManager
	srv    *http.Server
	mu     sync.RWMutex
}

// New creates a new server instance
func New(logger *log.Logger, ai AIGenerator, gh GitHubClient, hooks HooksManager) (*Server, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	if ai == nil {
		return nil, fmt.Errorf("ai generator is required")
	}
	if gh == nil {
		return nil, fmt.Errorf("github client is required")
	}
	if hooks == nil {
		return nil, fmt.Errorf("hooks manager is required")
	}

	return &Server{
		logger: logger,
		ai:     ai,
		github: gh,
		hooks:  hooks,
	}, nil
}

// Start starts the server
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create server
	mux := http.NewServeMux()
	mux.HandleFunc("/push", s.handlePush)

	// Get port from env or use default
	port := os.Getenv("GGQUICK_PORT")
	if port == "" {
		port = "8080"
	}

	// Save port for hooks
	if err := os.MkdirAll(filepath.Join(os.Getenv("HOME"), ".ggquick"), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(os.Getenv("HOME"), ".ggquick/port"), []byte(port), 0644); err != nil {
		return fmt.Errorf("failed to save port: %w", err)
	}

	s.srv = &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// Start server
	go func() {
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Server error: %v", err)
		}
	}()

	s.logger.Success("Server started on port %s", port)

	// Wait for context cancellation
	<-ctx.Done()
	return s.Stop()
}

// Stop stops the server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.srv != nil {
		if err := s.srv.Shutdown(context.Background()); err != nil {
			return fmt.Errorf("failed to stop server: %w", err)
		}
		s.srv = nil
	}

	return nil
}

// handlePush handles push events
func (s *Server) handlePush(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var ref, sha string
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		var event struct {
			Ref    string `json:"ref"`
			After  string `json:"after"`
			Before string `json:"before"`
		}
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ref = event.Ref
		sha = event.After
	} else {
		// Parse plain text format: <sha>\n<message>\n<author>
		scanner := bufio.NewScanner(r.Body)
		if scanner.Scan() {
			sha = scanner.Text()
		}
		ref = "refs/heads/" + sha // Use commit hash as branch name
	}

	// Get repository info from environment
	repoURL := os.Getenv("GITHUB_REPOSITORY")
	if repoURL == "" {
		http.Error(w, "GITHUB_REPOSITORY environment variable not set", http.StatusInternalServerError)
		return
	}

	// Create PR
	go s.createPR(repoURL, ref, sha)

	w.WriteHeader(http.StatusOK)
}

// createPR generates and creates a pull request
func (s *Server) createPR(repoFullName string, ref, sha string) {
	owner, repo, _ := s.github.ParseRepoURL("https://github.com/" + repoFullName)
	branch := ref[11:] // Remove "refs/heads/"

	// Get default branch
	defaultBranch, err := s.github.GetDefaultBranch(context.Background(), owner, repo)
	if err != nil {
		s.logger.Error("Failed to get default branch: %v", err)
		return
	}

	// Get contributing guide
	guide, err := s.github.GetContributingGuide(context.Background(), owner, repo)
	if err != nil {
		s.logger.Warning("No contributing guide found: %v", err)
	}

	// Get diff and analyze changes
	diffURL, err := s.github.GetDiff(context.Background(), owner, repo, defaultBranch, branch)
	if err != nil {
		s.logger.Error("Failed to get diff: %v", err)
		return
	}

	// Convert diff to analysis format
	diffAnalysis := ai.DiffAnalysis{
		Files:     []string{branch},
		Additions: 0, // We don't parse these from the diff text
		Deletions: 0,
		Changes: map[string]ai.Change{
			branch: {
				Path:      branch,
				Type:      "feature", // Default type
				Component: branch,
				Modified:  []string{fmt.Sprintf("Diff URL: %s", diffURL)},
			},
		},
	}

	// Generate PR content
	title, err := s.ai.GeneratePRTitle(diffAnalysis)
	if err != nil {
		s.logger.Error("Failed to generate title: %v", err)
		return
	}

	desc, err := s.ai.GeneratePRDescription(diffAnalysis)
	if err != nil {
		s.logger.Error("Failed to generate description: %v", err)
		return
	}

	// Add contributing guide context if available
	if guide != "" {
		desc = fmt.Sprintf("Following contributing guidelines:\n\n%s\n\n%s", guide, desc)
	}

	// Create PR
	pr, err := s.github.CreatePR(context.Background(), owner, repo, title, desc, branch, defaultBranch)
	if err != nil {
		s.logger.Error("Failed to create PR: %v", err)
		return
	}

	s.logger.Success("Created PR #%d: %s", pr.GetNumber(), pr.GetHTMLURL())
}
