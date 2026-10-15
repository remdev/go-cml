package cml

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestOfficialCorpus(t *testing.T) {
	root := filepath.Join("..", "testdata", "corpus")
	count := 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".cml") {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		count++
		t.Run(rel, func(t *testing.T) {
			model, err := ParseFile(path)
			if err != nil {
				t.Fatal(err)
			}
			printed := model.String()
			again, err := Parse(path, []byte(printed))
			if err != nil {
				t.Fatalf("reprint did not parse: %v", err)
			}
			if again.String() != printed {
				t.Fatal("reprint changed the model")
			}
			assertParsedInvariants(t, model)
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count != 55 {
		t.Fatalf("corpus size = %d, want 55", count)
	}
}

func TestCorpusRelationshipSemantics(t *testing.T) {
	model := parseCorpus(t, "ddd-sample/DDD-Sample_Alternative-Relationship-Syntax.cml")
	rels := model.Map.Relationships
	if rels[0].Kind != RelSharedKernel || rels[0].Participant1 != "CargoBookingContext" || rels[0].Participant2 != "VoyagePlanningContext" {
		t.Fatalf("shared kernel = %+v", rels[0])
	}
	assertUD(t, rels[1], "LocationContext", "CargoBookingContext", []string{"OHS", "PL"}, nil)
	assertUD(t, rels[2], "LocationContext", "VoyagePlanningContext", []string{"OHS", "PL"}, nil)

	team := parseCorpus(t, "insurance-example/Insurance-Example_Team-Map.cml")
	if team.Map.Type != "ORGANIZATIONAL" || team.Map.State != "TO_BE" {
		t.Fatalf("team map = %s %s", team.Map.Type, team.Map.State)
	}
	assertCS(t, team.Map.Relationships[0], "CustomerManagementContext", "CustomerSelfServiceContext", nil, nil)
	if team.FindBC("CustomersBackofficeTeam").Type != "TEAM" || team.FindBC("CustomersBackofficeTeam").Realizes[0] != "CustomerManagementContext" {
		t.Fatal("team realizes the wrong context")
	}
	if len(team.Domains[0].Subdomains) != 4 || team.Domains[0].Subdomains[3].Name != "DebtsDomain" {
		t.Fatalf("subdomains = %+v", team.Domains[0].Subdomains)
	}

	stage := parseCorpus(t, "ddd-sample/DDD-Sample-Stage-5.cml")
	_, _, cargoAgg := stage.FindAggregate("CargoItineraryLegDeliveryRouteSpecification")
	cargo := findObject(t, cargoAgg, "Cargo")
	if cargo.Repository == nil || cargo.Repository.Operations[0].Name != "find" || cargo.Repository.Operations[0].Return.String() != "@Cargo" {
		t.Fatalf("cargo repository = %+v", cargo.Repository)
	}
	if cargo.References[0].Name != "origin" || cargo.References[0].Type.RefName() != "LocationShared" {
		t.Fatalf("origin = %+v", cargo.References[0])
	}

	flow := parseCorpus(t, "workflow-samples/InsuranceQuoteRequest.cml")
	steps := flow.FindBC("InsuranceQuotes").Application.Flows[0].Steps
	if steps[0].Command != "SubmitRequest" || steps[0].Aggregate != "QuoteRequest" || len(steps[0].Transition.From) != 0 || steps[0].Transition.To[0].Value != "SUBMITTED" {
		t.Fatalf("submit = %+v", steps[0])
	}
	check := steps[2]
	if check.Operation != "checkRequest" || !check.Transition.Exclusive || check.EventAlt != "exclusive" || check.Events[1] != "RequestRejected" {
		t.Fatalf("check = %+v", check)
	}

	coord := parseCorpus(t, "service-coordination-example/InsuranceClaimCoordination.cml")
	stepsCoord := coord.FindBC("ClaimsManagement").Application.Coordinations[0].Steps
	if len(stepsCoord) != 4 || stepsCoord[1].BoundedContext != "InsuranceManagement" || stepsCoord[1].Operation != "checkInsurance" || stepsCoord[3].Service != "PaymentApplicationService" {
		t.Fatalf("coordination = %+v", stepsCoord)
	}
	if coord.Map.Relationships[0].Kind != RelSharedKernel {
		t.Fatalf("bare arrow = %+v", coord.Map.Relationships[0])
	}

	claims := parseCorpus(t, "ooad-sample-claims/claims-use-case.cml")
	var paid *UseCase
	for _, req := range claims.Requirements {
		if uc, ok := req.(*UseCase); ok && uc.Name == "Get_paid_for_car_accident" {
			paid = uc
		}
	}
	if paid == nil || paid.Actor != "Claimant" || len(paid.Features) != 6 || paid.Features[5].Verb != "close" || paid.Features[5].Entity != "Claim" || paid.Features[4].Attrs[1] != "lastName" {
		t.Fatalf("use case = %+v", paid)
	}

	values := parseCorpus(t, "same-day-delivery-shop/stakeholders-and-values.cml")
	group := values.StakeholderBlocks[0].Items[0].Group
	if group.Name != "Online_Shopping_Company" || len(group.Members) != 3 || group.Members[0].Influence != "MEDIUM" || group.Members[0].Interest != "HIGH" {
		t.Fatalf("stakeholder group = %+v", group)
	}
	cluster := values.ValueRegisters[0].Clusters[0]
	if cluster.Core != "AUTONOMY" || len(cluster.Demonstrators) != 2 {
		t.Fatalf("cluster = %+v", cluster)
	}
	if cluster.Values[0].Elicitations[0].Consequences[0].Type != "good" {
		t.Fatalf("consequence = %+v", cluster.Values[0].Elicitations[0].Consequences[0])
	}
	freedom := values.ValueRegisters[0].Values[0]
	if freedom.Name != "WorkLifeBalance" || !freedom.IsCore || freedom.Elicitations[0].Stakeholder != "Drivers" {
		t.Fatalf("value = %+v", freedom)
	}
}

func assertParsedInvariants(t *testing.T, model *Model) {
	t.Helper()
	names := map[string]bool{}
	for _, bc := range model.BoundedContexts {
		if bc.Name == "" {
			t.Fatal("bounded context without a name")
		}
		if names[bc.Name] {
			t.Fatalf("duplicate bounded context %s", bc.Name)
		}
		names[bc.Name] = true
		for _, agg := range bc.AllAggregates() {
			if agg.Name == "" {
				t.Fatal("aggregate without a name")
			}
			for _, obj := range agg.Objects {
				if obj.Name == "" || obj.Kind == "" {
					t.Fatalf("empty domain object in %s", agg.Name)
				}
			}
		}
		if model.FindBC(bc.Name) != bc {
			t.Fatalf("FindBC lost %s", bc.Name)
		}
	}
	if model.Map == nil {
		return
	}
	for _, name := range model.Map.Contains {
		if !names[name] {
			t.Fatalf("context map contains unknown bounded context %s", name)
		}
	}
	for _, rel := range model.Map.Relationships {
		left, right := rel.ends()
		if left == "" || right == "" {
			t.Fatalf("relationship without participants: %+v", rel)
		}
		if !names[left] || !names[right] {
			t.Fatalf("relationship %s -> %s references an undefined context", left, right)
		}
		switch rel.Kind {
		case RelPartnership, RelSharedKernel, RelUpstreamDownstream, RelCustomerSupplier:
		default:
			t.Fatalf("unknown relationship kind %q", rel.Kind)
		}
	}
}

func TestImports(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "base.cml")
	other := filepath.Join(dir, "other.cml")
	if err := os.WriteFile(other, []byte("BoundedContext Imported { type = TEAM }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(base, []byte("import \"other.cml\"\nBoundedContext Local\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	model, err := ParseFile(base)
	if err != nil {
		t.Fatal(err)
	}
	if model.FindBC("Local") == nil || model.FindBC("Imported") == nil || model.FindBC("Imported").Type != "TEAM" {
		t.Fatalf("imported model = %+v", model.BoundedContexts)
	}
	if len(model.Imports) != 1 || model.Imports[0].URI != "other.cml" {
		t.Fatalf("imports = %+v", model.Imports)
	}

	if err := os.WriteFile(other, []byte("import \"base.cml\"\nBoundedContext Imported\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseFile(base); err == nil || !strings.Contains(err.Error(), "cyclic import") {
		t.Fatalf("cyclic import error = %v", err)
	}

	if err := os.WriteFile(other, []byte("ContextMap ImportedMap {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(base, []byte("import \"other.cml\"\nContextMap LocalMap {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseFile(base); err == nil || !strings.Contains(err.Error(), "also defines a Context Map") {
		t.Fatalf("two maps error = %v", err)
	}
}

func TestServiceOperationDelegate(t *testing.T) {
	model := parseSrc(t, `
BoundedContext Orders {
	Service OrderService {
		void place() delegates to @Billing.charge;
		void local() => Warehouse.stock;
	}
}
`)
	ops := model.FindBC("Orders").Services[0].Operations
	if ops[0].Delegate == nil || !ops[0].Delegate.ExplicitAt || ops[0].Delegate.Target != "Billing" || ops[0].Delegate.Operation != "charge" {
		t.Fatalf("explicit delegate = %+v", ops[0].Delegate)
	}
	if ops[1].Delegate == nil || ops[1].Delegate.ExplicitAt || ops[1].Delegate.Target != "Warehouse" || ops[1].Delegate.Operation != "stock" {
		t.Fatalf("arrow delegate = %+v", ops[1].Delegate)
	}
}

func parseCorpus(t *testing.T, rel string) *Model {
	t.Helper()
	model, err := ParseFile(filepath.Join("..", "testdata", "corpus", rel))
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func TestCorpusQuoteFlowStates(t *testing.T) {
	model := parseCorpus(t, "lakeside-mutual/QuoteRequestFlow.cml")
	var flow *Flow
	for _, bc := range model.BoundedContexts {
		if bc.Application == nil {
			continue
		}
		for _, candidate := range bc.Application.Flows {
			if candidate.Name == "QuoteRequestFlow" {
				flow = candidate
			}
		}
	}
	if flow == nil {
		t.Fatal("QuoteFlow was not parsed")
	}
	accept := flow.Steps[len(flow.Steps)-1]
	if accept.Operation != "accept" || !slices.Equal(accept.Events, []string{"PolicyCreated", "QuoteExpired"}) || accept.EventAlt != "exclusive" {
		t.Fatalf("accept step = %+v", accept)
	}
}
