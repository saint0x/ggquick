package github

import (
	"net/http"
	"os"
	"testing"

	"github.com/saint0x/ggquick/pkg/log"
)

// mockHTTPClient implements http.Client for testing
type mockHTTPClient struct {
	response *http.Response
	err      error
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.response, m.err
}

func TestParseRepoURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
		wantError bool
	}{
		{
			name:      "HTTPS URL",
			url:       "https://github.com/owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantError: false,
		},
		{
			name:      "SSH URL",
			url:       "git@github.com:owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantError: false,
		},
		{
			name:      "Simple URL",
			url:       "https://github.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantError: false,
		},
		{
			name:      "Invalid URL",
			url:       "not-a-url",
			wantError: true,
		},
		{
			name:      "Invalid Path",
			url:       "https://github.com/invalid",
			wantError: true,
		},
	}

	logger := log.New(false)
	client := New(logger)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := client.ParseRepoURL(tt.url)

			if tt.wantError {
				if err == nil {
					t.Errorf("ParseRepoURL() error = nil, want error")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseRepoURL() error = %v, want nil", err)
				return
			}

			if owner != tt.wantOwner {
				t.Errorf("ParseRepoURL() owner = %v, want %v", owner, tt.wantOwner)
			}

			if repo != tt.wantRepo {
				t.Errorf("ParseRepoURL() repo = %v, want %v", repo, tt.wantRepo)
			}
		})
	}
}

func TestNew(t *testing.T) {
	// Save original token and restore after test
	origToken := os.Getenv("GITHUB_TOKEN")
	defer os.Setenv("GITHUB_TOKEN", origToken)

	tests := []struct {
		name      string
		token     string
		wantError bool
	}{
		{
			name:      "Valid token",
			token:     "test-token",
			wantError: false,
		},
		{
			name:      "Empty token",
			token:     "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("GITHUB_TOKEN", tt.token)
			logger := log.New(false)
			client := New(logger)

			if tt.wantError {
				if client != nil {
					t.Error("New() returned non-nil client when error expected")
				}
				return
			}

			if client == nil {
				t.Error("New() returned nil client")
				return
			}

			if client.client == nil {
				t.Error("New() client.client is nil")
			}

			if client.logger != logger {
				t.Error("New() client.logger not set correctly")
			}
		})
	}
}
