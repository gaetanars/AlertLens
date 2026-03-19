// Package gitops provides GitHub and GitLab pusher implementations for
// syncing Alertmanager configuration to a remote Git repository.
package gitops

import "context"

// PushOptions holds options for pushing a file to a remote Git repository.
type PushOptions struct {
	Repo          string // "owner/repo" for GitHub, "namespace/project" for GitLab
	Branch        string
	FilePath      string // path within the repo
	CommitMessage string
	AuthorName    string
	AuthorEmail   string
}

// PushResult contains the result of a successful push.
type PushResult struct {
	CommitSHA string `json:"commit_sha"`
	HTMLURL   string `json:"html_url,omitempty"`
}

// Pusher is the interface implemented by GitHub and GitLab clients.
type Pusher interface {
	// Push creates or updates a file in a remote Git repository.
	Push(ctx context.Context, opts PushOptions, content []byte) (*PushResult, error)
}
