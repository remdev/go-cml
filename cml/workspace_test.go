package cml

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeWorkspaceFile(t *testing.T, dir, name, source string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// Catches flattening away local syntax, import positions or raw comments.
func TestWorkspacePreservesDocuments(t *testing.T) {
	dir := t.TempDir()
	child := writeWorkspaceFile(t, dir, "child.cml", "// child comment\nBoundedContext Child\n")
	source := "// entry comment\nimport \"child.cml\"\nBoundedContext Entry\n"
	entry := writeWorkspaceFile(t, dir, "entry.cml", source)
	ws, err := LoadWorkspace(entry)
	if err != nil {
		t.Fatal(err)
	}
	if ws.Entry.Path != entry || len(ws.Documents) != 2 || ws.Documents[0] != ws.Entry {
		t.Fatalf("documents: %+v", ws)
	}
	if !bytes.Equal(ws.Entry.Source, []byte(source)) {
		t.Fatal("source bytes lost")
	}
	if len(ws.Entry.Model.BoundedContexts) != 1 || ws.Entry.Model.BoundedContexts[0].Name != "Entry" {
		t.Fatal("entry syntax was flattened")
	}
	edge := ws.Entry.Imports[0]
	if edge.Declaration.URI != "child.cml" || edge.Declaration.Pos.Line != 2 || edge.Target.Path != child {
		t.Fatalf("edge: %+v", edge)
	}
	if edge.Target.Model.BoundedContexts[0].Pos.File != child {
		t.Fatal("source position changed")
	}
	flat, err := ws.Flatten()
	if err != nil {
		t.Fatal(err)
	}
	if names := workspaceNames(flat); !reflect.DeepEqual(names, []string{"Child", "Entry"}) {
		t.Fatalf("flatten order: %v", names)
	}
	// Top-level edits to the flat view must not mutate document slices.
	flat.BoundedContexts[0] = nil
	flat.Imports[0].URI = "changed"
	again, err := ws.Flatten()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(workspaceNames(again), []string{"Child", "Entry"}) || ws.Entry.Model.Imports[0].URI != "child.cml" {
		t.Fatal("composition mutated document slices")
	}
}

func workspaceNames(model *Model) []string {
	var names []string
	for _, bc := range model.BoundedContexts {
		names = append(names, bc.Name)
	}
	return names
}

// Shared documents have one identity, while legacy flattening repeats imports.
func TestWorkspaceDiamondAndRepeatedImports(t *testing.T) {
	dir := t.TempDir()
	writeWorkspaceFile(t, dir, "shared.cml", "BoundedContext Shared\n")
	writeWorkspaceFile(t, dir, "left.cml", "import \"shared.cml\"\nBoundedContext Left\n")
	writeWorkspaceFile(t, dir, "right.cml", "import \"./shared.cml\"\nBoundedContext Right\n")
	entry := writeWorkspaceFile(t, dir, "entry.cml", "import \"left.cml\"\nimport \"right.cml\"\nimport \"left.cml\"\nBoundedContext Entry\n")
	ws, err := LoadWorkspace(entry)
	if err != nil {
		t.Fatal(err)
	}
	if len(ws.Documents) != 4 {
		t.Fatalf("document count=%d", len(ws.Documents))
	}
	left, right := ws.Entry.Imports[0].Target, ws.Entry.Imports[1].Target
	if left != ws.Entry.Imports[2].Target || left.Imports[0].Target != right.Imports[0].Target {
		t.Fatal("shared imports were reparsed")
	}
	flat, err := ws.Flatten()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Shared", "Left", "Shared", "Right", "Shared", "Left", "Entry"}
	if !reflect.DeepEqual(workspaceNames(flat), want) {
		t.Fatalf("legacy order=%v", workspaceNames(flat))
	}
	if !HasErrors(flat.Validate()) {
		t.Fatal("legacy duplicate-name diagnostics disappeared")
	}
}

func TestWorkspaceLoadFailures(t *testing.T) {
	for _, tc := range []struct{ name, child, want string }{
		{"cycle", "import \"entry.cml\"\n", "cyclic import"},
		{"syntax", "BoundedContext {", "expected"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			entry := writeWorkspaceFile(t, dir, "entry.cml", "import \"child.cml\"\n")
			writeWorkspaceFile(t, dir, "child.cml", tc.child)
			ws, err := LoadWorkspace(entry)
			if err == nil || ws != nil {
				t.Fatalf("partial/success: %v %v", ws, err)
			}
			if tc.name == "cycle" && !strings.Contains(err.Error(), tc.want) {
				t.Fatal(err)
			}
			if tc.name == "syntax" {
				syntax, ok := errors.AsType[*SyntaxError](err)
				if !ok || filepath.Base(syntax.File) != "child.cml" {
					t.Fatalf("syntax origin: %v", err)
				}
			}
		})
	}
	t.Run("missing", func(t *testing.T) {
		dir := t.TempDir()
		entry := writeWorkspaceFile(t, dir, "entry.cml", "import \"missing.cml\"\n")
		ws, err := LoadWorkspace(entry)
		if ws != nil || !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("missing import: %v %v", ws, err)
		}
	})
}

