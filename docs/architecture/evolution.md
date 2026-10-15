# Evolving CML deliberately

This is the boundary for the next stages, not new language syntax. The current
release remains a CML 6.12 implementation. Preparatory APIs preserve documents
and identify diagnostics; they do not add a permissive dialect.

## Semantic areas

| Area | Authority | Examples |
| --- | --- | --- |
| Domain | Meaning and model boundaries | CML contexts, aggregates, states, domain events |
| Contracts | What crosses model boundaries | APIs, integration contracts, versions, transformations |
| Implementation | Where a model executes and stores data | Applications, codebases, stores, channels |
| Evidence | Why an assertion is believed | Code/config bindings, observations, provenance, unknowns |
| Policy | What changes are acceptable | Constraints, exceptions, baseline, enforcement |

Domain and implementation are related many-to-many. Several processes realizing
one context is not intrinsically a violation. A database is not an aggregate;
a message is not automatically a domain event; a network call is not a DDD
relationship. Preserve these distinctions before naming new syntax.

Existing CML names retain their meanings: Application is a bounded context's
application layer, and Service is a domain/application service declaration. A
future running program must not silently replace either concept.

OpenSpec behavior, ADR rationale and independent acceptance tests remain linked
authorities. Do not copy their entire contents into CML. Product capabilities
may become a small optional linking area when a real navigation need emerges.

## Assertions and discrepancies

Distinguish declared architecture, observed implementation, hypotheses, target
design and constraints. Evidence carries its source/revision and uncertainty;
absence of an observed edge does not prove absence of interaction. Discrepancies
are findings, not a reason to overwrite declared design with extracted facts.

Represent an unwanted implementation edge as a fact, then analyze its relation
to the domain model. Do not globally disable syntax or resolution checks to
make legacy systems describeable. An explicit unknown/external reference is a
future modeling decision, not permission to accept misspelled identifiers.

## Pipeline and dependency direction

Current code:

- cml.Model: existing syntax-oriented model, printing and legacy algorithms.
- cml.Workspace: source documents plus ordered import edges; no filesystem writes.
- Workspace.Flatten: explicitly legacy composition for existing consumers.
- validation rules: identities/defaults and diagnostic emission, separate from traversal.
- gen/refactor/cli: existing consumers, without extension-specific fields.

Future direction:

    source documents -> workspace -> resolved, typed semantic model
                                      -> analyses -> findings -> project policy
                                      -> projections / generator-specific checks

Keep source syntax and resolved references distinct. A future semantic model
must retain source links without replacing the document graph. Do not turn Model
into an arbitrary node/edge/property container. Domain logic must not import
runtime/CLI adapters; policies consume findings rather than changing facts.

The four rule categories added now are descriptive. Existing CML validation is
not yet separated into an independent symbol resolver and policy phases. For
example cross-context restrictions still affect reference candidate selection.
This limitation is intentional compatibility, not a claim of completed resolution
separation. Generator eligibility is also still legacy behavior.

## Admission checklist for an extension

Before a new syntax proposal, document:

1. The concrete user question and a small real example.
2. Its semantic area, authority and distinction from existing concepts.
3. Identity, scope, allowed target kinds and multiplicities.
4. Incomplete/legacy reality and how it is represented without false certainty.
5. Which facts are authored and which are derived (one authority per fact).
6. Diagnostics, generator requirements and an explainable useful projection/query.
7. Compatibility mode/version, export loss and migration impact.

Reject redundant state and unconstrained semantic property bags. Keep extensions
optional and explicit; do not start with a plugin language or universal EA model.

## Staged language evolution

1. **Current foundation:** unchanged CML syntax and defaults, document graph,
   rule identities, compatibility corpus and explicit boundaries.
2. **Additive extension:** explicitly selected/versioned extension surface;
   ordinary CML keeps its current meaning. Begin with context realization and
   data access. Contract/evidence support follows demonstrated needs.
3. **Independent DSL:** the first deliberate grammar/semantic incompatibility
   requires a named dialect/version boundary, migration notes/tooling, and an
   explicit CML import/export contract. Never present a breaking dialect as
   silently compatible CML. Choosing its name/format is deferred.

Export must report unsupported/lost information. Core CML tests and extension
cases stay separate. Do not silently reinterpret existing IDs, Application,
Service, DDD relationship patterns or lifecycle syntax.

## Next bounded slice (not implemented)

Model application <-> context realization and application -> store/record access.
Exercise a modular monolith, API+worker for one context, shared database with
isolated tables, and a genuine cross-context write. Establish fact vs finding
before adding contracts, product hierarchy, MCP editing or architectural policies.
