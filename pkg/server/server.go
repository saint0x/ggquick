package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/v57/github"
	"github.com/saint0x/ggquick/pkg/ai"
	"github.com/saint0x/ggquick/pkg/log"
	"golang.org/x/time/rate"
)

// Config stores repository configuration
type Config struct {
	RepoURL       string `json:"repo_url"`
	Owner         string `json:"owner"`
	Name          string `json:"name"`
	DefaultBranch string `json:"default_branch"`
}

// GitHubClient interface for GitHub operations
type GitHubClient interface {
	CreatePullRequest(ctx context.Context, owner, repo string, pr *github.NewPullRequest) (*github.PullRequest, error)
	GetDefaultBranch(ctx context.Context, owner, repo string) (string, error)
}

// HooksManager interface for webhook management
type HooksManager interface {
	CreateHook(ctx context.Context, owner, repo, url string) error
	DeleteHook(ctx context.Context, owner, repo string) error
}

// RateLimiter wraps rate.Limiter with a mutex for concurrent access
type RateLimiter struct {
	limiter *rate.Limiter
	mu      sync.Mutex
}

// Server handles HTTP requests for the ggquick service
type Server struct {
	logger    *log.Logger
	config    *Config
	generator *ai.Generator
	limiter   *RateLimiter
	mu        sync.RWMutex
	github    GitHubClient
	hooks     HooksManager
	srv       *http.Server
}

