package cml

import (
	"slices"
	"strings"
	"testing"
)

func parseSrc(t *testing.T, src string) *Model {
	t.Helper()
	model, err := Parse("test.cml", []byte(src))
	if err != nil {
		t.Fatalf("parse: %v\n%s", err, src)
	}
	return model
}

func TestRelationshipForms(t *testing.T) {
	src := `
ContextMap Map {
	type = SYSTEM_LANDSCAPE
	state = AS_IS
	contains A, B, C, Up, Down, Customer, Supplier, Left, Right
	A [U, PL, OHS] -> [D, ACL, CF] B : Sync {
		implementationTechnology = "HTTP"
		exposedAggregates = Order, Payment
		downstreamRights = VETO_RIGHT
	}
	A [D, CF] <- [U, OHS, PL] C
	[P] Left <-> [P] Right
	A Partnership B : Partners
	A Shared-Kernel C
	A <-> B
	Customer Customer-Supplier Supplier
	Supplier [PL] Supplier-Customer [CF] Customer
	Up [OHS] Upstream-Downstream [ACL] Down
	Down [ACL] Downstream-Upstream [OHS, PL] Up
	Left [U, S, PL] -> [D, C] Right
}
BoundedContext A
BoundedContext B
BoundedContext C
BoundedContext Up
BoundedContext Down
BoundedContext Customer
BoundedContext Supplier
BoundedContext Left
BoundedContext Right
`
	model := parseSrc(t, src)
	if model.Map.Name != "Map" || model.Map.Type != "SYSTEM_LANDSCAPE" || model.Map.State != "AS_IS" {
		t.Fatalf("map header = %+v", model.Map)
	}
	if !slices.Equal(model.Map.Contains, []string{"A", "B", "C", "Up", "Down", "Customer", "Supplier", "Left", "Right"}) {
		t.Fatalf("contains = %v", model.Map.Contains)
	}
	rels := model.Map.Relationships
	if len(rels) != 11 {
		t.Fatalf("relationships = %d", len(rels))
	}
	assertUD(t, rels[0], "A", "B", []string{"PL", "OHS"}, []string{"ACL", "CF"})
	if rels[0].Name != "Sync" || rels[0].ImplementationTechnology != "HTTP" || rels[0].DownstreamRights != "VETO_RIGHT" {
		t.Fatalf("relationship body = %+v", rels[0])
	}
	if !slices.Equal(rels[0].ExposedAggregates, []string{"Order", "Payment"}) {
		t.Fatalf("exposed = %v", rels[0].ExposedAggregates)
	}
	assertUD(t, rels[1], "C", "A", []string{"OHS", "PL"}, []string{"CF"})
	if rels[2].Kind != RelPartnership || rels[2].Participant1 != "Left" || rels[2].Participant2 != "Right" {
		t.Fatalf("bracket partnership = %+v", rels[2])
	}
	if rels[3].Kind != RelPartnership || rels[3].Name != "Partners" || rels[3].Participant1 != "A" || rels[3].Participant2 != "B" {
		t.Fatalf("keyword partnership = %+v", rels[3])
	}
	if rels[4].Kind != RelSharedKernel || rels[4].Participant2 != "C" {
		t.Fatalf("shared kernel keyword = %+v", rels[4])
	}
	if rels[5].Kind != RelSharedKernel || rels[5].Participant1 != "A" || rels[5].Participant2 != "B" {
		t.Fatalf("bare shared kernel = %+v", rels[5])
	}
	if rels[6].Kind != RelCustomerSupplier || rels[6].Downstream != "Customer" || rels[6].Upstream != "Supplier" {
		t.Fatalf("customer-supplier = %+v", rels[6])
	}
	assertCS(t, rels[7], "Supplier", "Customer", []string{"PL"}, []string{"CF"})
	assertUD(t, rels[8], "Up", "Down", []string{"OHS"}, []string{"ACL"})
	assertUD(t, rels[9], "Up", "Down", []string{"OHS", "PL"}, []string{"ACL"})
	assertCS(t, rels[10], "Left", "Right", []string{"PL"}, nil)
}

func TestRelationshipCount(t *testing.T) {
	model := parseSrc(t, `
ContextMap M {
	A [U, S, PL] -> [D, C] B
}
BoundedContext A
BoundedContext B
`)
	rel := model.Map.Relationships[0]
	assertCS(t, rel, "A", "B", []string{"PL"}, nil)
}

