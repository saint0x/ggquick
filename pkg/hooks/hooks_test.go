package hooks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/saint0x/ggquick/pkg/log"
)

func setupTestRepo(t *testing.T) (string, func()) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "ggquick-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create .git directory
	gitDir := filepath.Join(tmpDir, ".git", "hooks")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create git hooks dir: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestHookInstallation(t *testing.T) {
	logger := log.New(true)
	manager := New(logger)

	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Test repo validation
	err := manager.ValidateGitRepo(repoDir)
	if err != nil {
		t.Errorf("ValidateGitRepo() error = %v", err)
	}

	// Create hooks directory
	hooksDir := filepath.Join(repoDir, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		t.Fatalf("Failed to create hooks dir: %v", err)
	}

	// Test hook installation
	err = manager.UpdateRepo(&RepoInfo{
		Path:      repoDir,
		HooksPath: hooksDir,
	})
	if err != nil {
		t.Errorf("UpdateRepo() error = %v", err)
	}

	// Verify hooks were created
	hooks := []string{"post-commit", "post-push"}
	for _, hook := range hooks {
		hookPath := filepath.Join(hooksDir, hook)
		if _, err := os.Stat(hookPath); os.IsNotExist(err) {
			t.Errorf("Hook %s was not created", hook)
			continue
		}

		// Check if hook is executable
		info, err := os.Stat(hookPath)
		if err != nil {
			t.Errorf("Failed to stat hook %s: %v", hook, err)
			continue
		}

		if info.Mode()&0111 == 0 {
			t.Errorf("Hook %s is not executable", hook)
		}

		// Read hook content
		content, err := os.ReadFile(hookPath)
		if err != nil {
			t.Errorf("Failed to read hook %s: %v", hook, err)
			continue
		}

		if len(content) == 0 {
			t.Errorf("Hook %s is empty", hook)
		}
	}
}

func TestHookRemoval(t *testing.T) {
	logger := log.New(true)
	manager := New(logger)

	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create hooks directory
	hooksDir := filepath.Join(repoDir, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		t.Fatalf("Failed to create hooks dir: %v", err)
	}

	// Install hooks first
	err := manager.UpdateRepo(&RepoInfo{
		Path:      repoDir,
		HooksPath: hooksDir,
	})
	if err != nil {
		t.Fatalf("UpdateRepo() error = %v", err)
	}

	// Test hook removal
	err = manager.RemoveHooks(hooksDir)
	if err != nil {
		t.Errorf("RemoveHooks() error = %v", err)
	}

	// Verify hooks were removed
	hooks := []string{"post-commit", "post-push"}
	for _, hook := range hooks {
		hookPath := filepath.Join(hooksDir, hook)
		if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
			t.Errorf("Hook %s was not removed", hook)
		}
	}
}

func TestValidateGitRepo(t *testing.T) {
	logger := log.New(true)
	manager := New(logger)

	tests := []struct {
		name      string
		setup     func() (string, func())
		wantError bool
	}{
		{
			name: "Valid git repo",
			setup: func() (string, func()) {
				return setupTestRepo(t)
			},
			wantError: false,
		},
		{
			name: "Invalid repo",
			setup: func() (string, func()) {
				dir, err := os.MkdirTemp("", "not-a-git-repo-*")
				if err != nil {
					t.Fatal(err)
				}
				return dir, func() { os.RemoveAll(dir) }
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, cleanup := tt.setup()
			defer cleanup()

			err := manager.ValidateGitRepo(dir)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateGitRepo() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
