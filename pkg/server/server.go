package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

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

	if logger.IsDebug() {
		logger.Info("Initializing server with components:")
		logger.Info("- AI Generator: ✓")
		logger.Info("- GitHub Client: ✓")
		logger.Info("- Hooks Manager: ✓")
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

	if s.logger.IsDebug() {
		s.logger.Info("Starting server initialization...")
	}

	// Create server
	mux := http.NewServeMux()
	mux.HandleFunc("/push", s.handlePush)
	if s.logger.IsDebug() {
		s.logger.Info("Registered webhook handler at /push")
	}

	// Get port from env or use default
	port := os.Getenv("GGQUICK_PORT")
	if port == "" {
		port = "8080"
		if s.logger.IsDebug() {
			s.logger.Info("Using default port: %s", port)
		}
	} else if s.logger.IsDebug() {
		s.logger.Info("Using configured port: %s", port)
	}

	// Try to find an available port
	listener, err := s.findAvailablePort(port)
	if err != nil {
		return fmt.Errorf("failed to find available port: %w", err)
	}
	actualPort := listener.Addr().(*net.TCPAddr).Port

	// Save port for hooks
	configDir := filepath.Join(os.Getenv("HOME"), ".ggquick")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		listener.Close()
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	portFile := filepath.Join(configDir, "port")
	if err := os.WriteFile(portFile, []byte(fmt.Sprintf("%d", actualPort)), 0644); err != nil {
		listener.Close()
		return fmt.Errorf("failed to save port: %w", err)
	}

	s.srv = &http.Server{
		Handler: mux,
	}

	// Start server using the existing listener
	go func() {
		if err := s.srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Server error: %v", err)
		}
	}()

	s.logger.Success("Server is running on port %d", actualPort)
	if s.logger.IsDebug() {
		s.logger.Info("Webhook URL: http://localhost:%d/push", actualPort)
		s.logger.Info("Press Ctrl+C to stop")
	}

	// Wait for context cancellation
	<-ctx.Done()
	return s.Stop()
}

// findAvailablePort tries to find an available port starting from the given port
func (s *Server) findAvailablePort(startPort string) (net.Listener, error) {
	// Try the specified port first
	listener, err := net.Listen("tcp", ":"+startPort)
	if err == nil {
		return listener, nil
	}

	if s.logger.IsDebug() {
		s.logger.Info("Port %s is in use, searching for available port...", startPort)
	}

	// Try to find a random available port
	listener, err = net.Listen("tcp", ":0")
	if err != nil {
		return nil, fmt.Errorf("failed to find available port: %w", err)
	}

	return listener, nil
}

// Stop stops the server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.srv != nil {
		// Create a timeout context for shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.srv.Shutdown(ctx); err != nil {
			s.logger.Error("Failed to stop server: %v", err)
			return fmt.Errorf("failed to stop server: %w", err)
		}
		s.srv = nil
		s.logger.Success("Server stopped")
	}

	return nil
}