func TestBoundedContextHeader(t *testing.T) {
	model := parseSrc(t, `
BoundedContext Orders implements Shop, Checkout realizes Web, Mobile refines Legacy {
	domainVisionStatement = "sell goods"
	type = APPLICATION
	responsibilities = "orders", "payment"
	implementationTechnology = "Go"
	knowledgeLevel = CONCRETE
	businessModel = "saas"
	evolution = PRODUCT
}
BoundedContext Bare
`)
	orders := model.FindBC("Orders")
	if orders == nil {
		t.Fatal("missing Orders")
	}
	if !slices.Equal(orders.Implements, []string{"Shop", "Checkout"}) || !slices.Equal(orders.Realizes, []string{"Web", "Mobile"}) || orders.Refines != "Legacy" {
		t.Fatalf("header = %+v", orders)
	}
	if orders.DomainVisionStatement != "sell goods" || orders.Type != "APPLICATION" || orders.KnowledgeLevel != "CONCRETE" || orders.Evolution != "PRODUCT" {
		t.Fatalf("attributes = %+v", orders)
	}
	if !slices.Equal(orders.Responsibilities, []string{"orders", "payment"}) || orders.ImplementationTechnology != "Go" || orders.BusinessModel != "saas" {
		t.Fatalf("strings = %+v", orders)
	}
	if bare := model.FindBC("Bare"); bare == nil || bare.hasBody() {
		t.Fatalf("bare context = %+v", bare)
	}
}

