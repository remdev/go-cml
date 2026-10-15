// Package cml implements the Context Mapper Language (CML): the strategic and
// tactic DDD syntax of Context Mapper, semantic checks, and a printer.
package cml

import "slices"

// Position is a 1-based source location.
type Position struct {
	File string
	Line int
	Col  int
}

// Model is a CML context-mapping model.
type Model struct {
	File              string
	Comment           string
	Imports           []Import
	Map               *ContextMap
	BoundedContexts   []*BoundedContext
	Domains           []*Domain
	Requirements      []Requirement
	StakeholderBlocks []*StakeholderBlock
	ValueRegisters    []*ValueRegister
}

// Import is an `import "..."` declaration.
type Import struct {
	Pos Position
	URI string
}

// ContextMap is the strategic context map.
type ContextMap struct {
	Pos           Position
	Comment       string
	Name          string
	Type          string
	State         string
	Contains      []string
	Relationships []*Relationship
}

// Relationship kinds.
const (
	RelPartnership        = "Partnership"
	RelSharedKernel       = "SharedKernel"
	RelUpstreamDownstream = "UpstreamDownstream"
	RelCustomerSupplier   = "CustomerSupplier"
)

// Relationship is a symmetric or upstream/downstream relationship.
type Relationship struct {
	Pos                      Position
	Comment                  string
	Kind                     string
	Participant1             string
	Participant2             string
	Upstream                 string
	Downstream               string
	UpstreamRoles            []string
	DownstreamRoles          []string
	Name                     string
	ImplementationTechnology string
	ExposedAggregates        []string
	DownstreamRights         string
}

// BoundedContext is a strategic bounded context or a team.
type BoundedContext struct {
	Pos                      Position
	Comment                  string
	Name                     string
	Implements               []string
	Realizes                 []string
	Refines                  string
	DomainVisionStatement    string
	Type                     string
	Responsibilities         []string
	ImplementationTechnology string
	KnowledgeLevel           string
	BusinessModel            string
	Evolution                string
	Application              *Application
	Modules                  []*Module
	Aggregates               []*Aggregate
	Services                 []*Service
}

// Application is the application layer of a bounded context.
type Application struct {
	Pos           Position
	Name          string
	Commands      []*DomainObject
	Events        []*DomainObject
	Services      []*Service
	Flows         []*Flow
	Coordinations []*Coordination
}

// Flow is an application or domain workflow.
type Flow struct {
	Pos   Position
	Name  string
	Steps []*FlowStep
}

// FlowStep is either an emit step or a trigger step.
type FlowStep struct {
	Pos        Position
	Kind       string // emit | trigger
	Actor      string
	Command    string
	Operation  string
	Aggregate  string
	Transition *StateTransition
	EventAlt   string
	Events     []string
	TriggerAlt string
	Commands   []string
	Operations []string
}

// StateTransition is a lifecycle transition inside brackets.
type StateTransition struct {
	From      []string
	To        []StateTarget
	Exclusive bool
}

// StateTarget is one target state, optionally terminal.
type StateTarget struct {
	Value string
	End   bool
}

// Coordination is a cross-context service coordination.
type Coordination struct {
	Pos   Position
	Name  string
	Steps []CoordinationStep
}

// CoordinationStep points at one application service operation.
type CoordinationStep struct {
	Pos            Position
	BoundedContext string
	Service        string
	Operation      string
}

// Module is a Sculptor module inside a bounded context.
type Module struct {
	Pos         Position
	Doc         string
	Name        string
	External    bool
	BasePackage string
	Hint        string
	Services    []*Service
	Resources   []*Resource
	Consumers   []*Consumer
	Aggregates  []*Aggregate
	Objects     []*DomainObject
}

// Aggregate is a tactic aggregate.
type Aggregate struct {
	Pos                     Position
	Comment                 string
	Doc                     string
	Name                    string
	Responsibilities        []string
	RequirementRefs         []string
	RequirementKind         string
	Owner                   string
	KnowledgeLevel          string
	LikelihoodForChange     string
	ContentVolatility       string
	AvailabilityCriticality string
	ConsistencyCriticality  string
	StorageSimilarity       string
	SecurityCriticality     string
	SecurityZone            string
	SecurityAccessGroup     string
	Services                []*Service
	Resources               []*Resource
	Consumers               []*Consumer
	Objects                 []*DomainObject
}

// Domain is a business domain.
type Domain struct {
	Pos                   Position
	Name                  string
	DomainVisionStatement string
	Subdomains            []*Subdomain
}

// Subdomain is a subdomain of a domain.
type Subdomain struct {
	Pos                   Position
	Name                  string
	Type                  string
	DomainVisionStatement string
	Supports              []string
	Entities              []*DomainObject
	Services              []*Service
}

