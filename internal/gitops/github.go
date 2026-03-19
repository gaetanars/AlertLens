package gitops

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v66/github"
	"golang.org/x/oauth2"
)

// GitHubPusher pushes files to GitHub using the GitHub API.
type GitHubPusher struct {
	client *github.Client
}

// NewGitHubPusher creates a GitHubPusher with the given personal access token.
func NewGitHubPusher(token string) *GitHubPusher {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	return &GitHubPusher{client: github.NewClient(tc)}
}

// Push creates or updates a file in a GitHub repository.
func (p *GitHubPusher) Push(ctx context.Context, opts PushOptions, content []byte) (*PushResult, error) {
	parts := strings.SplitN(opts.Repo, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid GitHub repo format %q: expected owner/repo", opts.Repo)
	}
	owner, repo := parts[0], parts[1]

	// Check if the file already exists to get its SHA (required for update).
	// GetContents returns (fileContent, dirContent, response, error).
	var fileSHA string
	existing, _, _, err := p.client.Repositories.GetContents(ctx, owner, repo, opts.FilePath,
		&github.RepositoryContentGetOptions{Ref: opts.Branch})
	if err == nil && existing != nil {
		fileSHA = existing.GetSHA()
	}

	commitMsg := opts.CommitMessage
	if commitMsg == "" {
		commitMsg = "Update alertmanager config via AlertLens"
	}

	fileOpts := &github.RepositoryContentFileOptions{
		Message: &commitMsg,
		Content: content,
		Branch:  &opts.Branch,
	}
	if fileSHA != "" {
		fileOpts.SHA = &fileSHA
	}
	if opts.AuthorName != "" {
		fileOpts.Author = &github.CommitAuthor{
			Name:  &opts.AuthorName,
			Email: &opts.AuthorEmail,
		}
	}

	resp, _, err := p.client.Repositories.UpdateFile(ctx, owner, repo, opts.FilePath, fileOpts)
	if err != nil {
		return nil, fmt.Errorf("pushing to GitHub: %w", err)
	}

	result := &PushResult{
		CommitSHA: resp.GetSHA(),
		HTMLURL:   resp.GetHTMLURL(),
	}
	return result, nil
}