func TestTacticModel(t *testing.T) {
	model := parseSrc(t, `
BoundedContext Orders {
	Module sales {
		external
		basePackage = com.example.sales
		hint = "core"
		Aggregate OrderAgg {
			responsibilities = "take", "ship"
			useCases = PlaceOrder
			owner = TeamA
			knowledgeLevel = CONCRETE
			likelihoodForChange = OFTEN
			contentVolatility = RARELY
			availabilityCriticality = HIGH
			consistencyCriticality = NORMAL
			storageSimilarity = HUGE
			securityCriticality = LOW
			securityZone = "pci"
			securityAccessGroup = "orders"
			Entity Order {
				aggregateRoot
				not optimisticLocking
				package = com.example.order
				databaseTable = "orders"
				inheritanceType = JOINED
				discriminatorType = STRING
				validate = "ok"
				belongsTo @Customer
				public String id key required
				package int version not changeable
				- List<Line> lines cascade = "all" opposite order;
				-- "contains" @Line;
				def abstract public void touch() : write [NEW -> OPEN*];
				* String label();
				Repository OrderRepository {
					@Order find(String id) throws NotFound : read-only;
					void store(@Order order) delegates to AccessObject;
					List<@Order> search(String q) query = "q" cache gap construct build map;
					Map<String, @Order> index();
				}
			}
			ValueObject Line {
				immutable
				String sku key
			}
			enum Status {
				aggregateLifecycle
				ordinal
				String code key;
				NEW("new", 1), OPEN, DONE;
			}
			DomainEvent Placed { persistent String orderId }
			abstract Command Place { String orderId }
			Trait Audited { String by }
			BasicType Money { not immutable String amount }
			DataTransferObject OrderView { String id }
			Service OrderService {
				gap
				webservice
				hint = "api"
				subscribe to orders/placed eventBus = bus
				> @OrderRepository
				inject Billing
				void place(String id) throws Fail publish @Placed to orders:topic eventBus = bus : write [NEW -> OPEN];
				association "uses" @Line;
			}
			Resource OrderResource {
				scaffold
				path = "/orders"
				String list() GET path = "/" return = "ok";
			}
			Consumer OrderConsumer {
				hint = "in"
				> @OrderService
				unmarshall to @Placed
				queueName = orders/in
				subscribe to orders/in eventBus = bus
			}
		}
	}
}
`)
	bc := model.FindBC("Orders")
	mod := bc.Modules[0]
	if !mod.External || mod.BasePackage != "com.example.sales" || mod.Hint != "core" || mod.Name != "sales" {
		t.Fatalf("module = %+v", mod)
	}
	agg := mod.Aggregates[0]
	if agg.Owner != "TeamA" || agg.LikelihoodForChange != "OFTEN" || agg.ContentVolatility != "RARELY" || agg.SecurityZone != "pci" {
		t.Fatalf("aggregate attrs = %+v", agg)
	}
	if !slices.Equal(agg.Responsibilities, []string{"take", "ship"}) || !slices.Equal(agg.RequirementRefs, []string{"PlaceOrder"}) || agg.RequirementKind != "useCases" {
		t.Fatalf("aggregate refs = %+v kind %s", agg.RequirementRefs, agg.RequirementKind)
	}
	order := findObject(t, agg, "Order")
	if !order.AggregateRoot || order.OptimisticLocking == nil || *order.OptimisticLocking || order.Package != "com.example.order" {
		t.Fatalf("order flags = %+v lock %v", order, order.OptimisticLocking)
	}
	if order.DatabaseTable != "orders" || order.InheritanceType != "JOINED" || order.DiscriminatorType != "STRING" || order.BelongsTo != "Customer" || !order.BelongsToAt {
		t.Fatalf("order storage = %+v", order)
	}
	if order.Attributes[0].Name != "id" || order.Attributes[0].Visibility != "public" || !hasMod(order.Attributes[0].Mods, "key") || !hasMod(order.Attributes[0].Mods, "required") {
		t.Fatalf("id attr = %+v", order.Attributes[0])
	}
	if order.Attributes[1].Name != "version" || order.Attributes[1].Visibility != "package" || !hasNot(order.Attributes[1].Mods, "changeable") {
		t.Fatalf("version attr = %+v", order.Attributes[1])
	}
	if order.References[0].Name != "lines" || order.References[0].Type.String() != "List<Line>" || order.References[0].Opposite != "order" {
		t.Fatalf("reference = %+v type %s", order.References[0], order.References[0].Type)
	}
	if order.Associations[0].Description != "contains" || order.Associations[0].Target != "Line" || !order.Associations[0].ExplicitAt {
		t.Fatalf("association = %+v", order.Associations[0])
	}
	if !order.Operations[0].Abstract || !order.Operations[0].Write || order.Operations[0].Visibility != "public" || !order.Operations[0].Transition.To[0].End {
		t.Fatalf("touch = %+v", order.Operations[0])
	}
	if order.Operations[1].Name != "label" || order.Operations[1].Return.String() != "String" {
		t.Fatalf("label = %+v", order.Operations[1])
	}
	repo := order.Repository
	if repo.Name != "OrderRepository" || len(repo.Operations) != 4 {
		t.Fatalf("repository = %+v", repo)
	}
	if repo.Operations[0].Return.String() != "@Order" || repo.Operations[0].Params[0].Name != "id" || !repo.Operations[0].ReadOnly || repo.Operations[0].Throws[0] != "NotFound" {
		t.Fatalf("find = %+v", repo.Operations[0])
	}
	if repo.Operations[1].RepoDelegate != "AccessObject" || repo.Operations[1].Params[0].Type.String() != "@Order" {
		t.Fatalf("store = %+v", repo.Operations[1])
	}
	if repo.Operations[2].Return.String() != "List<@Order>" || repo.Operations[2].Query != "q" || !repo.Operations[2].Cache || repo.Operations[2].Gap == nil || !*repo.Operations[2].Gap || !repo.Operations[2].MapFlag {
		t.Fatalf("search = %+v", repo.Operations[2])
	}
	if repo.Operations[3].Return.String() != "Map<String, @Order>" {
		t.Fatalf("index = %s", repo.Operations[3].Return)
	}
	line := findObject(t, agg, "Line")
	if line.Immutable == nil || !*line.Immutable || line.Kind != KindValue {
		t.Fatalf("line = %+v", line)
	}
	status := findObject(t, agg, "Status")
	if !status.AggregateLifecycle || !status.Ordinal || status.EnumAttributes[0].Name != "code" || !status.EnumAttributes[0].Key {
		t.Fatalf("enum attr = %+v", status)
	}
	if status.EnumValues[0].Name != "NEW" || status.EnumValues[0].Params[0].String != "new" || !status.EnumValues[0].Params[1].IsInt || status.EnumValues[0].Params[1].Int != "1" {
		t.Fatalf("enum value = %+v", status.EnumValues[0])
	}
	if status.EnumValues[2].Name != "DONE" {
		t.Fatalf("values = %+v", status.EnumValues)
	}
	if findObject(t, agg, "Placed").Persistent == nil || !*findObject(t, agg, "Placed").Persistent {
		t.Fatal("event persistent")
	}
	if !findObject(t, agg, "Place").Abstract || findObject(t, agg, "Place").Kind != KindCommand {
		t.Fatal("command")
	}
	if findObject(t, agg, "Money").Immutable == nil || *findObject(t, agg, "Money").Immutable {
		t.Fatal("basic type should be explicitly mutable")
	}
	svc := agg.Services[0]
	if svc.Gap == nil || !*svc.Gap || !svc.WebService || svc.Hint != "api" || svc.Subscribe.Topic != "orders/placed" || svc.Subscribe.EventBus != "bus" {
		t.Fatalf("service = %+v sub %+v", svc, svc.Subscribe)
	}
	if !svc.Dependencies[0].ExplicitAt || svc.Dependencies[0].Ref != "OrderRepository" || !svc.Dependencies[1].Inject || svc.Dependencies[1].Ref != "Billing" {
		t.Fatalf("deps = %+v", svc.Dependencies)
	}
	place := svc.Operations[0]
	if place.Publish == nil || place.Publish.Event != "Placed" || !place.Publish.ExplicitAt || place.Publish.Topic != "orders:topic" || place.Publish.EventBus != "bus" || !place.Write {
		t.Fatalf("publish = %+v", place.Publish)
	}
	if svc.Associations[0].Description != "uses" || svc.Associations[0].Target != "Line" {
		t.Fatalf("service association = %+v", svc.Associations[0])
	}
	res := agg.Resources[0]
	if !res.Scaffold || res.Path != "/orders" || res.Operations[0].HTTPMethod != "GET" || res.Operations[0].Path != "/" || res.Operations[0].ReturnString != "ok" {
		t.Fatalf("resource = %+v op %+v", res, res.Operations[0])
	}
	cons := agg.Consumers[0]
	if cons.Unmarshall != "Placed" || !cons.UnmarshallAt || cons.ChannelKind != "queueName" || cons.Channel != "orders/in" || cons.Subscribe.Topic != "orders/in" {
		t.Fatalf("consumer = %+v", cons)
	}
	first := model.String()
	again := parseSrc(t, first)
	if again.String() != first {
		t.Fatal("tactic reprint is not stable")
	}
}

