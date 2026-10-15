# Compatibility contract

## Reference and scope

The reference is Context Mapper DSL 6.12. The preparatory baseline is this
repository at commit 351c92b1f81081a98381ab76faec7412d987c37e. Existing corpus and
conformance tests are evidence of tested behavior, not proof of complete parity
with every JVM tool, generator or Freemarker feature.

No new language syntax, dialect, CLI command/flag, policy override or dependency
is introduced by the foundation work. Parse, ParseFile, printing, validation
text/default severity and the CLI keep their existing behavior on the corpus.

## Compatibility evidence

- 273 existing conformance fixtures retain their expected validity.
- 55 existing corpus fixtures retain parse/print tests.
- testdata/compatibility/baseline.json was captured from the unmodified commit
  above. It pins parse errors, printed-model SHA-256 values and legacy diagnostic
  messages, severities, positions and ordering across all 328 files.
- cml/compatibility_test.go compares against that frozen baseline. It was checked
  against a deliberate temporary printer mutation before production changes.
- Existing CLI, generator and refactoring tests remain in the full suite.

The hashes pin normalized printer output, not original whitespace/comments in
full fidelity. Workspace.Source retains exact bytes. This is not a JVM
side-by-side differential runner. A future deliberate change must explain its
compatibility/version impact before changing fixtures; never refresh a baseline
just to turn a failed test green.

## Workspace API

```go
workspace, err := cml.LoadWorkspace("architecture.cml")
// Check err before accessing workspace.
// workspace.Entry.Model contains only the entry document's local declarations.
// workspace.Entry.Imports preserves each source import and its target.
model, err := workspace.Flatten()
// Check err: a loadable graph can still contain multiple context maps.
```

Each cleaned absolute lexical path is loaded once per call. Documents use
first-discovery order; import edges use source order. There is no process-global
cache. Missing files, cyclic imports and syntax errors return no partial
workspace. Loading is read-only but is not an atomic filesystem snapshot.
Symlink aliases, case-folded aliases and hard links are not unified.

Flatten preserves legacy order: expand imports in declaration order, then local
declarations. Repeated/diamond imports intentionally repeat declarations and can
produce duplicate-name diagnostics (or a multiple-map composition error). The
single ContextMap rule is retained, including ParseFile error precedence: a
nested composition failure is returned before visiting a later sibling import. Changing this is a separate compatibility
decision, not an implicit deduplication optimization.

Composition does not change document slices. The result shares declaration
nodes with documents: callers own the returned objects and must treat them as
read-only when using both views. It is not a deep clone, immutable semantic IR,
editing transaction, or promise of concurrent mutation safety. Use another load
for independent editing. ParseFile still returns the flattened view.

## Diagnostic API

Diagnostic and Model.Validate keep their original Go shape and behavior,
including unkeyed literals and default JSON serialization. Model.ValidateDetailed
returns RuleDiagnostic values embedding the original Diagnostic plus Code and
Category. Both methods perform the same checks and use the same default severity
and ordering; only the detailed result adds metadata.

Rules() returns a code-sorted, caller-independent catalog; LookupRule returns
metadata by stable code. Codes are machine identities and must not be reassigned
to unrelated checks. Categories: structure, resolution, semantics, advice.
DefaultSeverity is the unchanged compatibility default; category does not control
whether a finding blocks validation. Diagnostic.String and CLI rendering retain
their old text. Syntax/load errors remain errors, not catalog rules. Rule metadata
does not yet provide configurable validation profiles. There is no new diagnostic
JSON CLI or promised detailed-result wire schema.

## Verification

```sh
go test ./... -count=1
go test -race ./...
go vet ./...
golangci-lint run --timeout=5m
test -z "$(gofmt -l .)"
```

Graphviz dot is required by the existing CLI integration tests. The module's Go
version is authoritative. Tests do not read the ignored .reference checkout and
need no upstream/network access. See architecture/evolution.md for the staged
transition to an independent DSL.
