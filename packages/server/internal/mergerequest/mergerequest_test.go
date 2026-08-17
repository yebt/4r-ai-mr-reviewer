package mergerequest

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		name      string
		in        string
		wantTitle string
		wantDesc  string
		wantErr   bool
	}{
		{
			name:      "title and description",
			in:        "Add retry to the uploader\n\n## Summary\nRetries transient failures.",
			wantTitle: "Add retry to the uploader",
			wantDesc:  "## Summary\nRetries transient failures.",
		},
		{
			name:      "extra leading blank lines",
			in:        "\n\nFix the flaky test\n\nBody here.",
			wantTitle: "Fix the flaky test",
			wantDesc:  "Body here.",
		},
		{
			name:      "markdown heading title is stripped",
			in:        "# Add caching\n\nDetails.",
			wantTitle: "Add caching",
			wantDesc:  "Details.",
		},
		{
			name:      "labelled parts are stripped",
			in:        "Title: Bump deps\n\nDescription: Routine upgrade.",
			wantTitle: "Bump deps",
			wantDesc:  "Routine upgrade.",
		},
		{
			name:      "whole reply wrapped in a fence",
			in:        "```\nRefactor parser\n\nSplit into functions.\n```",
			wantTitle: "Refactor parser",
			wantDesc:  "Split into functions.",
		},
		{
			name:      "title only",
			in:        "Tidy imports",
			wantTitle: "Tidy imports",
			wantDesc:  "",
		},
		{
			name:    "empty output errors",
			in:      "   \n  \n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Parse(%q) = %+v, want error", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tt.in, err)
			}
			if got.Title != tt.wantTitle {
				t.Errorf("title = %q, want %q", got.Title, tt.wantTitle)
			}
			if got.Description != tt.wantDesc {
				t.Errorf("description = %q, want %q", got.Description, tt.wantDesc)
			}
		})
	}
}
