# go-cml

[English](README.md)

Go-реализация [Context Mapper](https://contextmapper.org/): язык CML и утилита `cm` в одном модуле `github.com/remdev/go-cml`.

Язык совместим с Context Mapper DSL 6.12: контекстные карты, ограниченные контексты, тактическая модель Sculptor, сценарии использования, пользовательские истории, стейкхолдеры, реестры ценностей, потоки и координации. Рядом живёт SCL — язык пользовательских представлений Service Cutter.

## Сборка

```bash
go install github.com/remdev/go-cml/cmd/cm@latest
```

Из локальной копии:

```bash
go build -o bin/cm ./cmd/cm
```

## Команды

Каталог для `generate` должен уже существовать.

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

Генератор `context-map` всегда пишет Graphviz `.gv`. PNG и SVG появляются, если в `PATH` есть `dot`.

Генератор `generic` принимает подмножество Freemarker: `${...}`, `<#if>`, `<#elseif>`, `<#else>`, `<#list ... as ...>`, `<#assign>`, комментарии и маркеры пробелов `<#lt>`, `<#rt>`, `<#t>`, `<#nt>`. В выражении доступны `==`, `!=`, `??`, значение по умолчанию через `!` и суффиксы `?size`, `?has_content`. Корень шаблона — модель CML: `boundedContexts`, `map`, `domains`, `userRequirements`.

Флаги `cm validate` и `cm generate` совпадают с Context Mapper CLI 6.12 (`-i`, `-g`, `-o`, `-t`, `-f`).

## Библиотека

```go
model, err := cml.ParseFile("model.cml")
if cml.HasErrors(model.Validate()) {
    // диагностика в model.Validate()
}
fmt.Print(model.String())
```

| Пакет | Назначение |
| --- | --- |
| `cml` | Разбор, проверка, печать CML |
| `scl` | Разбор Service Cutter SCL |
| `gen` | PlantUML, контекстная карта, MDSL, SCL, текстовые шаблоны |
| `refactor` | Архитектурные рефакторинги |
| `cli` | Команды `cm` |

## Разработка

```bash
go test ./...
./scripts/cover.sh
```

`./scripts/cover.sh` печатает покрытие пакетов и функций `cml` и записывает `coverage.out`. HTML-отчёт: `go tool cover -html=coverage.out`.

GitHub Actions на каждый push и pull request запускает `go test ./...` и [golangci-lint](https://golangci-lint.run/) v2.14. Настройки линтера — в `.golangci.yml`.

## Лицензия

Apache License 2.0. Язык CML спроектирован проектом Context Mapper.