// handlePush handles push events
func (s *Server) handlePush(w http.ResponseWriter, r *http.Request) {
	if s.logger.IsDebug() {
		s.logger.Info("Received webhook request from %s", r.RemoteAddr)
	}

	if r.Method != http.MethodPost {
		s.logger.Warning("Invalid method %s from %s", r.Method, r.RemoteAddr)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var ref, sha string
	contentType := r.Header.Get("Content-Type")

	if strings.HasPrefix(contentType, "application/json") {
		var event struct {
			Ref    string `json:"ref"`
			After  string `json:"after"`
			Before string `json:"before"`
		}
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			s.logger.Error("Failed to decode JSON payload: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ref = event.Ref
		sha = event.After
		if s.logger.IsDebug() {
			s.logger.Info("Parsed JSON webhook: ref=%s, sha=%s", ref, sha)
		}
	} else {
		scanner := bufio.NewScanner(r.Body)
		if scanner.Scan() {
			sha = scanner.Text()
		}
		ref = "refs/heads/" + sha
		if s.logger.IsDebug() {
			s.logger.Info("Parsed text webhook: sha=%s", sha)
		}
	}

	// Get repository info from environment
	repoURL := os.Getenv("GITHUB_REPOSITORY")
	if repoURL == "" {
		s.logger.Error("GITHUB_REPOSITORY environment variable not set")
		http.Error(w, "GITHUB_REPOSITORY environment variable not set", http.StatusInternalServerError)
		return
	}

	// Create PR
	go s.createPR(repoURL, ref, sha)

	w.WriteHeader(http.StatusOK)
}

// createPR generates and creates a pull request
func (s *Server) createPR(repoFullName string, ref, sha string) {
	s.logger.Step("Starting PR creation...")

	// Parse repository info
	owner, repo, err := s.github.ParseRepoURL("https://github.com/" + repoFullName)
	if err != nil {
		s.logger.Error("Failed to parse repository URL: %v", err)
		return
	}
	s.logger.Info("Repository: %s/%s", owner, repo)

	// Get branch name
	branch := ref[11:] // Remove "refs/heads/"
	s.logger.Branch("Branch: %s", branch)

	// Get default branch
	s.logger.Step("Fetching repository info...")
	defaultBranch, err := s.github.GetDefaultBranch(context.Background(), owner, repo)
	if err != nil {
		s.logger.Error("Failed to get default branch: %v", err)
		return
	}
	s.logger.Branch("Default branch: %s", defaultBranch)

	// Get contributing guide
	s.logger.Step("Checking contributing guidelines...")
	guide, err := s.github.GetContributingGuide(context.Background(), owner, repo)
	if err != nil {
		s.logger.Warning("No contributing guide found: %v", err)
	} else {
		s.logger.Success("Found contributing guidelines (%d bytes)", len(guide))
	}

	// Get diff and analyze changes
	s.logger.Step("Analyzing changes...")
	diffURL, err := s.github.GetDiff(context.Background(), owner, repo, defaultBranch, branch)
	if err != nil {
		s.logger.Error("Failed to get diff: %v", err)
		return
	}
	s.logger.Diff("Changes between %s...%s", defaultBranch, branch)

	// Prepare diff analysis
	diffAnalysis := ai.DiffAnalysis{
		Files:     []string{branch},
		Additions: 0,
		Deletions: 0,
		Changes: map[string]ai.Change{
			branch: {
				Path:      branch,
				Type:      "feature",
				Component: branch,
				Modified:  []string{fmt.Sprintf("Diff URL: %s", diffURL)},
			},
		},
	}

	// Generate PR content
	s.logger.Step("Generating PR content...")
	title, err := s.ai.GeneratePRTitle(diffAnalysis)
	if err != nil {
		s.logger.Error("Failed to generate title: %v", err)
		return
	}
	s.logger.PR("Title: %s", title)

	desc, err := s.ai.GeneratePRDescription(diffAnalysis)
	if err != nil {
		s.logger.Error("Failed to generate description: %v", err)
		return
	}

	// Add contributing guide context if available
	if guide != "" {
		s.logger.Step("Applying contributing guidelines...")
		desc = fmt.Sprintf("Following repository guidelines:\n\n%s\n\n%s", guide, desc)
		if s.logger.IsDebug() {
			s.logger.Debug("Description with guidelines (%d bytes)", len(desc))
		}
	}

	// Add commit SHA for reference
	desc = fmt.Sprintf("%s\n\nCommit: %s", desc, sha)

	// Create PR
	s.logger.Step("Creating pull request...")
	pr, err := s.github.CreatePR(context.Background(), owner, repo, title, desc, branch, defaultBranch)
	if err != nil {
		s.logger.Error("Failed to create PR: %v", err)
		return
	}

	s.logger.Success("Created PR #%d", pr.GetNumber())
	s.logger.PR("URL: %s", pr.GetHTMLURL())
}
