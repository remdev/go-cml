package cml

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Codes identify the failed rule, independently of wording and source position.
func TestDiagnosticRuleIdentity(t *testing.T) {
	for _, tc := range []struct {
		name, src string
		code      RuleCode
		category  RuleCategory
		severity  Severity
	}{
		{"structure", "BoundedContext A\nBoundedContext A", RuleDuplicateBoundedContext, CategoryStructure, SevError},
		{"resolution", "ContextMap { contains Missing }", RuleUnknownBoundedContext, CategoryResolution, SevError},
		{"semantics", "BoundedContext A realizes B\nBoundedContext B", RuleRealizesRequiresTeam, CategorySemantics, SevError},
		{"advice", "BoundedContext Orders { Aggregate Order { Entity Order { String id key } } }", RulePrimitiveIdentifier, CategoryAdvice, SevWarning},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, err := Parse("rule.cml", []byte(tc.src))
			if err != nil {
				t.Fatal(err)
			}
			for _, d := range m.Validate() {
				if d.Code != tc.code {
					continue
				}
				if d.Category != tc.category || d.Severity != tc.severity {
					t.Fatalf("metadata: %+v", d)
				}
				rule, ok := LookupRule(d.Code)
				if !ok || rule.Category != d.Category || rule.DefaultSeverity != d.Severity {
					t.Fatalf("catalog mismatch: %+v %+v", d, rule)
				}
				return
			}
			t.Fatalf("missing rule %s: %v", tc.code, m.Validate())
		})
	}
}

func TestRuleCatalogCannotBeModifiedByCaller(t *testing.T) {
	rules := Rules()
	if len(rules) == 0 {
		t.Fatal("empty catalog")
	}
	first := rules[0]
	for i, rule := range rules {
		if rule.Code == "" || rule.Category == "" {
			t.Fatalf("incomplete: %+v", rule)
		}
		if i > 0 && rules[i-1].Code >= rule.Code {
			t.Fatal("catalog not sorted or contains duplicates")
		}
	}
	rules[0].Category = "modified"
	value, ok := LookupRule(first.Code)
	if !ok || value != first || Rules()[0] != first {
		t.Fatal("catalog leaked mutable state")
	}
	if _, ok := LookupRule("missing-rule"); ok {
		t.Fatal("unknown code resolved")
	}
}

// Catches an unregistered emission or an accidentally changed default in any
// existing parsed fixture, not only the four examples above.
func TestCorpusDiagnosticMetadata(t *testing.T) {
	count := 0
	for _, dir := range []string{"conformance", "corpus"} {
		err := filepath.WalkDir(filepath.Join("..", "testdata", dir), func(path string, e fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if e.IsDir() || !strings.HasSuffix(path, ".cml") {
				return nil
			}
			var m *Model
			if dir == "corpus" {
				m, err = ParseFile(path)
			} else {
				var src []byte
				src, err = os.ReadFile(path)
				if err == nil {
					m, err = Parse(path, src)
				}
			}
			if err != nil {
				return nil
			} // Syntax-invalid conformance cases have no semantic diagnostics.
			for _, d := range m.Validate() {
				count++
				rule, ok := LookupRule(d.Code)
				if !ok || rule.Category != d.Category || rule.DefaultSeverity != d.Severity {
					t.Errorf("%s: missing/inconsistent rule: %+v", path, d)
				}
				if strings.Contains(d.Message, "%!") {
					t.Errorf("%s: malformed rule message: %s", path, d.Message)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if count == 0 {
		t.Fatal("no diagnostics exercised")
	}
}

func TestDiagnosticStringPreservesLegacyRendering(t *testing.T) {
	d := Diagnostic{Code: RuleUnknownBoundedContext, Category: CategoryResolution, Severity: SevError, Pos: Position{File: "model.cml", Line: 2, Col: 3}, Message: "unknown context"}
	if got := d.String(); got != "model.cml:2:3: error: unknown context" {
		t.Fatal(got)
	}
	d.Pos = Position{}
	if got := d.String(); got != "error: unknown context" {
		t.Fatal(got)
	}
}
