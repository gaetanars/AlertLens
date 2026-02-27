package configbuilder

import (
	"github.com/sergi/go-diff/diffmatchpatch"
)

// DiffHunkType classifies a chunk of a diff.
type DiffHunkType string

const (
	DiffHunkEqual  DiffHunkType = "equal"
	DiffHunkInsert DiffHunkType = "insert"
	DiffHunkDelete DiffHunkType = "delete"
)

// DiffHunk is a single unit of a diff result.
// Using a structured type (rather than ANSI/HTML text) lets the frontend render
// the diff with full control over styling.
type DiffHunk struct {
	Type    DiffHunkType `json:"type"`
	Content string       `json:"content"`
}

// GenerateDiff computes a semantic diff between current and proposed YAML and
// returns it as a slice of DiffHunk values. The boolean is true when at least
// one insert or delete hunk exists.
func GenerateDiff(current, proposed string) ([]DiffHunk, bool) {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(current, proposed, false)
	dmp.DiffCleanupSemantic(diffs)

	hunks := make([]DiffHunk, 0, len(diffs))
	hasChanges := false

	for _, d := range diffs {
		var typ DiffHunkType
		switch d.Type {
		case diffmatchpatch.DiffInsert:
			typ = DiffHunkInsert
			hasChanges = true
		case diffmatchpatch.DiffDelete:
			typ = DiffHunkDelete
			hasChanges = true
		default:
			typ = DiffHunkEqual
		}
		hunks = append(hunks, DiffHunk{Type: typ, Content: d.Text})
	}

	return hunks, hasChanges
}
