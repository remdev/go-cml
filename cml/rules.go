package cml

import "slices"

// RuleCode is a stable machine-readable identity, independent of message text.
type RuleCode string

// RuleCategory describes a rule's concern, not whether it can be suppressed.
// Existing checks retain their resolution and validation behavior.
type RuleCategory string

const (
	CategoryStructure  RuleCategory = "structure"
	CategoryResolution RuleCategory = "resolution"
	CategorySemantics  RuleCategory = "semantics"
	CategoryAdvice     RuleCategory = "advice"
)

// Rule describes a built-in rule with its unchanged CML compatibility severity.
type Rule struct {
	Code            RuleCode
	Category        RuleCategory
	DefaultSeverity Severity
}

// Built-in rule identities. IDs must not be reassigned to different rules.
const (
	RuleDuplicateBoundedContext       RuleCode = "cml.duplicate-bounded-context"
	RuleDuplicateContextService       RuleCode = "cml.duplicate-context-service"
	RuleDuplicateDomain               RuleCode = "cml.duplicate-domain"
	RuleDuplicateSubdomain            RuleCode = "cml.duplicate-subdomain"
	RuleDuplicateSubdomainService     RuleCode = "cml.duplicate-subdomain-service"
	RuleDuplicateDomainObject         RuleCode = "cml.duplicate-domain-object"
	RuleDuplicateModule               RuleCode = "cml.duplicate-module"
	RuleUngroupedModuleMembers        RuleCode = "cml.ungrouped-module-members"
	RuleDuplicateAggregate            RuleCode = "cml.duplicate-aggregate"
	RuleDuplicateRequirement          RuleCode = "cml.duplicate-requirement"
	RuleDuplicateFlow                 RuleCode = "cml.duplicate-flow"
	RuleDuplicateCoordination         RuleCode = "cml.duplicate-coordination"
	RuleReusedDomainObjectName        RuleCode = "cml.reused-domain-object-name"
	RuleUnknownBoundedContext         RuleCode = "cml.unknown-bounded-context"
	RuleOrganizationalMapWithoutTeams RuleCode = "cml.organizational-map-without-teams"
	RuleTeamOnSystemLandscape         RuleCode = "cml.team-on-system-landscape"
	RuleContextNotInMap               RuleCode = "cml.context-not-in-map"
	RuleSelfRelationship              RuleCode = "cml.self-relationship"
	RuleCustomerSupplierOHS           RuleCode = "cml.customer-supplier-ohs"
	RuleCustomerSupplierACL           RuleCode = "cml.customer-supplier-acl"
	RuleCustomerSupplierConformist    RuleCode = "cml.customer-supplier-conformist"
	RuleExposedAggregateOwnership     RuleCode = "cml.exposed-aggregate-ownership"
	RuleRealizesRequiresTeam          RuleCode = "cml.realizes-requires-team"
	RuleUnknownDomain                 RuleCode = "cml.unknown-domain"
	RuleRedundantDomainImplementation RuleCode = "cml.redundant-domain-implementation"
	RuleMultipleImplementedDomains    RuleCode = "cml.multiple-implemented-domains"
	RuleSubdomainMatchesDomain        RuleCode = "cml.subdomain-matches-domain"
	RuleUnknownRequirement            RuleCode = "cml.unknown-requirement"
	RuleMultipleAggregateRoots        RuleCode = "cml.multiple-aggregate-roots"
	RuleMultipleLifecycleEnums        RuleCode = "cml.multiple-lifecycle-enums"
	RuleOwnerRequiresTeam             RuleCode = "cml.owner-requires-team"
	RuleFeatureEntityIdentifier       RuleCode = "cml.feature-entity-identifier"
	RuleFeatureVerbIdentifier         RuleCode = "cml.feature-verb-identifier"
	RuleStorySplitSuggestion          RuleCode = "cml.story-split-suggestion"
	RuleUnknownUserStory              RuleCode = "cml.unknown-user-story"
	RuleFlowMemberScope               RuleCode = "cml.flow-member-scope"
	RuleUnknownDomainEvent            RuleCode = "cml.unknown-domain-event"
	RuleUnknownAggregate              RuleCode = "cml.unknown-aggregate"
	RuleUnknownAggregateState         RuleCode = "cml.unknown-aggregate-state"
	RuleCoordinationReachability      RuleCode = "cml.coordination-reachability"
	RuleUnknownApplicationService     RuleCode = "cml.unknown-application-service"
	RuleUnknownApplicationOperation   RuleCode = "cml.unknown-application-operation"
	RuleAmbiguousApplicationOperation RuleCode = "cml.ambiguous-application-operation"
	RuleUnknownDomainObject           RuleCode = "cml.unknown-domain-object"
	RuleAmbiguousDomainObject         RuleCode = "cml.ambiguous-domain-object"
	RuleObjectReachability            RuleCode = "cml.object-reachability"
	RulePrimitiveIdentifier           RuleCode = "cml.primitive-identifier"
	RuleNullablePrimitive             RuleCode = "cml.nullable-primitive"
	RuleUnknownTrait                  RuleCode = "cml.unknown-trait"
)

