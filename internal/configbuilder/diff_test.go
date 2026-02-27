package configbuilder

import (
	"testing"
)

// ─── GenerateDiff ─────────────────────────────────────────────────────────────

func TestGenerateDiff_Identical_NoChanges(t *testing.T) {
	content := "route:\n  receiver: 'null'\nreceivers:\n  - name: 'null'\n"
	hunks, hasChanges := GenerateDiff(content, content)

	if hasChanges {
		t.Error("expected hasChanges=false for identical inputs")
	}
	for _, h := range hunks {
		if h.Type != DiffHunkEqual {
			t.Errorf("expected only equal hunks, got type %q: %q", h.Type, h.Content)
		}
	}
}

func TestGenerateDiff_Identical_Empty(t *testing.T) {
	hunks, hasChanges := GenerateDiff("", "")
	if hasChanges {
		t.Error("expected hasChanges=false for two empty strings")
	}
	if len(hunks) != 0 {
		t.Errorf("expected 0 hunks for empty inputs, got %d", len(hunks))
	}
}

func TestGenerateDiff_PureInsertion(t *testing.T) {
	current := ""
	proposed := "route:\n  receiver: 'null'\n"
	hunks, hasChanges := GenerateDiff(current, proposed)

	if !hasChanges {
		t.Error("expected hasChanges=true when content is added")
	}
	hasInsert := false
	for _, h := range hunks {
		if h.Type == DiffHunkInsert {
			hasInsert = true
		}
		if h.Type == DiffHunkDelete {
			t.Errorf("unexpected delete hunk for pure insertion: %q", h.Content)
		}
	}
	if !hasInsert {
		t.Error("expected at least one insert hunk")
	}
}

func TestGenerateDiff_PureDeletion(t *testing.T) {
	current := "route:\n  receiver: 'null'\n"
	proposed := ""
	hunks, hasChanges := GenerateDiff(current, proposed)

	if !hasChanges {
		t.Error("expected hasChanges=true when content is removed")
	}
	hasDelete := false
	for _, h := range hunks {
		if h.Type == DiffHunkDelete {
			hasDelete = true
		}
		if h.Type == DiffHunkInsert {
			t.Errorf("unexpected insert hunk for pure deletion: %q", h.Content)
		}
	}
	if !hasDelete {
		t.Error("expected at least one delete hunk")
	}
}

func TestGenerateDiff_LineChanged_BothInsertAndDelete(t *testing.T) {
	current := "route:\n  receiver: 'old-receiver'\n"
	proposed := "route:\n  receiver: 'new-receiver'\n"
	hunks, hasChanges := GenerateDiff(current, proposed)

	if !hasChanges {
		t.Error("expected hasChanges=true for modified content")
	}
	var types []DiffHunkType
	for _, h := range hunks {
		types = append(types, h.Type)
	}

	hasInsert, hasDelete := false, false
	for _, typ := range types {
		if typ == DiffHunkInsert {
			hasInsert = true
		}
		if typ == DiffHunkDelete {
			hasDelete = true
		}
	}
	if !hasInsert {
		t.Error("expected an insert hunk in a modification diff")
	}
	if !hasDelete {
		t.Error("expected a delete hunk in a modification diff")
	}
}

func TestGenerateDiff_ContentIsPreservedInHunks(t *testing.T) {
	current := "a: 1\n"
	proposed := "a: 1\nb: 2\n"
	hunks, _ := GenerateDiff(current, proposed)

	// Reconstruct the proposed string from equal + insert hunks.
	var reconstructed string
	for _, h := range hunks {
		if h.Type == DiffHunkEqual || h.Type == DiffHunkInsert {
			reconstructed += h.Content
		}
	}
	if reconstructed != proposed {
		t.Errorf("reconstructed proposed mismatch:\n  want: %q\n  got:  %q", proposed, reconstructed)
	}
}

func TestGenerateDiff_HunkTypes_AreValid(t *testing.T) {
	validTypes := map[DiffHunkType]bool{
		DiffHunkEqual:  true,
		DiffHunkInsert: true,
		DiffHunkDelete: true,
	}
	hunks, _ := GenerateDiff("foo: bar\n", "foo: baz\n")
	for _, h := range hunks {
		if !validTypes[h.Type] {
			t.Errorf("unexpected hunk type: %q", h.Type)
		}
	}
}