func TestApplicationFlowAndCoordination(t *testing.T) {
	model := parseSrc(t, `
BoundedContext Orders {
	Application Checkout {
		Command Submit
		DomainEvent Submitted
		DomainEvent Paid
		DomainEvent Done
		Service CheckoutService {
			void place();
			void refund();
		}
		Flow Purchase {
			command Submit [initiated by "clerk"] delegates to Orders [NEW, HOLD -> OPEN X FAILED*] emits event Submitted + Paid
			event Submitted + Paid triggers operation place O refund
			operation place delegates to Orders [-> DONE] emits event Done X Cancelled
		}
		Coordination Pay {
			Orders :: CheckoutService :: place;
			Billing :: BillingService :: charge;
		}
	}
}
`)
	app := model.FindBC("Orders").Application
	if app.Name != "Checkout" || len(app.Commands) != 1 || app.Commands[0].Kind != KindCommand || len(app.Events) != 3 {
		t.Fatalf("application = %+v", app)
	}
	flow := app.Flows[0]
	if flow.Name != "Purchase" || len(flow.Steps) != 3 {
		t.Fatalf("steps = %d", len(flow.Steps))
	}
	emit := flow.Steps[0]
	if emit.Command != "Submit" || emit.Actor != "clerk" || emit.Aggregate != "Orders" || emit.EventAlt != "concurrent" {
		t.Fatalf("emit = %+v", emit)
	}
	if !slices.Equal(emit.Transition.From, []string{"NEW", "HOLD"}) || !emit.Transition.Exclusive || emit.Transition.To[1].Value != "FAILED" || !emit.Transition.To[1].End {
		t.Fatalf("transition = %+v", emit.Transition)
	}
	if !slices.Equal(emit.Events, []string{"Submitted", "Paid"}) {
		t.Fatalf("events = %v", emit.Events)
	}
	trigger := flow.Steps[1]
	if trigger.Kind != "trigger" || trigger.TriggerAlt != "inclusive" || !slices.Equal(trigger.Operations, []string{"place", "refund"}) {
		t.Fatalf("trigger = %+v", trigger)
	}
	alt := flow.Steps[2]
	if alt.Operation != "place" || len(alt.Transition.From) != 0 || alt.Transition.To[0].Value != "DONE" || alt.EventAlt != "exclusive" {
		t.Fatalf("exclusive emit = %+v trans %+v", alt, alt.Transition)
	}
	if len(app.Coordinations) != 1 || app.Coordinations[0].Steps[1].BoundedContext != "Billing" || app.Coordinations[0].Steps[1].Service != "BillingService" || app.Coordinations[0].Steps[1].Operation != "charge" {
		t.Fatalf("coordination = %+v", app.Coordinations)
	}
}