// Domain object kinds.
const (
	KindEntity  = "Entity"
	KindValue   = "ValueObject"
	KindEvent   = "DomainEvent"
	KindCommand = "Command"
	KindTrait   = "Trait"
	KindDTO     = "DataTransferObject"
	KindBasic   = "BasicType"
	KindEnum    = "enum"
)

// DomainObject is an entity, value object, event, command, trait, DTO, basic type, or enum.
type DomainObject struct {
	Pos                 Position
	Comment             string
	Doc                 string
	Kind                string
	Name                string
	Abstract            bool
	Extends             string
	ExtendsAt           bool
	ExtendsJava         bool
	Traits              []TraitRef
	Package             string
	Hint                string
	OptimisticLocking   *bool
	Auditable           *bool
	Cache               *bool
	Gap                 *bool
	Scaffold            bool
	DatabaseTable       string
	DiscriminatorValue  string
	DiscriminatorColumn string
	DiscriminatorType   string
	DiscriminatorLength string
	InheritanceType     string
	ValidateRule        string
	AggregateRoot       bool
	Persistent          *bool
	Immutable           *bool
	BelongsTo           string
	BelongsToAt         bool
	Attributes          []*Attribute
	References          []*Reference
	Operations          []*Operation
	Associations        []*Association
	Repository          *Repository
	AggregateLifecycle  bool
	Ordinal             bool
	EnumAttributes      []*EnumAttribute
	EnumValues          []*EnumValue
}

// TraitRef is a trait mixed into a domain object.
type TraitRef struct {
	Name       string
	ExplicitAt bool
}

// Attribute is a scalar field.
type Attribute struct {
	Pos        Position
	Doc        string
	Visibility string
	Type       *ComplexType
	Name       string
	Mods       []Modifier
}

// Reference is a reference to another domain object.
type Reference struct {
	Pos        Position
	Doc        string
	Visibility string
	Type       *ComplexType
	Name       string
	Mods       []Modifier
	Opposite   string
	Label      string
}

// Modifier is a sculptor property modifier such as `key` or `hint = "..."`.
type Modifier struct {
	Not      bool
	Name     string
	Value    string
	HasValue bool
}

// ComplexType is a CML/Sculptor type, collection, or domain-object reference.
type ComplexType struct {
	Collection   string
	TypeName     string
	DomainObject string
	ExplicitAt   bool
	MapKey       string
	MapKeyDomain string
	MapKeyAt     bool
	Inner        *ComplexType
}

// Association is a loose association to a domain object.
type Association struct {
	Doc         string
	Description string
	Target      string
	ExplicitAt  bool
}

// Operation is a service, resource, repository, or domain-object operation.
type Operation struct {
	Pos          Position
	Doc          string
	Abstract     bool
	Visibility   string
	Return       *ComplexType
	Void         bool
	Name         string
	Params       []*Parameter
	Throws       []string
	Hint         string
	Publish      *Publish
	Cache        bool
	Gap          *bool
	Query        string
	Condition    string
	Select       string
	GroupBy      string
	OrderBy      string
	Construct    bool
	Build        bool
	MapFlag      bool
	HTTPMethod   string
	Path         string
	ReturnString string
	ReadOnly     bool
	Write        bool
	Transition   *StateTransition
	Delegate     *Delegate
	RepoDelegate string
}

// Parameter is an operation parameter.
type Parameter struct {
	Doc  string
	Type *ComplexType
	Name string
}

// Delegate is `delegates to Type.operation`.
type Delegate struct {
	ExplicitAt bool
	Target     string
	Operation  string
}

// Publish publishes an event to a channel.
type Publish struct {
	ExplicitAt bool
	Event      string
	Topic      string
	EventBus   string
}

// Service is a domain or application service.
type Service struct {
	Pos          Position
	Comment      string
	Doc          string
	Name         string
	Gap          *bool
	Hint         string
	WebService   bool
	Subscribe    *Subscription
	Dependencies []Dependency
	Operations   []*Operation
	Associations []*Association
}

// Resource is a REST-style resource.
type Resource struct {
	Pos          Position
	Doc          string
	Name         string
	Gap          *bool
	Hint         string
	Scaffold     bool
	Path         string
	Dependencies []Dependency
	Operations   []*Operation
}

// Consumer consumes messages from a channel.
type Consumer struct {
	Pos          Position
	Doc          string
	Name         string
	Hint         string
	Dependencies []Dependency
	Unmarshall   string
	UnmarshallAt bool
	ChannelKind  string
	Channel      string
	Subscribe    *Subscription
}

// Subscription is `subscribe to <channel>`.
type Subscription struct {
	Topic    string
	EventBus string
}

// Dependency is an injected service or repository.
type Dependency struct {
	Inject     bool
	Ref        string
	ExplicitAt bool
}

