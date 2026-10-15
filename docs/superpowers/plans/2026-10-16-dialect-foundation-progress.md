# Execution ledger — 2026-10-16-dialect-foundation.md

[English](2026-10-16-dialect-foundation-progress.md) · [Русский](2026-10-16-dialect-foundation-progress.ru.md)

- Baseline: 351c92b; clean original checkout; isolated clone at /private/tmp/go-cml-foundation on codex/dialect-foundation.
- Baseline suite: go test ./... -count=1 PASS (all six packages).
- Ruling: user explicitly requested planning and implementation; proceed inline without another approval handoff. No new syntax.
- Pre-flight: workspace changes feed compatibility tests; rule changes feed the same tests. Preserve the legacy boundary in both tasks.

- Task 1 complete: 328 baseline entries captured before implementation; deliberate printer mutation failed the baseline test; restored printer and green suite.
- Task 2 complete: missing LoadWorkspace API observed RED; document graph and legacy composition implemented; full suite PASS.
- Task 3 complete: missing rule APIs observed RED; 49 explicit rule identities/categories/defaults added; legacy diagnostic baseline and full suite PASS.
- Ruling: preserve diamond/repeated-import composition, despite shared graph identities; deduplication would be a separate behavior change.
- Ruling: keep current mutable syntax nodes, document shallow sharing; immutable resolved IR and filesystem transactions remain future work.
- Task 4 in progress: architecture/compatibility docs written; full verification and independent review pending.

- Independent review found one P2 compatibility issue: loading all documents before legacy composition changed first-error precedence for a nested map conflict followed by missing sibling. Regression reproduced RED; legacy per-document composition restored while LoadWorkspace stays graph-only. No other actionable findings; reviewer checked all 61 emissions against prior templates/severities/arguments.
- CI now runs race detector and vet in addition to existing full tests and lint.

- Ruling: preserve the exported Diagnostic struct shape as well as CML compatibility. An unkeyed-literal regression reproduced RED with in-place fields; metadata moved to RuleDiagnostic and ValidateDetailed. Legacy JSON stays unchanged. Both validation APIs are cross-checked across the corpus, independently pinned by the pre-change baseline.

- Task 4 complete: independent reviewer rechecked the import-precedence fix and legacy Diagnostic API preservation; no remaining material findings.
- Verified in the original repository on codex/dialect-foundation: uncached full suite, race suite, vet, configured golangci-lint (0 issues), gofmt and diff whitespace checks all PASS. Main remains at 351c92b1f81081a98381ab76faec7412d987c37e.
- Completion audit: no grammar/dialect/CLI changes; 328-file frozen baseline, 273 existing conformance cases and 55 corpus cases unchanged; document graph and legacy composition tests pass; detailed/legacy diagnostic values agree; compatibility/evolution documentation and CI gates delivered.