func TestRequirementsStakeholdersAndValues(t *testing.T) {
	model := parseSrc(t, `
BoundedContext Orders
UseCase PlaceOrder {
	actor = "Clerk"
	secondaryActors = "Auditor"
	interactions = "submit" a "Order" with its "id", "total" for a "Customer", read the "Order" in an "Archive"
	benefit = "placed"
	scope = "shop"
	level = "goal"
}
UserStory SplitMe split by PlaceOrder {
	As an "Auditor" I want to "log" a "Order" with its "id" so that "it is traced"
	and that "traceability", "audit" are promoted, accepting that "speed" is reduced
}
Stakeholders of Orders, Billing {
	StakeholderGroup Buyers {
		Stakeholder Clerk { influence = HIGH interest = LOW description = "buyer" }
	}
	Stakeholder Guest
}
ValueRegister Ethics for Orders {
	ValueCluster Fair {
		core = FAIRNESS
		demonstrator = "equal treatment"
		Value Honesty { isCore }
	}
	ValueEpic SpeedEpic {
		As a Clerk I value "speed" as demonstrated in
		reduction of "waiting"
		realization of "fast checkout"
	}
	ValueNarrative CheckoutStory {
		When the SOI executes "checkout",
		stakeholders expect it to promote, protect or create "trust",
		possibly degrading or prohibiting "privacy"
		with the following externally observable and/or internally auditable behavior: "receipt"
	}
	ValueWeighting SpeedOverCost {
		In the context of the SOI,
		stakeholder Clerk values "speed" more than "cost"
		expecting benefits such as "revenue"
		running the risk of harms such as "burnout"
	}
}
Domain Shop {
	domainVisionStatement = "sell"
	Subdomain Checkout supports PlaceOrder {
		type = CORE_DOMAIN
		domainVisionStatement = "pay"
		Entity Cart { String id }
		Service CartService { void add(); }
	}
}
`)
	uc := model.Requirements[0].(*UseCase)
	if uc.Actor != "Clerk" || uc.SecondaryActors[0] != "Auditor" || uc.Benefit != "placed" || uc.Scope != "shop" || uc.Level != "goal" {
		t.Fatalf("usecase = %+v", uc)
	}
	if len(uc.Features) != 2 || uc.Features[0].Verb != "submit" || uc.Features[0].Article != "a" || uc.Features[0].Entity != "Order" || uc.Features[0].Container != "Customer" {
		t.Fatalf("feature0 = %+v", uc.Features[0])
	}
	if !slices.Equal(uc.Features[0].Attrs, []string{"id", "total"}) || uc.Features[1].Verb != "read" || uc.Features[1].ContainerPrep != "in" {
		t.Fatalf("features = %+v / %+v", uc.Features[0], uc.Features[1])
	}
	story := model.Requirements[1].(*UserStory)
	if story.SplitBy != "PlaceOrder" || story.Role != "Auditor" || story.Features[0].Verb != "log" || !story.Features[0].WantTo || story.Benefit != "it is traced" {
		t.Fatalf("story = %+v feature %+v", story, story.Features[0])
	}
	if story.Valuation == nil || !story.Valuation.PromotedPlural || story.Valuation.HarmWord != "reduced" || story.Valuation.Promoted[1] != "audit" {
		t.Fatalf("valuation = %+v", story.Valuation)
	}
	block := model.StakeholderBlocks[0]
	if !slices.Equal(block.Contexts, []string{"Orders", "Billing"}) || block.Items[0].Group.Members[0].Influence != "HIGH" || block.Items[0].Group.Members[0].Description != "buyer" {
		t.Fatalf("stakeholders = %+v", block)
	}
	if block.Items[1].Person.Name != "Guest" {
		t.Fatalf("guest = %+v", block.Items[1])
	}
	reg := model.ValueRegisters[0]
	if reg.Context != "Orders" || reg.Clusters[0].Core != "FAIRNESS" || reg.Clusters[0].Demonstrators[0] != "equal treatment" || !reg.Clusters[0].Values[0].IsCore {
		t.Fatalf("cluster = %+v", reg.Clusters[0])
	}
	epic := reg.Epics[0]
	if epic.Stakeholder != "Clerk" || epic.Value != "speed" || epic.ReducedValues[0] != "waiting" || epic.RealizedValues[0] != "fast checkout" {
		t.Fatalf("epic = %+v", epic)
	}
	if reg.Narratives[0].Feature != "checkout" || reg.Narratives[0].PromotedValues != "trust" || reg.Narratives[0].PreAndPostConditions != "receipt" {
		t.Fatalf("narrative = %+v", reg.Narratives[0])
	}
	if reg.Weightings[0].Value1 != "speed" || reg.Weightings[0].Value2 != "cost" || reg.Weightings[0].Harms != "burnout" {
		t.Fatalf("weighting = %+v", reg.Weightings[0])
	}
	_, sub := model.FindSubdomain("Checkout")
	if sub == nil || sub.Type != "CORE_DOMAIN" || sub.Supports[0] != "PlaceOrder" || sub.Entities[0].Attributes[0].Name != "id" || sub.Services[0].Operations[0].Name != "add" {
		t.Fatalf("subdomain = %+v", sub)
	}
	data := model.TemplateData()
	bcs, _ := data["boundedContexts"].([]any)
	if len(bcs) != 1 || bcs[0].(map[string]any)["name"] != "Orders" {
		t.Fatalf("template data = %+v", data["boundedContexts"])
	}
	printed := model.String()
	if parseSrc(t, printed).String() != printed {
		t.Fatal("requirements reprint is not stable")
	}
}

