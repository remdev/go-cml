# Dialect foundation

[English](2026-10-16-dialect-foundation.md) · [Русский](2026-10-16-dialect-foundation.ru.md)

## Intent

Prepare go-cml for additive language evolution and, eventually, an explicitly
versioned independent DSL. This change adds no grammar, dialect, CLI flags,
validation profiles, runtime concepts, policy engine, or generic plugin system.
The user authorized planning and implementation, on a separate branch.

## Compatibility boundary

Context Mapper DSL 6.12 remains the reference for the existing CML surface.
Preserve Parse, ParseFile, Model.Validate, Model.String, generator/refactor entry
points and existing CLI output/exit behavior. Diagnostic metadata is exposed by a separate ValidateDetailed API;
legacy text, severity, source positions and ordering remain unchanged. The exported Diagnostic shape, unkeyed literals and default JSON serialization
remain unchanged.

Preserve current import composition, including repeated declarations through
diamond imports and the single ContextMap restriction. Do not silently fix or
reinterpret these semantics as part of preparation. Future graph consumers can
inspect shared documents without importing these composition rules.

## Deliverables

1. A document workspace API retaining raw source bytes, per-file syntax models,
   ordered import edges and shared document identity. ParseFile continues to
   return its legacy flattened model. Separate loading from composition.
2. A catalog of stable validation rule codes, categories and default severities.
   Every emitted semantic diagnostic identifies its rule. Preserve the legacy
   validation behavior; no severity overrides or new permissive profile.
3. A pre-change compatibility baseline over tracked CML corpora, plus meaningful
   tests for loading, composition, diagnostic identity and catalog isolation.
4. Architecture and compatibility documentation, linked from both READMEs,
   explaining extension boundaries, known limits and the later DSL transition.
5. Existing test suite, race tests, vet, formatting and configured lint pass.

## Design choices

Keep cml.Model as the existing syntax-oriented API, not a universal architecture
IR. Workspace owns the import document graph; it does not mutate or write files.
Paths use absolute cleaned lexical identities, consistent with existing import
resolution. No symlink identity unification, package imports or concurrent-load
snapshot guarantee is introduced. Each load is independent; shared imports are
read once within a load. The API returns caller-owned mutable syntax objects.
Composition creates new top-level slices but shares declaration nodes, as
explicitly documented; it is not a deep clone or editing transaction.

Rules are categorized as structure, resolution, semantics or advice. Categories
are descriptive metadata, not permission to ignore failures. Existing checks
can mix concerns; this stage records that limitation rather than changing which
references resolve. Catalog lookups and enumeration return values, not mutable
internal definitions.

## Acceptance

The tracked corpus baseline must be captured from commit
351c92b1f81081a98381ab76faec7412d987c37e before production edits and must not be
regenerated to accommodate regressions. It pins flattened printer output and
legacy diagnostics. New tests prove document preservation, shared imports,
repeatable composition without mutation, cycles, missing/invalid imports,
multiple maps, reload after edits, and rule metadata/default consistency.
No modifications to grammar acceptance or existing conformance expectations.
