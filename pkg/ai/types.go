package ai

// RepoInfo contains repository information
type RepoInfo struct {
	BranchName    string
	CommitMessage string
	Changes       map[string]Change
}

// Change represents a file change
type Change struct {
	Path     string
	Content  string
	IsDelete bool
}

// PRContent contains pull request content
type PRContent struct {
	Title       string
	Description string
}
