package ai

// RepoInfo contains all relevant repository information
type RepoInfo struct {
	Files            []string          `json:"files"`
	Changes          map[string]Change `json:"changes"`
	CommitMessage    string            `json:"commit_message"`
	BranchName       string            `json:"branch_name"`
	ContributingFile string            `json:"contributing_file,omitempty"`
}

// Change represents a file change
type Change struct {
	Path     string   `json:"path"`
	Added    []string `json:"added"`
	Removed  []string `json:"removed"`
	Modified []string `json:"modified"`
}

// PRContent represents generated PR content
type PRContent struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}
