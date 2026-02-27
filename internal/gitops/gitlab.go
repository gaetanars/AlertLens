package gitops

import (
	"context"
	"fmt"
	"net/http"

	gitlab "github.com/xanzy/go-gitlab"
)

// GitLabPusher pushes files to GitLab using the GitLab API.
type GitLabPusher struct {
	client *gitlab.Client
}

// NewGitLabPusher creates a GitLabPusher with the given personal access token and base URL.
func NewGitLabPusher(token, baseURL string) (*GitLabPusher, error) {
	opts := []gitlab.ClientOptionFunc{}
	if baseURL != "" && baseURL != "https://gitlab.com" {
		opts = append(opts, gitlab.WithBaseURL(baseURL))
	}
	client, err := gitlab.NewClient(token, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating GitLab client: %w", err)
	}
	return &GitLabPusher{client: client}, nil
}

// Push creates or updates a file in a GitLab repository.
func (p *GitLabPusher) Push(ctx context.Context, opts PushOptions, content []byte) (*PushResult, error) {
	commitMsg := opts.CommitMessage
	if commitMsg == "" {
		commitMsg = "Update alertmanager config via AlertLens"
	}

	// Check if the file exists. We must distinguish a genuine 404 (file absent)
	// from any other error (auth failure, network issue, etc.) to avoid silently
	// creating a file when the real problem is something else entirely.
	_, resp, err := p.client.RepositoryFiles.GetFile(opts.Repo, opts.FilePath,
		&gitlab.GetFileOptions{Ref: gitlab.Ptr(opts.Branch)})

	if err != nil {
		if resp == nil || resp.StatusCode != http.StatusNotFound {
			return nil, fmt.Errorf("checking file existence on GitLab: %w", err)
		}
		// 404 — file does not exist yet: create it.
		createOpts := &gitlab.CreateFileOptions{
			Branch:        gitlab.Ptr(opts.Branch),
			Content:       gitlab.Ptr(string(content)),
			CommitMessage: gitlab.Ptr(commitMsg),
		}
		if opts.AuthorName != "" {
			createOpts.AuthorName = gitlab.Ptr(opts.AuthorName)
			createOpts.AuthorEmail = gitlab.Ptr(opts.AuthorEmail)
		}
		fileInfo, _, createErr := p.client.RepositoryFiles.CreateFile(opts.Repo, opts.FilePath, createOpts)
		if createErr != nil {
			return nil, fmt.Errorf("creating file on GitLab: %w", createErr)
		}
		// COR-06: GitLab's CreateFile response provides Branch, not a commit SHA.
		_ = fileInfo
		return &PushResult{}, nil
	}

	// File exists — update it.
	updateOpts := &gitlab.UpdateFileOptions{
		Branch:        gitlab.Ptr(opts.Branch),
		Content:       gitlab.Ptr(string(content)),
		CommitMessage: gitlab.Ptr(commitMsg),
	}
	if opts.AuthorName != "" {
		updateOpts.AuthorName = gitlab.Ptr(opts.AuthorName)
		updateOpts.AuthorEmail = gitlab.Ptr(opts.AuthorEmail)
	}
	fileInfo, _, updateErr := p.client.RepositoryFiles.UpdateFile(opts.Repo, opts.FilePath, updateOpts)
	if updateErr != nil {
		return nil, fmt.Errorf("updating file on GitLab: %w", updateErr)
	}
	// COR-06: UpdateFile also returns Branch, not a commit SHA.
	_ = fileInfo
	return &PushResult{}, nil
}
