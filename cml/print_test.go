package cml

import (
	"path/filepath"
	"testing"
)

func TestPrintRoundTrip(t *testing.T) {
	for _, name := range []string{"ddd-sample.cml", "insurance-team.cml", "claims-flow.cml"} {
		t.Run(name, func(t *testing.T) {
			model, err := ParseFile(filepath.Join("..", "examples", name))
			if err != nil {
				t.Fatal(err)
			}
			first := model.String()
			again, err := Parse(name, []byte(first))
			if err != nil {
				t.Fatalf("reprint failed to parse: %v\n%s", err, first)
			}
			second := again.String()
			if first != second {
				t.Fatalf("print is not stable\n--- first ---\n%s\n--- second ---\n%s", first, second)
			}
			if HasErrors(again.Validate()) {
				for _, d := range again.Validate() {
					if d.Severity == SevError {
						t.Error(d)
					}
				}
			}
		})
	}
}