// New creates a new server instance
func New(logger *log.Logger, generator *ai.Generator, github GitHubClient, hooks HooksManager) (*Server, error) {
	// Validate required components
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	if generator == nil {
		return nil, fmt.Errorf("AI generator is required")
	}
	if github == nil {
		return nil, fmt.Errorf("GitHub client is required")
	}
	if hooks == nil {
		return nil, fmt.Errorf("hooks manager is required")
	}

	// Create rate limiter: 1 request per second with burst of 5
	limiter := &RateLimiter{
		limiter: rate.NewLimiter(rate.Every(time.Second), 5),
	}

	return &Server{
		logger:    logger,
		generator: generator,
		github:    github,
		hooks:     hooks,
		limiter:   limiter,
		mu:        sync.RWMutex{},
	}, nil
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	// Validate server state
	if err := s.validateState(); err != nil {
		return fmt.Errorf("invalid server state: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", s.handleWebhook)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/config", s.handleConfig)

	// Get server address from environment
	addr := ":8080" // Default port
	if bind := os.Getenv("BIND"); bind != "" {
		addr = bind // Use full bind address if specified
	} else if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port // Use just the port if specified
	}

	s.srv = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Single, clear startup sequence
	s.logger.Loading("🚀 Starting ggquick server...")
	s.logger.Info("🔧 Debug mode: %v", s.logger.IsDebug())

	// Check environment
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		s.logger.Success("✅ GITHUB_TOKEN configured")
	} else {
		s.logger.Error("❌ GITHUB_TOKEN not configured")
		return fmt.Errorf("GITHUB_TOKEN not configured")
	}
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		s.logger.Success("✅ OPENAI_API_KEY configured")
	} else {
		s.logger.Error("❌ OPENAI_API_KEY not configured")
		return fmt.Errorf("OPENAI_API_KEY not configured")
	}

	// Initialize components
	s.logger.Loading("⚙️ Initializing components...")
	s.logger.Success("✅ AI generator ready")
	s.logger.Success("✅ GitHub client ready")
	s.logger.Success("✅ Git hooks ready")
	s.logger.Success("✅ Server initialized")

	// Start HTTP server
	s.logger.Loading("🌐 Starting HTTP server on %s...", addr)
	s.logger.Info("⚡ Endpoints initialized:")
	s.logger.Info("   • /health - Server health check")
	s.logger.Info("   • /config - Repository configuration")
	s.logger.Info("   • /webhook - GitHub event handling")

	errCh := make(chan error, 1)
	go func() {
		s.logger.Debug("Starting server on %s", addr)
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("❌ Server error: %v", err)
			errCh <- fmt.Errorf("server error: %w", err)
		}
		close(errCh)
	}()

	s.logger.Success("✅ Server is ready to accept connections")

	// Wait for either context cancellation or server error
	select {
	case err := <-errCh:
		if err != nil {
			s.logger.Error("❌ Server error: %v", err)
		}
		return err
	case <-ctx.Done():
		s.logger.Info("🛑 Initiating graceful shutdown...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.srv.Shutdown(shutdownCtx)
	}
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// handleConfig handles setting the repository configuration
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	s.logger.Loading("📥 Receiving configuration request...")
	s.logger.Debug("Request from: %s", r.RemoteAddr)

	if r.Method != http.MethodPost {
		s.logger.Error("❌ Invalid method: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var config Config
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		s.logger.Error("❌ Failed to decode configuration: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Parse owner and name from URL if not set
	if config.Owner == "" || config.Name == "" {
		parts := strings.Split(strings.TrimSuffix(config.RepoURL, ".git"), "/")
		if len(parts) < 2 {
			s.logger.Error("❌ Invalid repository URL format")
			http.Error(w, "Invalid repository URL format", http.StatusBadRequest)
			return
		}
		config.Owner = parts[len(parts)-2]
		config.Name = parts[len(parts)-1]
	}

	s.logger.Success("✅ Parsed repository details:")
	s.logger.Info("   📦 Repository: %s", config.RepoURL)
	s.logger.Info("   👤 Owner: %s", config.Owner)
	s.logger.Info("   📝 Name: %s", config.Name)

	// Get default branch
	defaultBranch, err := s.github.GetDefaultBranch(r.Context(), config.Owner, config.Name)
	if err != nil {
		s.logger.Error("❌ Failed to get default branch: %v", err)
		http.Error(w, "Failed to get repository details", http.StatusInternalServerError)
		return
	}
	config.DefaultBranch = defaultBranch
	s.logger.Info("   🌿 Default branch: %s", defaultBranch)

	// Store config in memory
	s.logger.Loading("💾 Storing configuration...")
	s.mu.Lock()
	s.config = &config
	s.mu.Unlock()
	s.logger.Success("✨ Configuration stored successfully")

	// Create webhook
	s.logger.Loading("🔗 Setting up GitHub webhook...")
	// Use fly.io domain for production, fallback to local address for development
	webhookURL := "https://ggquick.fly.dev/webhook"
	if os.Getenv("FLY_APP_NAME") == "" {
		// For local development, use the actual server port
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		webhookURL = fmt.Sprintf("http://localhost:%s/webhook", port)
	}
	s.logger.Debug("Webhook URL: %s", webhookURL)

	// Check webhook status
	s.logger.Loading("🔍 Checking webhook status...")
	if err := s.hooks.CreateHook(r.Context(), config.Owner, config.Name, webhookURL); err != nil {
		s.logger.Error("❌ Failed to manage webhook: %v", err)
		http.Error(w, "Failed to manage webhook", http.StatusInternalServerError)
		return
	}
	s.logger.Success("✅ GitHub webhook configured")

	// Send confirmation response with repository details
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "config_stored",
		"owner":  config.Owner,
		"name":   config.Name,
	})
	s.logger.Success("🔄 Ready to process Git events for %s/%s", config.Owner, config.Name)
}

// handleWebhook handles incoming GitHub webhook events
func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	s.logger.Loading("📥 Processing incoming webhook...")
	s.logger.Debug("Request from: %s", r.RemoteAddr)

	if r.Method != http.MethodPost {
		s.logger.Error("❌ Invalid method: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check rate limit
	if err := s.checkRateLimit(r.Context()); err != nil {
		s.logger.Error("❌ Rate limit exceeded: %v", err)
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	// Parse webhook event
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Error("❌ Failed to read request body: %v", err)
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		s.logger.Error("❌ Failed to parse webhook: %v", err)
		http.Error(w, "Invalid webhook payload", http.StatusBadRequest)
		return
	}

	// Handle push event
	switch e := event.(type) {
	case *github.PushEvent:
		s.logger.Success("✅ Received push event")
		s.logger.Info("📝 Repository: %s", *e.Repo.FullName)
		s.logger.Info("📝 Branch: %s", strings.TrimPrefix(*e.Ref, "refs/heads/"))

		// Get stored config
		s.mu.RLock()
		config := s.config
		s.mu.RUnlock()

		if config == nil {
			s.logger.Error("❌ No repository configuration found")
			http.Error(w, "Repository not configured", http.StatusBadRequest)
			return
		}

		s.logger.Info("📝 Using stored config for %s/%s", config.Owner, config.Name)

		// Process push event
		if err := s.processPushEvent(r.Context(), e); err != nil {
			s.logger.Error("❌ Failed to process push event: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		s.logger.Success("✨ Push event processed successfully")

	default:
		s.logger.Info("ℹ️ Ignoring unsupported event type: %s", github.WebHookType(r))
	}

	w.WriteHeader(http.StatusOK)
}

// checkRateLimit checks if the request should be allowed based on rate limiting
func (s *Server) checkRateLimit(ctx context.Context) error {
	s.limiter.mu.Lock()
	defer s.limiter.mu.Unlock()

	if err := s.limiter.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limit exceeded: %w", err)
	}
	return nil
}

// processPushEvent processes a GitHub push event and creates a PR if needed
func (s *Server) processPushEvent(ctx context.Context, event *github.PushEvent) error {
	// Check rate limit before processing
	if err := s.checkRateLimit(ctx); err != nil {
		s.logger.Error("❌ Rate limit check failed: %v", err)
		return err
	}

	s.logger.Loading("🔄 Processing push event...")

	// Get stored config
	s.mu.RLock()
	config := s.config
	s.mu.RUnlock()

	// Get commit info
	branch := strings.TrimPrefix(*event.Ref, "refs/heads/")
	commitMsg := *event.HeadCommit.Message
	commitSHA := *event.HeadCommit.ID

	s.logger.Info("📝 Processing commit: %s", commitSHA)
	s.logger.Info("📝 Message: %s", commitMsg)

	// Get repository info
	repoInfo := ai.RepoInfo{
		BranchName:    branch,
		CommitMessage: commitMsg,
		Changes:       make(map[string]ai.Change),
	}

	// Generate PR content
	s.logger.Loading("🤖 Generating PR content...")
	prContent, err := s.generator.GeneratePR(ctx, repoInfo)
	if err != nil {
		s.logger.Error("❌ Failed to generate PR: %v", err)
		return fmt.Errorf("failed to generate PR: %w", err)
	}

	// Create PR
	s.logger.Loading("📝 Creating PR...")
	pr := &github.NewPullRequest{
		Title:               github.String(prContent.Title),
		Body:                github.String(prContent.Description),
		Head:                github.String(branch),
		Base:                github.String(config.DefaultBranch),
		MaintainerCanModify: github.Bool(true),
	}

	_, err = s.github.CreatePullRequest(ctx, config.Owner, config.Name, pr)
	if err != nil {
		s.logger.Error("❌ Failed to create PR: %v", err)
		return fmt.Errorf("failed to create PR: %w", err)
	}

	s.logger.Success("✨ PR created successfully")
	return nil
}

// validateState ensures all required components are initialized
func (s *Server) validateState() error {
	if s.logger == nil {
		return fmt.Errorf("logger not initialized")
	}
	if s.generator == nil {
		return fmt.Errorf("AI generator not initialized")
	}
	if s.github == nil {
		return fmt.Errorf("GitHub client not initialized")
	}
	if s.hooks == nil {
		return fmt.Errorf("hooks manager not initialized")
	}
	if s.limiter == nil {
		return fmt.Errorf("rate limiter not initialized")
	}
	return nil
}
