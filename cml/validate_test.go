package cml

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestExamplesHaveNoErrors(t *testing.T) {
	for _, name := range []string{"ddd-sample.cml", "insurance-team.cml", "claims-flow.cml"} {
		model, err := ParseFile(filepath.Join("..", "examples", name))
		if err != nil {
			t.Fatal(err)
		}
		var errors []string
		for _, d := range model.Validate() {
			if d.Severity == SevError {
				errors = append(errors, d.String())
			}
		}
		if len(errors) > 0 {
			t.Fatalf("%s\n%s", name, strings.Join(errors, "\n"))
		}
	}
}

func TestSemanticRules(t *testing.T) {
	src := `
ContextMap Landscape {
	type = SYSTEM_LANDSCAPE
	contains TeamA, Orders
	TeamA [P] <-> [P] TeamA
	Orders [U, OHS] -> [D, CF] Missing
	Orders [P] <-> [P] Outside
}
BoundedContext Outside
BoundedContext TeamA { type = TEAM }
BoundedContext Orders realizes TeamA {
	type = FEATURE
	Aggregate Twice {
		owner = Orders
		Entity A { aggregateRoot }
		Entity B { aggregateRoot }
		enum S { aggregateLifecycle A, B }
		enum T { aggregateLifecycle A, B }
	}
}
Domain D {
	Subdomain D { type = CORE_DOMAIN }
}
`
	model, err := Parse("bad.cml", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	text := diagnostics(model.Validate())
	for _, want := range []string{
		"TEAM",
		"two different bounded contexts",
		"not a team",
		"multiple aggregate roots",
		"multiple enums",
		"same name as the domain",
		"Couldn't resolve reference to BoundedContext 'Missing'",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in:\n%s", want, text)
		}
	}
}

func diagnostics(ds []Diagnostic) string {
	var b strings.Builder
	for _, d := range ds {
		b.WriteString(d.Message)
		b.WriteByte('\n')
	}
	return b.String()
}