// Repository is a repository of an aggregate root.
type Repository struct {
	Pos          Position
	Doc          string
	Name         string
	Gap          *bool
	Hint         string
	Subscribe    *Subscription
	Dependencies []Dependency
	Operations   []*Operation
}

// EnumAttribute is a field of an enum.
type EnumAttribute struct {
	Doc  string
	Type string
	Name string
	Key  bool
}

// EnumValue is one enum literal.
type EnumValue struct {
	Doc    string
	Name   string
	Params []EnumParam
}

// EnumParam is a string or integer enum constructor argument.
type EnumParam struct {
	String string
	Int    string
	IsInt  bool
}

// Requirement is a use case or a user story.
type Requirement interface {
	isRequirement()
	ReqName() string
	ReqPos() Position
	ReqFeatures() []*Feature
}

// UseCase is a user requirement in use-case form.
type UseCase struct {
	Pos             Position
	Comment         string
	Name            string
	Actor           string
	SecondaryActors []string
	Features        []*Feature
	Benefit         string
	Scope           string
	Level           string
}

func (*UseCase) isRequirement() {}

func (u *UseCase) ReqName() string { return u.Name }

func (u *UseCase) ReqPos() Position { return u.Pos }

func (u *UseCase) ReqFeatures() []*Feature { return u.Features }

// UserStory is a user story, optionally with an ethical valuation.
type UserStory struct {
	Pos       Position
	Comment   string
	Name      string
	SplitBy   string
	Role      string
	Features  []*Feature
	Benefit   string
	Valuation *Valuation
}

func (*UserStory) isRequirement() {}

func (u *UserStory) ReqName() string { return u.Name }

func (u *UserStory) ReqPos() Position { return u.Pos }

func (u *UserStory) ReqFeatures() []*Feature { return u.Features }

// Feature is one interaction of a use case or story.
type Feature struct {
	Pos              Position
	WantTo           bool
	Verb             string
	Article          string
	Entity           string
	AttrPrep         string
	Attrs            []string
	ContainerPrep    string
	ContainerArticle string
	Container        string
}

// Valuation is the ethical value statement of a user story.
type Valuation struct {
	Promoted       []string
	PromotedPlural bool
	Harmed         []string
	HarmedPlural   bool
	HarmWord       string
}

// StakeholderBlock is a Stakeholders declaration.
type StakeholderBlock struct {
	Pos      Position
	Contexts []string
	Items    []StakeholderItem
}

// StakeholderItem is either a group or a stakeholder.
type StakeholderItem struct {
	Group  *StakeholderGroup
	Person *Stakeholder
}

// StakeholderGroup groups stakeholders.
type StakeholderGroup struct {
	Pos     Position
	Name    string
	Members []*Stakeholder
}

// Stakeholder is one stakeholder.
type Stakeholder struct {
	Pos         Position
	Name        string
	Influence   string
	Interest    string
	Description string
}

// ValueRegister collects values for a bounded context.
type ValueRegister struct {
	Pos        Position
	Name       string
	Context    string
	Clusters   []*ValueCluster
	Values     []*Value
	Epics      []*ValueEpic
	Narratives []*ValueNarrative
	Weightings []*ValueWeighting
}

// ValueCluster groups values around a core value.
type ValueCluster struct {
	Pos            Position
	Name           string
	Core           string
	CoreLiteral    bool
	Demonstrators  []string
	RelatedValues  []string
	OpposingValues []string
	Values         []*Value
	Elicitations   []*ValueElicitation
}

// Value is one ethical or business value.
type Value struct {
	Pos            Position
	Name           string
	IsCore         bool
	Demonstrators  []string
	RelatedValues  []string
	OpposingValues []string
	Elicitations   []*ValueElicitation
}

// ValueElicitation records one stakeholder's view of a value.
type ValueElicitation struct {
	Pos          Position
	Plural       bool
	Stakeholder  string
	Priority     string
	Impact       string
	Consequences []Consequence
}

// Consequence is a good, bad, or neutral consequence.
type Consequence struct {
	Type   string
	Text   string
	Action *Action
}

// Action is a response to a consequence.
type Action struct {
	Text string
	Type string
}

// ValueEpic is a value epic statement.
type ValueEpic struct {
	Pos            Position
	Name           string
	Stakeholder    string
	Value          string
	RealizedValues []string
	ReducedValues  []string
}

// ValueNarrative is a value narrative statement.
type ValueNarrative struct {
	Pos                  Position
	Name                 string
	Feature              string
	PromotedValues       string
	HarmedValues         string
	PreAndPostConditions string
}

// ValueWeighting compares two values. The CML keyword keeps the upstream spelling.
type ValueWeighting struct {
	Pos         Position
	Name        string
	Stakeholder string
	Value1      string
	Value2      string
	Benefits    string
	Harms       string
}

