# Execution ledger — 2026-10-16-dialect-foundation.md

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
