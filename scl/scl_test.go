package scl

import "testing"

func TestParseGeneratedShape(t *testing.T) {
	src := `
UseCase BookCargo {
	reads "Cargo.origin"
	writes "Cargo.trackingId", "Cargo.destination"
}
Aggregate "Cargo" {
	"Cargo.trackingId", "Cargo.origin"
}
Compatibilities {
	AvailabilityCriticality {
		characteristic HIGH
		"Cargo.origin"
	}
}
`
	model, err := Parse("sample.scl", src)
	if err != nil {
		t.Fatal(err)
	}
	if len(model.UseCases) != 1 || len(model.UseCases[0].Reads) != 1 || len(model.UseCases[0].Writes) != 2 {
		t.Fatalf("%+v", model.UseCases[0])
	}
	if len(model.Aggregates) != 1 || model.Aggregates[0].Name != "Cargo" {
		t.Fatalf("%+v", model.Aggregates)
	}
	again, err := Parse("reprint.scl", model.String())
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Characteristics) != 1 || again.Characteristics[0].Name != "HIGH" {
		t.Fatalf("%+v", again.Characteristics)
	}
}
