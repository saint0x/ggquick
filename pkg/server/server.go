package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/v57/github"
	"github.com/saint0x/ggquick/pkg/ai"
	"github.com/saint0x/ggquick/pkg/hooks"
	"github.com/saint0x/ggquick/pkg/log"
	"golang.org/x/time/rate"
)

// RateLimiter wraps rate.Limiter with IP tracking
type RateLimiter struct {
	visitors map[string]*rate.Limiter
	mtx      sync.RWMutex
	rate     rate.Limit
	burst    int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	return &RateLimiter{
		visitors: make(map[string]*rate.Limiter),
		rate:     r,
		burst:    b,
	}
}

// GetVisitor gets or creates a limiter for an IP
func (rl *RateLimiter) GetVisitor(ip string) *rate.Limiter {
	rl.mtx.Lock()
	defer rl.mtx.Unlock()

	limiter, exists := rl.visitors[ip]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.visitors[ip] = limiter
	}

	return limiter
}

// CleanupVisitors removes old IP entries
func (rl *RateLimiter) CleanupVisitors() {
	rl.mtx.Lock()
	defer rl.mtx.Unlock()

	for ip := range rl.visitors {
		delete(rl.visitors, ip)
	}
}

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
	logger  *log.Logger
	ai      AIGenerator
	github  GitHubClient
	hooks   HooksManager
	srv     *http.Server
	mu      sync.RWMutex
	limiter *RateLimiter
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
		logger.Info("- Rate Limiter: ✓")
	}

	// Create rate limiter with 5 requests per second burst of 10
	limiter := NewRateLimiter(5, 10)

	return &Server{
		logger:  logger,
		ai:      ai,
		github:  gh,
		hooks:   hooks,
		limiter: limiter,
	}, nil
}

// rateLimit middleware applies rate limiting
func (s *Server) rateLimit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get IP from X-Forwarded-For or remote address
		ip := r.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip = r.RemoteAddr
		}

		limiter := s.limiter.GetVisitor(ip)
		if !limiter.Allow() {
			if s.logger.IsDebug() {
				s.logger.Warning("Rate limit exceeded for IP: %s", ip)
			}
			w.Header().Set("X-RateLimit-Limit", "5")
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Second).Unix()))
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		// Add rate limit headers
		w.Header().Set("X-RateLimit-Limit", "5")
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%.0f", limiter.Tokens()))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Second).Unix()))

		next(w, r)
	}
}

// Start starts the server
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.logger.IsDebug() {
		s.logger.Info("Starting server initialization...")
	}

	// Create server
	s.logger.Loading("Setting up server routes...")
	mux := http.NewServeMux()

	// Add health check first to ensure basic functionality
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		if s.logger.IsDebug() {
			s.logger.Debug("Health check request received")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","version":"1.0.0"}`))
		if s.logger.IsDebug() {
			s.logger.Debug("Health check response sent")
		}
	})

	// Add rate-limited routes
	mux.HandleFunc("/push", s.rateLimit(s.handlePush))

	s.srv = &http.Server{
		Addr:         "0.0.0.0:8080",
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	s.logger.Loading("Starting HTTP server...")
	serverErr := make(chan error, 1)
	serverStarted := make(chan struct{})

	go func() {
		s.logger.Debug("Server goroutine starting...")

		// Signal that we're about to start listening
		close(serverStarted)

		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Server error: %v", err)

			// Signal that we're done listening
			close(serverStarted)

			serverErr <- err
			return
		}
	}()

	// Wait for server to start or fail
	select {
	case <-serverStarted:
		s.logger.Success("Server started on 0.0.0.0:8080")
	case err := <-serverErr:
		return fmt.Errorf("server failed to start: %w", err)
	case <-time.After(5 * time.Second):
		return fmt.Errorf("server failed to start within timeout")
	}

	s.logger.Loading("Waiting for requests...")

	// Start periodic cleanup of old rate limit entries
	cleanup := time.NewTicker(10 * time.Minute)
	defer cleanup.Stop()

	// Wait for either context cancellation or server error
	for {
		select {
		case <-ctx.Done():
			s.logger.Loading("Shutting down server...")
			return s.Stop()
		case err := <-serverErr:
			return fmt.Errorf("server error: %w", err)
		case <-cleanup.C:
			if s.logger.IsDebug() {
				s.logger.Debug("Running rate limiter cleanup...")
			}
			s.limiter.CleanupVisitors()
		}
	}
}

// Stop stops the server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.srv != nil {
		s.logger.Loading("Gracefully stopping server...")

		// Create a context with timeout for shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := s.srv.Shutdown(ctx); err != nil {
			return fmt.Errorf("error shutting down server: %w", err)
		}
		s.logger.Success("Server stopped successfully")
	}

	return nil
}

