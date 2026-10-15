package refactor

import (
	"path/filepath"
	"testing"

	"github.com/remdev/go-cml/cml"
)

func TestExtractSharedKernel(t *testing.T) {
	model := parseExample(t, "ddd-sample.cml")
	if err := Apply(model, "extract-shared-kernel", []string{"CargoBookingContext", "VoyagePlanningContext", "SharedVoyage"}); err != nil {
		t.Fatal(err)
	}
	if model.FindBC("SharedVoyage") == nil {
		t.Fatal("shared context was not created")
	}
	for _, rel := range model.Map.Relationships {
		if rel.Kind == cml.RelSharedKernel {
			t.Fatal("shared kernel relationship remained")
		}
	}
	if cml.HasErrors(model.Validate()) {
		t.Fatal(model.Validate())
	}
}

func TestPartnershipToUpstream(t *testing.T) {
	model := parseExample(t, "insurance-team.cml")
	if err := Apply(model, "partnership-to-upstream-downstream", []string{"PolicyManagementContext", "RiskManagementContext"}); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, rel := range model.Map.Relationships {
		if rel.Upstream == "PolicyManagementContext" && rel.Downstream == "RiskManagementContext" && rel.Kind == cml.RelUpstreamDownstream {
			found = true
		}
	}
	if !found {
		t.Fatal("partnership was not converted")
	}
}

func TestDeriveBoundedContexts(t *testing.T) {
	model := parseExample(t, "insurance-team.cml")
	before := len(model.BoundedContexts)
	if err := Apply(model, "derive-bounded-contexts-from-subdomains", nil); err != nil {
		t.Fatal(err)
	}
	if len(model.BoundedContexts) <= before {
		t.Fatal("expected new contexts for unimplemented subdomains")
	}
	if model.FindBC("DebtsDomainContext") == nil {
		t.Fatal("DebtsDomain was not derived")
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