func TestWorkspaceMapCompositionAndReload(t *testing.T) {
	dir := t.TempDir()
	child := writeWorkspaceFile(t, dir, "child.cml", "ContextMap Child {}\n")
	entry := writeWorkspaceFile(t, dir, "entry.cml", "import \"child.cml\"\nContextMap Entry {}\n")
	ws, err := LoadWorkspace(entry)
	if err != nil {
		t.Fatal(err)
	}
	// Loading preserves documents even if they cannot compose into one legacy map.
	if _, err := ws.Flatten(); err == nil || !strings.Contains(err.Error(), "also defines a Context Map") {
		t.Fatalf("map conflict: %v", err)
	}
	if _, err := ParseFile(entry); err == nil {
		t.Fatal("ParseFile accepted multiple maps")
	}
	writeWorkspaceFile(t, dir, "child.cml", "BoundedContext Updated\n")
	fresh, err := LoadWorkspace(entry)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Entry.Imports[0].Target.Model.Map != nil || fresh.Entry.Imports[0].Target.Model.FindBC("Updated") == nil {
		t.Fatal("stale document cache")
	}
	if ws.Entry.Imports[0].Target.Model.Map == nil || ws.Entry.Imports[0].Target.Path != child {
		t.Fatal("reload mutated prior workspace")
	}
	flat, err := fresh.Flatten()
	if err != nil || flat.Map.Name != "Entry" {
		t.Fatalf("reload: %v %v", flat, err)
	}
}

// Public graph values are mutable; malformed composition must fail, not recurse
// indefinitely or panic when a caller constructs a workspace programmatically.
func TestWorkspaceFlattenRejectsIncompleteAndCyclicGraphs(t *testing.T) {
	for _, ws := range []*Workspace{nil, {}, {Entry: &Document{}}} {
		if _, err := ws.Flatten(); err == nil {
			t.Fatal("incomplete workspace accepted")
		}
	}
	doc := &Document{Path: "cycle.cml", Model: &Model{File: "cycle.cml"}}
	doc.Imports = []DocumentImport{{Target: doc}}
	if _, err := (&Workspace{Entry: doc}).Flatten(); err == nil || !strings.Contains(err.Error(), "cyclic import") {
		t.Fatalf("cycle: %v", err)
	}
	doc.Imports = []DocumentImport{{}}
	if _, err := (&Workspace{Entry: doc}).Flatten(); err == nil {
		t.Fatal("nil import accepted")
	}
}

// Legacy ParseFile must report an earlier nested composition failure before a
// later sibling's I/O error. LoadWorkspace intentionally does not compose.
func TestParseFilePreservesNestedCompositionErrorPrecedence(t *testing.T) {
	dir := t.TempDir()
	writeWorkspaceFile(t, dir, "map.cml", "ContextMap Imported {}\n")
	writeWorkspaceFile(t, dir, "conflict.cml", "import \"map.cml\"\nContextMap Local {}\n")
	entry := writeWorkspaceFile(t, dir, "entry.cml", "import \"conflict.cml\"\nimport \"missing.cml\"\n")
	if _, err := ParseFile(entry); err == nil || !strings.Contains(err.Error(), "also defines a Context Map") {
		t.Fatalf("first legacy error: %v", err)
	}
	if _, err := LoadWorkspace(entry); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("graph load should reach missing file: %v", err)
	}
}