// handlePush handles push events
func (s *Server) handlePush(w http.ResponseWriter, r *http.Request) {
	s.logger.Loading("🔄 Processing push event...")
	s.logger.Debug("📥 Push event received from %s", r.RemoteAddr)

	// Validate method
	if r.Method != http.MethodPost {
		s.logger.Error("❌ Invalid method: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var payload struct {
		Ref string `json:"ref"`
		SHA string `json:"sha"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.logger.Error("❌ Failed to decode payload: %v", err)
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}
	s.logger.Debug("✅ Payload decoded: ref=%s, sha=%s", payload.Ref, payload.SHA)

	// Extract branch name from ref
	branchName := strings.TrimPrefix(payload.Ref, "refs/heads/")
	s.logger.Debug("🔍 Branch name extracted: %s", branchName)

	// Initialize GitHub client
	s.logger.Loading("🔐 Initializing GitHub client...")
	if err := s.hooks.InitGitHub(os.Getenv("GITHUB_TOKEN"), "saint0x", "z-sample-repo"); err != nil {
		s.logger.Error("❌ Failed to initialize GitHub client: %v", err)
		http.Error(w, "Failed to initialize GitHub", http.StatusInternalServerError)
		return
	}
	s.logger.Debug("✅ GitHub client initialized with token")

	// Get default branch
	s.logger.Loading("🔍 Getting default branch...")
	defaultBranch, err := s.github.GetDefaultBranch(r.Context(), "saint0x", "z-sample-repo")
	if err != nil {
		s.logger.Error("❌ Failed to get default branch: %v", err)
		http.Error(w, "Failed to get default branch", http.StatusInternalServerError)
		return
	}
	s.logger.Debug("✅ Default branch is: %s", defaultBranch)

	// Get diff for analysis
	s.logger.Loading("📝 Fetching diff from GitHub...")
	diffURL, err := s.github.GetDiff(r.Context(), "saint0x", "z-sample-repo", defaultBranch, branchName)
	if err != nil {
		s.logger.Error("❌ Failed to get diff: %v", err)
		http.Error(w, "Failed to get diff", http.StatusInternalServerError)
		return
	}
	s.logger.Debug("✅ Got diff URL: %s", diffURL)

	// Get contributing guide
	guide, err := s.github.GetContributingGuide(r.Context(), "saint0x", "z-sample-repo")
	if err != nil {
		s.logger.Warning("⚠️ No contributing guide found: %v", err)
	}
	if guide != "" {
		s.logger.Debug("✅ Found contributing guide")
	}

	// Analyze diff
	s.logger.Loading("🔍 Preparing diff analysis...")
	analysis := ai.DiffAnalysis{
		Files: []string{},
		Changes: map[string]ai.Change{
			branchName: {
				Path:      "",
				Added:     []string{},
				Type:      "feature",
				Component: branchName,
			},
		},
		ContributingGuide: guide,
		Additions:         0,
		Deletions:         0,
	}
	s.logger.Debug("✅ Diff analysis prepared")

	// Generate PR title and description
	s.logger.Loading("🤖 Generating PR title with AI...")
	title, err := s.ai.GeneratePRTitle(analysis)
	if err != nil {
		s.logger.Error("❌ Failed to generate PR title: %v", err)
		http.Error(w, "Failed to generate PR title", http.StatusInternalServerError)
		return
	}
	s.logger.Debug("✅ Generated PR title: %s", title)

	s.logger.Loading("🤖 Generating PR description with AI...")
	desc, err := s.ai.GeneratePRDescription(analysis)
	if err != nil {
		s.logger.Error("❌ Failed to generate PR description: %v", err)
		http.Error(w, "Failed to generate PR description", http.StatusInternalServerError)
		return
	}
	s.logger.Debug("✅ Generated PR description")

	// Create PR
	s.logger.Loading("📦 Creating pull request...")
	pr, err := s.hooks.CreatePullRequest(r.Context(), &hooks.PullRequestOptions{
		Title:       title,
		Description: desc,
		Branch:      branchName,
		BaseBranch:  "main",
		Labels:      []string{"feature"},
	})
	if err != nil {
		s.logger.Error("❌ Failed to create PR: %v", err)
		http.Error(w, "Failed to create PR", http.StatusInternalServerError)
		return
	}
	s.logger.Success("✨ Pull request created successfully!")
	s.logger.Info("🔗 PR URL: %s", pr.GetHTMLURL())
	s.logger.Info("📝 Title: %s", title)
	s.logger.Info("🏷️  Labels: feature")

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if s.logger.IsDebug() {
		s.logger.Loading("Health check...")
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
	if s.logger.IsDebug() {
		s.logger.Success("Health check passed")
	}
}