func TestSyntaxRejectsMalformedCML(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{name: "second map", src: "ContextMap A {}\nContextMap B {}\n", want: "only one Context Map"},
		{name: "bad type", src: "BoundedContext A { type = NOPE }", want: "expected one of"},
		{name: "unclosed", src: "BoundedContext A {", want: "expected"},
		{name: "bad verb", src: "UseCase U { interactions = 1 }", want: "expected a verb"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse("bad.cml", []byte(tc.src))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func assertUD(t *testing.T, rel *Relationship, up, down string, upRoles, downRoles []string) {
	t.Helper()
	if rel.Kind != RelUpstreamDownstream || rel.Upstream != up || rel.Downstream != down {
		t.Fatalf("upstream/downstream = %+v, want %s -> %s", rel, up, down)
	}
	if !slices.Equal(rel.UpstreamRoles, upRoles) || !slices.Equal(rel.DownstreamRoles, downRoles) {
		t.Fatalf("roles up=%v down=%v", rel.UpstreamRoles, rel.DownstreamRoles)
	}
}

func assertCS(t *testing.T, rel *Relationship, up, down string, upRoles, downRoles []string) {
	t.Helper()
	if rel.Kind != RelCustomerSupplier || rel.Upstream != up || rel.Downstream != down {
		t.Fatalf("customer-supplier = %+v, want %s -> %s", rel, up, down)
	}
	if upRoles == nil {
		upRoles = []string{}
	}
	if downRoles == nil {
		downRoles = []string{}
	}
	gotUp, gotDown := rel.UpstreamRoles, rel.DownstreamRoles
	if gotUp == nil {
		gotUp = []string{}
	}
	if gotDown == nil {
		gotDown = []string{}
	}
	if !slices.Equal(gotUp, upRoles) || !slices.Equal(gotDown, downRoles) {
		t.Fatalf("roles up=%v down=%v", rel.UpstreamRoles, rel.DownstreamRoles)
	}
}

func findObject(t *testing.T, agg *Aggregate, name string) *DomainObject {
	t.Helper()
	for _, obj := range agg.Objects {
		if obj.Name == name {
			return obj
		}
	}
	t.Fatalf("object %s not found", name)
	return nil
}
