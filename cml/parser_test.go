package cml

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseExamples(t *testing.T) {
	files := []string{
		"ddd-sample.cml",
		"insurance-team.cml",
		"claims-flow.cml",
	}
	for _, name := range files {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("..", "examples", name)
			model, err := ParseFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if len(model.BoundedContexts) == 0 {
				t.Fatal("no bounded contexts")
			}
		})
	}
}

func TestDDDSampleShape(t *testing.T) {
	model := mustParse(t, "ddd-sample.cml")
	if model.Map == nil || model.Map.Name != "DDDSampleMap" {
		t.Fatalf("map = %+v", model.Map)
	}
	if len(model.Map.Contains) != 3 || len(model.Map.Relationships) != 3 {
		t.Fatalf("contains=%v rels=%d", model.Map.Contains, len(model.Map.Relationships))
	}
	sk := model.Map.Relationships[0]
	if sk.Kind != RelSharedKernel || sk.Participant1 != "CargoBookingContext" || sk.Participant2 != "VoyagePlanningContext" {
		t.Fatalf("shared kernel = %+v", sk)
	}
	ud := model.Map.Relationships[1]
	if ud.Kind != RelUpstreamDownstream || ud.Upstream != "LocationContext" || ud.Downstream != "CargoBookingContext" {
		t.Fatalf("upstream = %+v", ud)
	}
	if !slicesHas(ud.UpstreamRoles, "OHS") || !slicesHas(ud.UpstreamRoles, "PL") {
		t.Fatalf("roles = %v", ud.UpstreamRoles)
	}
	cargo := model.FindBC("CargoBookingContext")
	if cargo == nil || len(cargo.Modules) != 2 || cargo.Modules[0].BasePackage != "se.citerus.dddsample.domain.model" {
		t.Fatalf("cargo module = %+v", cargo)
	}
	_, _, agg := model.FindAggregate("CargoItineraryLegDeliveryRouteSpecification")
	if agg == nil {
		t.Fatal("missing cargo aggregate")
	}
	var root *DomainObject
	for _, obj := range agg.Objects {
		if obj.Name == "Cargo" {
			root = obj
		}
	}
	if root == nil || !root.AggregateRoot || root.Repository == nil || len(root.Repository.Operations) != 4 {
		t.Fatalf("cargo entity = %+v", root)
	}
}

func TestInsuranceTeamMap(t *testing.T) {
	model := mustParse(t, "insurance-team.cml")
	if model.Map.Type != "ORGANIZATIONAL" || model.Map.State != "TO_BE" {
		t.Fatalf("map type/state = %s %s", model.Map.Type, model.Map.State)
	}
	cs := model.Map.Relationships[0]
	if cs.Kind != RelCustomerSupplier || cs.Upstream != "CustomerManagementContext" || cs.Downstream != "CustomerSelfServiceContext" {
		t.Fatalf("customer-supplier = %+v", cs)
	}
	team := model.FindBC("CustomersBackofficeTeam")
	if team == nil || team.Type != "TEAM" || len(team.Realizes) != 1 || len(team.Implements) != 1 {
		t.Fatalf("team = %+v", team)
	}
	if len(model.Domains) != 1 || len(model.Domains[0].Subdomains) != 4 {
		t.Fatalf("domains = %+v", model.Domains)
	}
	if model.Domains[0].Subdomains[0].Type != "CORE_DOMAIN" {
		t.Fatal(model.Domains[0].Subdomains[0].Type)
	}
}

func TestClaimsFlow(t *testing.T) {
	model := mustParse(t, "claims-flow.cml")
	bc := model.FindBC("ClaimsManagement")
	if bc == nil || bc.Application == nil || len(bc.Application.Commands) != 7 || len(bc.Application.Flows) != 1 {
		t.Fatalf("application = %+v", bc)
	}
	flow := bc.Application.Flows[0]
	if flow.Name != "ClaimsFlow" || len(flow.Steps) < 8 {
		t.Fatalf("flow steps = %d", len(flow.Steps))
	}
	var accept *FlowStep
	for _, step := range flow.Steps {
		if step.Command == "AcceptClaim" && step.Aggregate == "Claims" {
			accept = step
		}
	}
	if accept == nil || accept.Transition == nil || accept.Transition.From[0] != "OPEN" || accept.Transition.To[0].Value != "ACCEPTED" {
		t.Fatalf("accept step = %+v", accept)
	}
	_, _, agg := model.FindAggregate("Claims")
	if agg == nil {
		t.Fatal("missing claims aggregate")
	}
	var state *DomainObject
	var claim *DomainObject
	for _, obj := range agg.Objects {
		if obj.Name == "ClaimState" {
			state = obj
		}
		if obj.Name == "Claim" {
			claim = obj
		}
	}
	if state == nil || !state.AggregateLifecycle || len(state.EnumValues) != 4 {
		t.Fatalf("state enum = %+v", state)
	}
	if claim == nil || !claim.AggregateRoot {
		t.Fatal("claim root missing")
	}
	foundDesc := false
	for _, attr := range claim.Attributes {
		if attr.Name == "description" {
			foundDesc = true
		}
	}
	if !foundDesc {
		t.Fatal("escaped description attribute was not parsed")
	}
}

