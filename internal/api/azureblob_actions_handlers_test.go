package api

import "testing"

func TestRestrictedActionConfirmation(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		confirm   string
		wantBlock bool
	}{
		{name: "archive requires apply", operation: "archive", confirm: "", wantBlock: true},
		{name: "archive accepts apply", operation: "archive", confirm: "APPLY", wantBlock: false},
		{name: "delete rejects apply", operation: "delete", confirm: "APPLY", wantBlock: true},
		{name: "delete accepts delete", operation: "delete", confirm: "DELETE", wantBlock: false},
		{name: "tagging does not need confirmation", operation: "apply_blob_metadata", confirm: "", wantBlock: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := restrictedActionConfirmation(tt.operation, tt.confirm)
			if (got != "") != tt.wantBlock {
				t.Fatalf("restrictedActionConfirmation(%q, %q) = %q, want block %v", tt.operation, tt.confirm, got, tt.wantBlock)
			}
		})
	}
}

func TestNormalizeWorkbenchOperationEnterpriseAliases(t *testing.T) {
	tests := map[string]string{
		"copy":             "copy_blob",
		"move-object":      "move_blob",
		"set_access_tier":  "set_tier",
		"retrieve":         "rehydrate_blob",
		"container_create": "create_container",
	}

	for input, want := range tests {
		if got := normalizeWorkbenchOperation(input); got != want {
			t.Fatalf("normalizeWorkbenchOperation(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestWorkbenchTargetBlobPath(t *testing.T) {
	if got := workbenchTargetBlobPath("exact/output.txt", "ignored", "source/input.txt"); got != "exact/output.txt" {
		t.Fatalf("exact target path = %q", got)
	}
	if got := workbenchTargetBlobPath("", "copied", "source/input.txt"); got != "copied/source/input.txt" {
		t.Fatalf("prefixed target path = %q", got)
	}
	if got := workbenchTargetBlobPath("", "", "source/input.txt"); got != "source/input.txt" {
		t.Fatalf("fallback target path = %q", got)
	}
}

func TestNormalizeActionExecuteOperationEnterpriseAliases(t *testing.T) {
	tests := map[string]string{
		"set_metadata": "apply_blob_metadata",
		"set-tags":     "apply_blob_index_tags",
		"copy_blob":    "migration",
		"move_prep":    "migration",
		"delete_blob":  "delete",
	}

	for input, want := range tests {
		if got := normalizeActionExecuteOperation(input); got != want {
			t.Fatalf("normalizeActionExecuteOperation(%q) = %q, want %q", input, got, want)
		}
	}
}
