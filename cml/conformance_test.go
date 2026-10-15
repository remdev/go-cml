package cml

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type conformanceCase struct {
	File  string `json:"file"`
	Valid bool   `json:"valid"`
}

// TestGrammarConformance checks CML 6.12 grammar and semantic fragments.
// A fragment is accepted only when it parses and has no semantic errors.
func TestGrammarConformance(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "testdata", "conformance")
	raw, err := os.ReadFile(filepath.Join(root, "expected.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cases []conformanceCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) != 273 {
		t.Fatalf("conformance corpus = %d, want 273", len(cases))
	}
	for _, tc := range cases {
		t.Run(tc.File, func(t *testing.T) {
			t.Parallel()
			src, err := os.ReadFile(filepath.Join(root, tc.File))
			if err != nil {
				t.Fatal(err)
			}
			model, err := Parse(tc.File, src)
			valid := err == nil && !HasErrors(model.Validate())
			if valid == tc.Valid {
				return
			}
			if err != nil {
				t.Fatalf("parse error, expected valid=%v: %v", tc.Valid, err)
			}
			var msgs []string
			for _, d := range model.Validate() {
				if d.Severity == SevError {
					msgs = append(msgs, d.Message)
				}
			}
			t.Fatalf("valid=%v expected=%v errors=%v", valid, tc.Valid, msgs)
		})
	}
}