func TestRelationshipKeywords(t *testing.T) {
	src := `
ContextMap M {
	A Partnership B : Rel { implementationTechnology = "HTTP" }
	A Shared-Kernel C
	Up Upstream-Downstream Down
	Down2 [ACL] Downstream-Upstream [OHS, PL] Up2
	Customer Customer-Supplier Supplier
	Supplier2 [PL] Supplier-Customer [CF] Customer2
	Left [U, S, PL] -> [D, C] Right
}
BoundedContext A
BoundedContext B
BoundedContext C
BoundedContext Up
BoundedContext Down
BoundedContext Up2
BoundedContext Down2
BoundedContext Customer
BoundedContext Supplier
BoundedContext Customer2
BoundedContext Supplier2
BoundedContext Left
BoundedContext Right
`
	model, err := Parse("rel.cml", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	rels := model.Map.Relationships
	if rels[0].Kind != RelPartnership || rels[0].Name != "Rel" || rels[0].ImplementationTechnology != "HTTP" {
		t.Fatalf("partnership %+v", rels[0])
	}
	if rels[1].Kind != RelSharedKernel || rels[1].Participant2 != "C" {
		t.Fatalf("sk %+v", rels[1])
	}
	if rels[2].Kind != RelUpstreamDownstream || rels[2].Upstream != "Up" || rels[2].Downstream != "Down" {
		t.Fatalf("ud %+v", rels[2])
	}
	if rels[3].Upstream != "Up2" || rels[3].Downstream != "Down2" || !slicesHas(rels[3].UpstreamRoles, "OHS") || !slicesHas(rels[3].DownstreamRoles, "ACL") {
		t.Fatalf("down-up %+v", rels[3])
	}
	if rels[4].Kind != RelCustomerSupplier || rels[4].Downstream != "Customer" || rels[4].Upstream != "Supplier" {
		t.Fatalf("cs %+v", rels[4])
	}
	if rels[5].Kind != RelCustomerSupplier || rels[5].Upstream != "Supplier2" || rels[5].Downstream != "Customer2" {
		t.Fatalf("sc %+v", rels[5])
	}
	if rels[6].Kind != RelCustomerSupplier || !slicesHas(rels[6].UpstreamRoles, "PL") {
		t.Fatalf("bracket cs %+v", rels[6])
	}
}

func TestStoriesAndValues(t *testing.T) {
	src := `
BoundedContext Orders
UserStory CreateOrder {
	As a "Clerk" I want to create a "Order" with its "id", "total" so that "it is stored"
	and that "traceability" is promoted, accepting that "speed" is reduced
}
UseCase Book {
	actor = "Clerk"
	interactions = create a "Order", read the "Order" in an "Archive"
	benefit = "booked"
}
Stakeholders of Orders {
	Stakeholder Clerk { influence = HIGH interest = MEDIUM description = "back office" }
}
ValueRegister Ethics for Orders {
	ValueCluster FairnessCluster {
		core = FAIRNESS
		Value Fairness {
			isCore
			Stakeholder Clerk {
				priority = HIGH
				consequences
					good "trust" action "publish" ACT
			}
		}
	}
}
`
	model, err := Parse("story.cml", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	story := model.Requirements[0].(*UserStory)
	if story.Role != "Clerk" || story.Benefit != "it is stored" || len(story.Features) != 1 || story.Features[0].Verb != "create" {
		t.Fatalf("story %+v feature %+v", story, story.Features)
	}
	if story.Valuation == nil || story.Valuation.HarmWord != "reduced" {
		t.Fatalf("valuation %+v", story.Valuation)
	}
	uc := model.Requirements[1].(*UseCase)
	if len(uc.Features) != 2 || uc.Features[1].Container != "Archive" {
		t.Fatalf("usecase %+v", uc.Features)
	}
	if model.ValueRegisters[0].Clusters[0].Values[0].Elicitations[0].Consequences[0].Action.Type != "ACT" {
		t.Fatal("value elicitation")
	}
}

func mustParse(t *testing.T, name string) *Model {
	t.Helper()
	model, err := ParseFile(filepath.Join("..", "examples", name))
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func slicesHas(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func TestUseCaseStringVerbs(t *testing.T) {
	src := `
UseCase Get_paid_for_car_accident {
	actor "Claimant"
	interactions
		"submit" a "Claim" with its "date", "amountClaimed", "description" for a "Policy",
		"verifyExistanceOf" "Policy" with its "startDate", "endDate" for a "Contract",
		"assign" an "Agent" with its "personalID", "firstName", "lastName" for "Claim",
		"verify" "Policy" for a "Contract",
		"pay" "Claimant" with its "firstName", "lastName",
		"close" "Claim" for "Claimant"
	benefit "A claimant submits a claim and gets paid."
	scope "Insurance company"
	level "Summary"
}
`
	model, err := Parse("claims-use-case.cml", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	uc := model.Requirements[0].(*UseCase)
	if uc.Actor != "Claimant" || len(uc.Features) != 6 {
		t.Fatalf("actor=%q features=%d", uc.Actor, len(uc.Features))
	}
	pay := uc.Features[4]
	if pay.Verb != "pay" || len(pay.Attrs) != 2 || pay.Attrs[1] != "lastName" {
		t.Fatalf("pay feature = %+v", pay)
	}
	closeFeature := uc.Features[5]
	if closeFeature.Verb != "close" || closeFeature.Entity != "Claim" || closeFeature.Container != "Claimant" {
		t.Fatalf("close feature = %+v", closeFeature)
	}
}

func TestParseSourceBuffer(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("..", "examples", "ddd-sample.cml"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse("ddd-sample.cml", src); err != nil {
		t.Fatal(err)
	}
}
