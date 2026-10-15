package cml

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

// Workspace retains the documents reachable from one CML entry file. Documents
// are ordered by first discovery, following imports in source order. A cleaned
// absolute path identifies a document; symlink aliases are not unified.
// Values are caller-owned and should be treated as read-only while composing.
type Workspace struct {
	Entry     *Document
	Documents []*Document
}

// Document retains exact source bytes and a local, unflattened syntax model.
// Imported declarations are available through Imports, not inserted in Model.
type Document struct {
	Path    string
	Source  []byte
	Model   *Model
	Imports []DocumentImport
}

// DocumentImport retains each import occurrence, its location, and its target.
// Repeated or diamond imports point to the same Document within a workspace.
type DocumentImport struct {
	Declaration Import
	Target      *Document
}

// LoadWorkspace reads and parses an entry file and its imports. It preserves
// the document graph independently of legacy single-model composition. Shared
// files are read once per call; loads do not share caches. Filesystem changes
// during a load are not an atomic snapshot. On any load error it returns nil.
func LoadWorkspace(path string) (*Workspace, error) {
	return loadWorkspace(path, nil)
}

// finish runs after each document subtree, before loading the next sibling.
// ParseFile uses this to preserve legacy composition error precedence.
func loadWorkspace(path string, finish func(*Document) error) (*Workspace, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	ws := &Workspace{}
	loaded := map[string]*Document{}
	active := map[string]bool{}
	var load func(string) (*Document, error)
	load = func(path string) (*Document, error) {
		path = filepath.Clean(path)
		if active[path] {
			return nil, fmt.Errorf("%s: cyclic import", path)
		}
		if doc := loaded[path]; doc != nil {
			return doc, nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		model, err := Parse(path, src)
		if err != nil {
			return nil, err
		}
		doc := &Document{Path: path, Source: src, Model: model}
		loaded[path] = doc
		ws.Documents = append(ws.Documents, doc)
		active[path] = true
		defer delete(active, path)
		for _, imp := range model.Imports {
			next := imp.URI
			if !filepath.IsAbs(next) {
				next = filepath.Join(filepath.Dir(path), next)
			}
			child, err := load(next)
			if err != nil {
				return nil, err
			}
			doc.Imports = append(doc.Imports, DocumentImport{Declaration: imp, Target: child})
		}
		if finish != nil {
			if err := finish(doc); err != nil {
				return nil, err
			}
		}
		return doc, nil
	}
	ws.Entry, err = load(abs)
	if err != nil {
		return nil, err
	}
	return ws, nil
}

// Flatten composes the legacy Model: imported declarations precede local ones,
// each import occurrence is expanded, only the entry imports are retained, and
// at most one ContextMap is allowed. Thus diamond imports retain the existing
// duplicate-declaration behavior, even though documents have shared identity.
// Top-level slices are fresh, but declaration nodes are shared with Documents;
// this is a read view, not a deep copy or an editing transaction.
func (ws *Workspace) Flatten() (*Model, error) {
	if ws == nil || ws.Entry == nil {
		return nil, fmt.Errorf("workspace has no entry document")
	}
	active := map[*Document]bool{}
	var flatten func(*Document) (*Model, error)
	flatten = func(doc *Document) (*Model, error) {
		if doc == nil || doc.Model == nil {
			return nil, fmt.Errorf("workspace contains an incomplete document")
		}
		if active[doc] {
			return nil, fmt.Errorf("%s: cyclic import", doc.Path)
		}
		active[doc] = true
		defer delete(active, doc)
		var imported []*Model
		for _, imp := range doc.Imports {
			child, err := flatten(imp.Target)
			if err != nil {
				return nil, err
			}
			imported = append(imported, child)
		}
		return composeDocument(doc, imported)
	}
	return flatten(ws.Entry)
}

// composeDocument preserves the legacy composition order and map restriction.
func composeDocument(doc *Document, imported []*Model) (*Model, error) {
	merged := &Model{File: doc.Model.File, Comment: doc.Model.Comment}
	for _, child := range imported {
		if err := merged.absorb(child); err != nil {
			return nil, err
		}
	}
	if err := merged.absorb(doc.Model); err != nil {
		return nil, err
	}
	merged.Imports = slices.Clone(doc.Model.Imports)
	return merged, nil
}
