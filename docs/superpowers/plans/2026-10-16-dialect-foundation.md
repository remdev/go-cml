# Dialect Foundation Implementation Plan

> **For agentic workers:** Use superpowers:executing-plans to implement task-by-task in this session. User authorized execution and a separate branch.

**Goal:** Prepare compatible CML APIs and repository boundaries for future DSL evolution without adding syntax.

**Architecture:** Retain the syntax model and existing behavior. Introduce a document workspace beneath ParseFile and machine-readable rule identities beneath Validate. Freeze current behavior before editing production code.

**Tech Stack:** Go 1.27, standard library, existing Go tests and golangci-lint.

**Spec:** ../specs/2026-10-16-dialect-foundation.md

## Global Constraints

- No new grammar, dialect, flags, severity profiles or external dependencies.
- Preserve existing output, diagnostics and import composition semantics.
- Work on codex/dialect-foundation; source checkout initially clean.
- Do not change conformance expected.json or refresh captured baseline.

## Review Focus

- Diamond/repeated imports: shared document identity, legacy duplicate composition.
- Caller edits a flattened top-level slice: document graph remains intact; nodes are shared by contract.
- Missing/syntax-invalid imports: fail without returning partial workspaces.
- Repeated loads after filesystem edits: no stale package cache.
- Stable rule metadata: no unregistered emission, default severity mismatch or writable catalog leak.

## Task 1: Freeze the compatibility boundary

Files: cml/compatibility_test.go, testdata/compatibility/baseline.json.
- [x] Capture legacy printer hashes and diagnostic fields from the unmodified baseline.
- [x] Run the characterization test and the existing suite; verify it detects an intentional temporary printer mutation, then restore.

## Task 2: Preserve documents and import graph

Files: cml/workspace.go, cml/workspace_test.go; modify cml/parser.go.
Interfaces: LoadWorkspace(path string) (*Workspace, error), (*Workspace).Flatten() (*Model, error); Workspace.Entry and Documents; Document.Path, Source, Model, Imports; DocumentImport.Declaration and Target.
- [x] Write workspace tests covering the review-focus cases; run and observe missing API failure.
- [x] Extract import loading into workspace.go; delegate ParseFile to loading and Flatten while preserving legacy composition.
- [x] Run focused tests, compatibility baseline and full suite.

## Task 3: Identify validation rules

Files: cml/diagnostic.go, cml/rules.go, cml/rules_test.go; modify cml/validate.go.
Interfaces: RuleCode, RuleCategory, Rule; LookupRule(code RuleCode) (Rule, bool), Rules() []Rule; ValidateDetailed() []RuleDiagnostic adds Code and Category without changing Diagnostic. Existing Diagnostic.String stays unchanged.
- [x] Write tests for known resolution/semantic/structural/advisory findings, catalog isolation and corpus-wide metadata consistency; observe missing API failure.
- [x] Assign explicit stable codes to existing checks and move message templates/default severities to a private typed catalog. Separate diagnostic support from validation traversal.
- [x] Run focused tests, unchanged baseline and full suite.

## Task 4: Document and verify

Files: docs/architecture/evolution.md, docs/compatibility.md, README.md, README.ru.md; CI verification command if needed.
- [x] Document ownership, API contracts, compatibility boundary, staged future DSL versioning and remaining work.
- [x] Run go test ./... -count=1, go test -race ./..., go vet ./..., golangci-lint run and gofmt check.
- [x] Review the diff independently; repair material findings and repeat relevant verification.
- [x] Transfer verified commits to the original repository on the requested separate branch, reverify there and leave a clean checkout.
