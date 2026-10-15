package gen

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/remdev/go-cml/cml"
)

func TestGenerators(t *testing.T) {
	model := parseExample(t, "ddd-sample.cml")
	files, err := PlantUML(model, "ddd-sample")
	if err != nil {
		t.Fatal(err)
	}
	joined := joinFiles(files)
	if !strings.Contains(joined, "@startuml") || !strings.Contains(joined, "CargoBookingContext") {
		t.Fatal("plantuml diagram is incomplete")
	}
	dot, err := ContextMapDOT(model)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(dot, "CargoBookingContext") || !strings.Contains(dot, "SK") {
		t.Fatal(dot)
	}
	mdsl, err := MDSL(model, "ddd-sample")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(joinFiles(mdsl), "endpoint type CargoRepository") {
		t.Fatal(joinFiles(mdsl))
	}
	claims := parseExample(t, "claims-flow.cml")
	states, err := PlantUML(claims, "claims")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(joinFiles(states), "OPEN --> ACCEPTED") {
		t.Fatal("flow state diagram missing")
	}
	text, err := Render("<#list boundedContexts as bc>${bc.name}\n</#list>", model.TemplateData())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "LocationContext") {
		t.Fatal(text)
	}
}

func TestServiceCutterRoundTrip(t *testing.T) {
	model := parseExample(t, "insurance-team.cml")
	body, err := ServiceCutter(model)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "Aggregate \"Customers\"") || !strings.Contains(body, "Customer.firstname") {
		t.Fatal(body)
	}
}

func parseExample(t *testing.T, name string) *cml.Model {
	t.Helper()
	model, err := cml.ParseFile(filepath.Join("..", "examples", name))
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func joinFiles(files []File) string {
	var b strings.Builder
	for _, file := range files {
		b.WriteString(file.Body)
	}
	return b.String()
}