// String renders a complex type in CML notation.
func (t *ComplexType) String() string {
	if t == nil {
		return ""
	}
	if t.Collection == "Map" {
		key := t.MapKey
		if t.MapKeyDomain != "" {
			if t.MapKeyAt {
				key = "@" + t.MapKeyDomain
			} else {
				key = t.MapKeyDomain
			}
		}
		return "Map<" + key + ", " + t.Inner.String() + ">"
	}
	if t.Collection != "" {
		return t.Collection + "<" + t.Inner.String() + ">"
	}
	if t.TypeName != "" && t.DomainObject != "" {
		return t.TypeName + "<@" + t.DomainObject + ">"
	}
	if t.DomainObject != "" {
		if t.ExplicitAt || t.TypeName == "" {
			return "@" + t.DomainObject
		}
		return t.DomainObject
	}
	return t.TypeName
}

// RefName is the domain object named by a reference or collection element.
func (t *ComplexType) RefName() string {
	if t == nil {
		return ""
	}
	if t.Collection != "" && t.Inner != nil {
		return t.Inner.RefName()
	}
	if t.DomainObject != "" {
		return t.DomainObject
	}
	return t.TypeName
}

// LinkedName is set when the type is an explicit domain-object reference (@Type).
func (t *ComplexType) LinkedName() (string, bool) {
	if t == nil {
		return "", false
	}
	if t.Collection != "" {
		return t.Inner.LinkedName()
	}
	if t.DomainObject != "" {
		return t.DomainObject, true
	}
	return "", false
}

// AllAggregates returns aggregates declared on the context and inside its modules.
func (bc *BoundedContext) AllAggregates() []*Aggregate {
	if bc == nil {
		return nil
	}
	out := slices.Clone(bc.Aggregates)
	for _, mod := range bc.Modules {
		out = append(out, mod.Aggregates...)
	}
	return out
}

// FindBC returns the bounded context with the given name.
func (m *Model) FindBC(name string) *BoundedContext {
	if m == nil {
		return nil
	}
	for _, bc := range m.BoundedContexts {
		if bc.Name == name {
			return bc
		}
	}
	return nil
}

// FindDomain returns the domain with the given name.
func (m *Model) FindDomain(name string) *Domain {
	for _, d := range m.Domains {
		if d.Name == name {
			return d
		}
	}
	return nil
}

// FindSubdomain returns the subdomain with the given name.
func (m *Model) FindSubdomain(name string) (*Domain, *Subdomain) {
	for _, d := range m.Domains {
		for _, sd := range d.Subdomains {
			if sd.Name == name {
				return d, sd
			}
		}
	}
	return nil, nil
}

// FindAggregate returns the first aggregate with the given name and the context that owns it.
func (m *Model) FindAggregate(name string) (*BoundedContext, *Module, *Aggregate) {
	for _, bc := range m.BoundedContexts {
		for _, agg := range bc.Aggregates {
			if agg.Name == name {
				return bc, nil, agg
			}
		}
		for _, mod := range bc.Modules {
			for _, agg := range mod.Aggregates {
				if agg.Name == name {
					return bc, mod, agg
				}
			}
		}
	}
	return nil, nil, nil
}

// WalkObjects visits every domain object in the model.
func (m *Model) WalkObjects(fn func(owner *BoundedContext, agg *Aggregate, obj *DomainObject)) {
	if m == nil {
		return
	}
	for _, bc := range m.BoundedContexts {
		for _, obj := range bc.moduleObjects() {
			fn(bc, nil, obj)
		}
		for _, agg := range bc.AllAggregates() {
			for _, obj := range agg.Objects {
				fn(bc, agg, obj)
			}
		}
		if bc.Application != nil {
			for _, obj := range bc.Application.Commands {
				fn(bc, nil, obj)
			}
			for _, obj := range bc.Application.Events {
				fn(bc, nil, obj)
			}
		}
	}
	for _, d := range m.Domains {
		for _, sd := range d.Subdomains {
			for _, obj := range sd.Entities {
				fn(nil, nil, obj)
			}
		}
	}
}

func (bc *BoundedContext) moduleObjects() []*DomainObject {
	var out []*DomainObject
	for _, mod := range bc.Modules {
		out = append(out, mod.Objects...)
	}
	return out
}

// DetachAggregate removes an aggregate from the context and returns it.
func (bc *BoundedContext) DetachAggregate(name string) *Aggregate {
	for i, agg := range bc.Aggregates {
		if agg.Name == name {
			bc.Aggregates = slices.Delete(bc.Aggregates, i, i+1)
			return agg
		}
	}
	for _, mod := range bc.Modules {
		for i, agg := range mod.Aggregates {
			if agg.Name == name {
				mod.Aggregates = slices.Delete(mod.Aggregates, i, i+1)
				return agg
			}
		}
	}
	return nil
}