type ruleDefinition struct {
	Rule
	format string
}

var ruleDefinitions = map[RuleCode]ruleDefinition{
	RuleDuplicateBoundedContext:       {Rule: Rule{Code: RuleDuplicateBoundedContext, Category: CategoryStructure, DefaultSeverity: SevError}, format: "Multiple bounded contexts with the name '%s' have been declared."},
	RuleDuplicateContextService:       {Rule: Rule{Code: RuleDuplicateContextService, Category: CategoryStructure, DefaultSeverity: SevError}, format: "Multiple services with the name '%s' have been declared in this Bounded Context."},
	RuleDuplicateDomain:               {Rule: Rule{Code: RuleDuplicateDomain, Category: CategoryStructure, DefaultSeverity: SevError}, format: "Multiple domains with the name '%s' have been declared."},
	RuleDuplicateSubdomain:            {Rule: Rule{Code: RuleDuplicateSubdomain, Category: CategoryStructure, DefaultSeverity: SevError}, format: "Multiple subdomains with the name '%s' have been declared."},
	RuleDuplicateSubdomainService:     {Rule: Rule{Code: RuleDuplicateSubdomainService, Category: CategoryStructure, DefaultSeverity: SevError}, format: "Multiple services with the name '%s' have been declared in this Subdomain."},
	RuleDuplicateDomainObject:         {Rule: Rule{Code: RuleDuplicateDomainObject, Category: CategoryStructure, DefaultSeverity: SevError}, format: "Multiple domain objects with the name '%s' have been declared."},
	RuleDuplicateModule:               {Rule: Rule{Code: RuleDuplicateModule, Category: CategoryStructure, DefaultSeverity: SevError}, format: "Duplicate name. There is already an existing Module named '%s'."},
	RuleUngroupedModuleMembers:        {Rule: Rule{Code: RuleUngroupedModuleMembers, Category: CategoryAdvice, DefaultSeverity: SevWarning}, format: "The module '%s' contains Services and/or Entities that are not part of an Aggregate. The MDSL and Service Cutter generators ignore them. Please move them into an Aggregate."},
	RuleDuplicateAggregate:            {Rule: Rule{Code: RuleDuplicateAggregate, Category: CategoryStructure, DefaultSeverity: SevError}, format: "Multiple aggregates with the name '%s' have been declared."},
	RuleDuplicateRequirement:          {Rule: Rule{Code: RuleDuplicateRequirement, Category: CategoryStructure, DefaultSeverity: SevError}, format: "Multiple use cases with the name '%s' have been declared."},
	RuleDuplicateFlow:                 {Rule: Rule{Code: RuleDuplicateFlow, Category: CategoryStructure, DefaultSeverity: SevError}, format: "Multiple flows with the name '%s' have been declared. Please provide unique names!"},
	RuleDuplicateCoordination:         {Rule: Rule{Code: RuleDuplicateCoordination, Category: CategoryStructure, DefaultSeverity: SevError}, format: "Multiple coordinations with the name '%s' have been declared. Please provide unique names!"},
	RuleReusedDomainObjectName:        {Rule: Rule{Code: RuleReusedDomainObjectName, Category: CategoryAdvice, DefaultSeverity: SevWarning}, format: "The name '%s' is used for multiple domain objects. We suggest to keep the names of domain objects distinct."},
	RuleUnknownBoundedContext:         {Rule: Rule{Code: RuleUnknownBoundedContext, Category: CategoryResolution, DefaultSeverity: SevError}, format: "Couldn't resolve reference to BoundedContext '%s'."},
	RuleOrganizationalMapWithoutTeams: {Rule: Rule{Code: RuleOrganizationalMapWithoutTeams, Category: CategoryAdvice, DefaultSeverity: SevWarning}, format: "Your Context Map is of the type ORGANIZATIONAL but does not contain Bounded Contexts of the type TEAM. This type of Context Map is intended to model team relationships."},
	RuleTeamOnSystemLandscape:         {Rule: Rule{Code: RuleTeamOnSystemLandscape, Category: CategorySemantics, DefaultSeverity: SevError}, format: "A Bounded Context of type 'TEAM' is not allowed on a SYSTEM_LANDSCAPE map."},
	RuleContextNotInMap:               {Rule: Rule{Code: RuleContextNotInMap, Category: CategorySemantics, DefaultSeverity: SevError}, format: "The Bounded Context '%s' is not part of the Context Map."},
	RuleSelfRelationship:              {Rule: Rule{Code: RuleSelfRelationship, Category: CategorySemantics, DefaultSeverity: SevError}, format: "Bounded context relationships must be declared between two different bounded contexts."},
	RuleCustomerSupplierOHS:           {Rule: Rule{Code: RuleCustomerSupplierOHS, Category: CategorySemantics, DefaultSeverity: SevError}, format: "The OPEN-HOST SERVICE pattern is not applicable for a Customer-Supplier relationship."},
	RuleCustomerSupplierACL:           {Rule: Rule{Code: RuleCustomerSupplierACL, Category: CategoryAdvice, DefaultSeverity: SevWarning}, format: "Are you sure you need an ANTICORRUPTION LAYER here? This pattern should not be necessarily needed in a Customer-Supplier relationship."},
	RuleCustomerSupplierConformist:    {Rule: Rule{Code: RuleCustomerSupplierConformist, Category: CategorySemantics, DefaultSeverity: SevError}, format: "The CONFORMIST pattern is not applicable for a Customer-Supplier relationship."},
	RuleExposedAggregateOwnership:     {Rule: Rule{Code: RuleExposedAggregateOwnership, Category: CategorySemantics, DefaultSeverity: SevError}, format: "The aggregate '%s' is not part of the upstream context '%s'."},
	RuleRealizesRequiresTeam:          {Rule: Rule{Code: RuleRealizesRequiresTeam, Category: CategorySemantics, DefaultSeverity: SevError}, format: "Only teams can realize bounded contexts. '%s' is not a team!"},
	RuleUnknownDomain:                 {Rule: Rule{Code: RuleUnknownDomain, Category: CategoryResolution, DefaultSeverity: SevError}, format: "Couldn't resolve reference to Domain '%s'."},
	RuleRedundantDomainImplementation: {Rule: Rule{Code: RuleRedundantDomainImplementation, Category: CategorySemantics, DefaultSeverity: SevError}, format: "The subdomain '%s' is already implemented through its domain '%s'."},
	RuleMultipleImplementedDomains:    {Rule: Rule{Code: RuleMultipleImplementedDomains, Category: CategoryAdvice, DefaultSeverity: SevWarning}, format: "Are you sure you want to implement multiple Domains within one Bounded Context? A Bounded Context should typically implement only a part of your Domain; one or multiple Subdomains."},
	RuleSubdomainMatchesDomain:        {Rule: Rule{Code: RuleSubdomainMatchesDomain, Category: CategoryStructure, DefaultSeverity: SevError}, format: "The subdomain '%s' has the same name as the domain. Please define a unique subdomain name."},
	RuleUnknownRequirement:            {Rule: Rule{Code: RuleUnknownRequirement, Category: CategoryResolution, DefaultSeverity: SevError}, format: "Couldn't resolve reference to UserRequirement '%s'."},
	RuleMultipleAggregateRoots:        {Rule: Rule{Code: RuleMultipleAggregateRoots, Category: CategorySemantics, DefaultSeverity: SevError}, format: "Your aggregate '%s' contains multiple aggregate roots. An aggregate must only contain one root."},
	RuleMultipleLifecycleEnums:        {Rule: Rule{Code: RuleMultipleLifecycleEnums, Category: CategorySemantics, DefaultSeverity: SevError}, format: "Your aggregate '%s' contains multiple enums that define its states. Ensure that only one enum declares the states of the Aggregate."},
	RuleOwnerRequiresTeam:             {Rule: Rule{Code: RuleOwnerRequiresTeam, Category: CategorySemantics, DefaultSeverity: SevError}, format: "'%s' is not a team. The owner attribute must refer to a Bounded Context representing a team (type = TEAM)!"},
	RuleFeatureEntityIdentifier:       {Rule: Rule{Code: RuleFeatureEntityIdentifier, Category: CategoryAdvice, DefaultSeverity: SevWarning}, format: "The entity name should consist of the following characters: a-z, A-Z, 0-9, _. Please rename the entity accordingly before applying transformations."},
	RuleFeatureVerbIdentifier:         {Rule: Rule{Code: RuleFeatureVerbIdentifier, Category: CategoryAdvice, DefaultSeverity: SevWarning}, format: "The verb should consist of the following characters: a-z, A-Z, 0-9, _. Please rename it accordingly before applying transformations."},
	RuleStorySplitSuggestion:          {Rule: Rule{Code: RuleStorySplitSuggestion, Category: CategoryAdvice, DefaultSeverity: SevInfo}, format: "You could split your story by its operation/verb."},
	RuleUnknownUserStory:              {Rule: Rule{Code: RuleUnknownUserStory, Category: CategoryResolution, DefaultSeverity: SevError}, format: "Couldn't resolve reference to UserStory '%s'."},
	RuleFlowMemberScope:               {Rule: Rule{Code: RuleFlowMemberScope, Category: CategoryResolution, DefaultSeverity: SevError}, format: "The command or operation '%s' is not part of the '%s' Bounded Context. Please ensure that your workflow only uses commands, operations and events that are part of the same Bounded Context."},
	RuleUnknownDomainEvent:            {Rule: Rule{Code: RuleUnknownDomainEvent, Category: CategoryResolution, DefaultSeverity: SevError}, format: "Couldn't resolve reference to DomainEvent '%s'."},
	RuleUnknownAggregate:              {Rule: Rule{Code: RuleUnknownAggregate, Category: CategoryResolution, DefaultSeverity: SevError}, format: "Couldn't resolve reference to Aggregate '%s'."},
	RuleUnknownAggregateState:         {Rule: Rule{Code: RuleUnknownAggregateState, Category: CategoryResolution, DefaultSeverity: SevError}, format: "'%s' is not a state of the Aggregate '%s'."},
	RuleCoordinationReachability:      {Rule: Rule{Code: RuleCoordinationReachability, Category: CategorySemantics, DefaultSeverity: SevError}, format: "The Bounded Context '%s' is not reachable through Context Map relationships. Maybe you want to create a relationship with this context?"},
	RuleUnknownApplicationService:     {Rule: Rule{Code: RuleUnknownApplicationService, Category: CategoryResolution, DefaultSeverity: SevError}, format: "The Service '%s' is not part of the Bounded Context '%s' Application."},
	RuleUnknownApplicationOperation:   {Rule: Rule{Code: RuleUnknownApplicationOperation, Category: CategoryResolution, DefaultSeverity: SevError}, format: "The operation '%s' is not part of the Service '%s' of the Bounded Context '%s' Application."},
	RuleAmbiguousApplicationOperation: {Rule: Rule{Code: RuleAmbiguousApplicationOperation, Category: CategoryResolution, DefaultSeverity: SevWarning}, format: "The reference to operation '%s' is ambiguous, since there exist multiple operations with that name in Service '%s'. We suggest keeping the operation names distinct when referencing them through Coordinations."},
	RuleUnknownDomainObject:           {Rule: Rule{Code: RuleUnknownDomainObject, Category: CategoryResolution, DefaultSeverity: SevError}, format: "Couldn't resolve reference to '%s'."},
	RuleAmbiguousDomainObject:         {Rule: Rule{Code: RuleAmbiguousDomainObject, Category: CategoryResolution, DefaultSeverity: SevWarning}, format: "The reference to the type '%s' is ambiguous, since there exist multiple domain objects with that name in your model. We suggest to keep the names of domain objects distinct."},
	RuleObjectReachability:            {Rule: Rule{Code: RuleObjectReachability, Category: CategorySemantics, DefaultSeverity: SevWarning}, format: "The object '%s' is not part of this Bounded Context and not reachable through Context Map relationships. Maybe you want to create a relationship with the context that contains that object?"},
	RulePrimitiveIdentifier:           {Rule: Rule{Code: RulePrimitiveIdentifier, Category: CategoryAdvice, DefaultSeverity: SevWarning}, format: "This attribute seems to be an identifier. Have you thought about creating a Value Object?"},
	RuleNullablePrimitive:             {Rule: Rule{Code: RuleNullablePrimitive, Category: CategorySemantics, DefaultSeverity: SevError}, format: "nullable is not relevant for primitive types"},
	RuleUnknownTrait:                  {Rule: Rule{Code: RuleUnknownTrait, Category: CategoryResolution, DefaultSeverity: SevError}, format: "Couldn't resolve reference to Trait '%s'."},
}

// LookupRule returns a copy of the metadata for a built-in validation rule.
func LookupRule(code RuleCode) (Rule, bool) {
	definition, ok := ruleDefinitions[code]
	return definition.Rule, ok
}

// Rules returns independent metadata sorted by code. Changing the returned
// slice or its values cannot change subsequent validation behavior.
func Rules() []Rule {
	rules := make([]Rule, 0, len(ruleDefinitions))
	for _, definition := range ruleDefinitions {
		rules = append(rules, definition.Rule)
	}
	slices.SortFunc(rules, func(a, b Rule) int {
		if a.Code < b.Code {
			return -1
		}
		if a.Code > b.Code {
			return 1
		}
		return 0
	})
	return rules
}
