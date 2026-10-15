# go-cml

[![CI](https://github.com/remdev/go-cml/actions/workflows/ci.yml/badge.svg)](https://github.com/remdev/go-cml/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

[Русский](README.ru.md)

Go implementation of [Context Mapper](https://contextmapper.org/): the CML language and the `cm` command-line tool in one module, `github.com/remdev/go-cml`.

CML follows Context Mapper DSL 6.12. It covers context maps, bounded contexts, the Sculptor tactic model, use cases, user stories, stakeholders, value registers, application flows, and coordinations. SCL, the Service Cutter user-representation language, is included as well.

## Install

```bash
go install github.com/remdev/go-cml/cmd/cm@latest
```

Or build from a checkout:

```bash
go build -o bin/cm ./cmd/cm
```

## Commands

The output directory passed to `generate` must already exist.

```bash
mkdir -p out
cm validate -i examples/ddd-sample.cml
cm generate -i examples/ddd-sample.cml -g plantuml -o out
cm generate -i examples/ddd-sample.cml -g context-map -o out
cm generate -i examples/ddd-sample.cml -g mdsl -o out
cm generate -i examples/ddd-sample.cml -g scl -o out
cm generate -i examples/ddd-sample.cml -g generic -o out -t template.ftl -f glossary.md
cm format -i examples/ddd-sample.cml
cm refactor -l
cm refactor -i examples/ddd-sample.cml -o out/shared.cml -r extract-shared-kernel CargoBookingContext VoyagePlanningContext SharedVoyage
```

`context-map` always writes a Graphviz `.gv` file. PNG and SVG are written when `dot` is on `PATH`.

`generic` accepts a subset of Freemarker: `${...}`, `<#if>`, `<#elseif>`, `<#else>`, `<#list ... as ...>`, `<#assign>`, comments, and the whitespace markers `<#lt>`, `<#rt>`, `<#t>`, and `<#nt>`. Expressions support `==`, `!=`, `??`, a default value with `!`, and the `?size` and `?has_content` builtins. The template root is the CML model: `boundedContexts`, `map`, `domains`, and `userRequirements`.

`cm validate` and `cm generate` use the same flags as Context Mapper CLI 6.12 (`-i`, `-g`, `-o`, `-t`, `-f`).

## Library

```go
package main

import (
    "fmt"

    "github.com/remdev/go-cml/cml"
)

func main() {
    model, err := cml.ParseFile("model.cml")
    if err != nil {
        panic(err)
    }
    if cml.HasErrors(model.Validate()) {
        for _, d := range model.Validate() {
            fmt.Println(d)
        }
        return
    }
    fmt.Print(model.String())
}
```

| Package | Role |
| --- | --- |
| `cml` | Parse, validate, and print CML |
| `scl` | Parse Service Cutter SCL |
| `gen` | PlantUML, context map, MDSL, SCL, and text templates |
| `refactor` | Architectural refactorings |
| `cli` | The `cm` commands |

## Language evolution

CML 6.12 remains the compatibility baseline. Preparatory APIs preserve source
documents (`cml.LoadWorkspace`) and identify validation rules (`cml.Rules`);
they add no syntax or dialect. See the [compatibility contract](docs/compatibility.md)
and [architecture boundaries](docs/architecture/evolution.md) before extending the language.

## Development

```bash
go test ./...
./scripts/cover.sh
```

`./scripts/cover.sh` prints per-package coverage and a function report for `cml`, and writes `coverage.out`. Open it with `go tool cover -html=coverage.out`.

GitHub Actions runs `go test ./...` and [golangci-lint](https://golangci-lint.run/) v2.14 on every push and pull request. The linter configuration is `.golangci.yml`.

## License

Apache License 2.0. The CML language was designed by the Context Mapper project.
