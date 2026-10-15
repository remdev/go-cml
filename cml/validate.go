package cml

import (
	"fmt"
	"regexp"
	"slices"
)

// Severity is the level of a semantic diagnostic.
type Severity int

const (
	// SevError rejects the model.
	SevError Severity = iota
	// SevWarning is reported without failing validation.
	SevWarning
	// SevInfo is a suggestion.
	SevInfo
)

func (s Severity) String() string {
	switch s {
	case SevWarning:
		return "warning"
	case SevInfo:
		return "info"
	default:
		return "error"
	}
}

// Diagnostic is one semantic finding.
type Diagnostic struct {
	Severity Severity
	Pos      Position
	Message  string
}

func (d Diagnostic) String() string {
	if d.Pos.File == "" {
		return d.Severity.String() + ": " + d.Message
	}
	return fmt.Sprintf("%s:%d:%d: %s: %s", d.Pos.File, d.Pos.Line, d.Pos.Col, d.Severity, d.Message)
}

// HasErrors reports whether any diagnostic is an error.
func HasErrors(ds []Diagnostic) bool {
	for _, d := range ds {
		if d.Severity == SevError {
			return true
		}
	}
	return false
}

var idPattern = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

var primitiveTypes = map[string]bool{
	"String": true, "int": true, "Integer": true, "long": true, "Long": true,
	"boolean": true, "Boolean": true, "Date": true, "DateTime": true, "Timestamp": true,
	"BigDecimal": true, "BigInteger": true, "double": true, "Double": true,
	"float": true, "Float": true, "Blob": true, "Clob": true, "Key": true,
}

// Validate checks semantic rules of the Context Mapper language.
func (m *Model) Validate() []Diagnostic {
	if m == nil {
		return nil
	}
	var ds []Diagnostic
	add := func(sev Severity, pos Position, format string, args ...any) {
		ds = append(ds, Diagnostic{Severity: sev, Pos: pos, Message: fmt.Sprintf(format, args...)})
	}
	m.checkUnique(add)
	m.checkMap(add)
	m.checkTeams(add)
	m.checkDomains(add)
	m.checkAggregates(add)
	m.checkRequirements(add)
	m.checkFlows(add)
	m.checkOperationStates(add)
	m.checkReferences(add)
	return ds
}

type adder func(Severity, Position, string, ...any)

func (m *Model) checkUnique(add adder) {
	countBC := map[string]int{}
	for _, bc := range m.BoundedContexts {
		countBC[bc.Name]++
	}
	for _, bc := range m.BoundedContexts {
		if countBC[bc.Name] > 1 {
			add(SevError, bc.Pos, "Multiple bounded contexts with the name '%s' have been declared.", bc.Name)
		}
		names := map[string]bool{}
		for _, svc := range bc.allServices() {
			if names[svc.Name] {
				add(SevError, svc.Pos, "Multiple services with the name '%s' have been declared in this Bounded Context.", svc.Name)
			}
			names[svc.Name] = true
		}
	}
	countDomain := map[string]int{}
	countSub := map[string]int{}
	for _, d := range m.Domains {
		countDomain[d.Name]++
		for _, sd := range d.Subdomains {
			countSub[sd.Name]++
		}
	}
	for _, d := range m.Domains {
		if countDomain[d.Name] > 1 {
			add(SevError, d.Pos, "Multiple domains with the name '%s' have been declared.", d.Name)
		}
		for _, sd := range d.Subdomains {
			if countSub[sd.Name] > 1 {
				add(SevError, sd.Pos, "Multiple subdomains with the name '%s' have been declared.", sd.Name)
			}
			svcNames := map[string]bool{}
			for _, svc := range sd.Services {
				if svcNames[svc.Name] {
					add(SevError, svc.Pos, "Multiple services with the name '%s' have been declared in this Subdomain.", svc.Name)
				}
				svcNames[svc.Name] = true
			}
			seen := map[string]bool{}
			for _, obj := range sd.Entities {
				if obj.Name == "" {
					continue
				}
				if seen[obj.Name] {
					add(SevError, obj.Pos, "Multiple domain objects with the name '%s' have been declared.", obj.Name)
				}
				seen[obj.Name] = true
			}
		}
	}
	countMod := map[string]int{}
	countAgg := map[string]int{}
	var modules []*Module
	var aggs []*Aggregate
	for _, bc := range m.BoundedContexts {
		modules = append(modules, bc.Modules...)
		aggs = append(aggs, bc.AllAggregates()...)
	}
	for _, mod := range modules {
		countMod[mod.Name]++
	}
	for _, agg := range aggs {
		countAgg[agg.Name]++
	}
	for _, mod := range modules {
		if countMod[mod.Name] > 1 {
			add(SevError, mod.Pos, "Duplicate name. There is already an existing Module named '%s'.", mod.Name)
		}
		seen := map[string]bool{}
		for _, obj := range mod.Objects {
			if seen[obj.Name] {
				add(SevError, obj.Pos, "Multiple domain objects with the name '%s' have been declared.", obj.Name)
			}
			seen[obj.Name] = true
		}
		if len(mod.Services) > 0 || hasEntity(mod.Objects) {
			add(SevWarning, mod.Pos, "The module '%s' contains Services and/or Entities that are not part of an Aggregate. The MDSL and Service Cutter generators ignore them. Please move them into an Aggregate.", mod.Name)
		}
	}
	for _, agg := range aggs {
		if countAgg[agg.Name] > 1 {
			add(SevError, agg.Pos, "Multiple aggregates with the name '%s' have been declared.", agg.Name)
		}
		seen := map[string]bool{}
		for _, obj := range agg.Objects {
			if obj.Name == "" {
				continue
			}
			if seen[obj.Name] {
				add(SevError, obj.Pos, "Multiple domain objects with the name '%s' have been declared.", obj.Name)
			}
			seen[obj.Name] = true
		}
	}
	countReq := map[string]int{}
	for _, req := range m.Requirements {
		countReq[req.ReqName()]++
	}
	for _, req := range m.Requirements {
		if countReq[req.ReqName()] > 1 {
			add(SevError, req.ReqPos(), "Multiple use cases with the name '%s' have been declared.", req.ReqName())
		}
	}
	countFlow := map[string]int{}
	countCoord := map[string]int{}
	type namedPos struct {
		name string
		pos  Position
	}
	var flows []namedPos
	var coords []namedPos
	for _, bc := range m.BoundedContexts {
		if bc.Application == nil {
			continue
		}
		for _, f := range bc.Application.Flows {
			countFlow[f.Name]++
			flows = append(flows, namedPos{f.Name, f.Pos})
		}
		for _, c := range bc.Application.Coordinations {
			countCoord[c.Name]++
			coords = append(coords, namedPos{c.Name, c.Pos})
		}
	}
	for _, f := range flows {
		if countFlow[f.name] > 1 {
			add(SevError, f.pos, "Multiple flows with the name '%s' have been declared. Please provide unique names!", f.name)
		}
	}
	for _, c := range coords {
		if countCoord[c.name] > 1 {
			add(SevError, c.pos, "Multiple coordinations with the name '%s' have been declared. Please provide unique names!", c.name)
		}
	}
	objCount := map[string]int{}
	m.WalkObjects(func(_ *BoundedContext, _ *Aggregate, obj *DomainObject) {
		objCount[obj.Name]++
	})
	m.WalkObjects(func(_ *BoundedContext, _ *Aggregate, obj *DomainObject) {
		if objCount[obj.Name] > 1 {
			add(SevWarning, obj.Pos, "The name '%s' is used for multiple domain objects. We suggest to keep the names of domain objects distinct.", obj.Name)
		}
	})
}

func hasEntity(objs []*DomainObject) bool {
	for _, obj := range objs {
		if obj.Kind == KindEntity {
			return true
		}
	}
	return false
}

func (m *Model) checkMap(add adder) {
	if m.Map == nil {
		return
	}
	contains := map[string]bool{}
	for _, name := range m.Map.Contains {
		if m.FindBC(name) == nil {
			add(SevError, m.Map.Pos, "Couldn't resolve reference to BoundedContext '%s'.", name)
			continue
		}
		contains[name] = true
	}
	teamCount := 0
	for _, name := range m.Map.Contains {
		bc := m.FindBC(name)
		if bc != nil && bc.Type == "TEAM" {
			teamCount++
		}
	}
	if m.Map.Type == "ORGANIZATIONAL" && teamCount == 0 {
		add(SevWarning, m.Map.Pos, "Your Context Map is of the type ORGANIZATIONAL but does not contain Bounded Contexts of the type TEAM. This type of Context Map is intended to model team relationships.")
	}
	if m.Map.Type == "SYSTEM_LANDSCAPE" {
		for _, name := range m.Map.Contains {
			bc := m.FindBC(name)
			if bc != nil && bc.Type == "TEAM" {
				add(SevError, m.Map.Pos, "A Bounded Context of type 'TEAM' is not allowed on a SYSTEM_LANDSCAPE map.")
			}
		}
	}
	for _, rel := range m.Map.Relationships {
		left, right := rel.ends()
		for _, name := range []string{left, right} {
			if name == "" {
				continue
			}
			if m.FindBC(name) == nil {
				add(SevError, rel.Pos, "Couldn't resolve reference to BoundedContext '%s'.", name)
				continue
			}
			if !contains[name] {
				add(SevError, rel.Pos, "The Bounded Context '%s' is not part of the Context Map.", name)
			}
		}
		if left != "" && left == right {
			add(SevError, rel.Pos, "Bounded context relationships must be declared between two different bounded contexts.")
		}
		if rel.Kind == RelCustomerSupplier {
			if slices.Contains(rel.UpstreamRoles, "OHS") {
				add(SevError, rel.Pos, "The OPEN-HOST SERVICE pattern is not applicable for a Customer-Supplier relationship.")
			}
			if slices.Contains(rel.DownstreamRoles, "ACL") {
				add(SevWarning, rel.Pos, "Are you sure you need an ANTICORRUPTION LAYER here? This pattern should not be necessarily needed in a Customer-Supplier relationship.")
			}
			if slices.Contains(rel.DownstreamRoles, "CF") {
				add(SevError, rel.Pos, "The CONFORMIST pattern is not applicable for a Customer-Supplier relationship.")
			}
		}
		if rel.Upstream != "" {
			up := m.FindBC(rel.Upstream)
			if up == nil {
				continue
			}
			known := map[string]bool{}
			for _, agg := range up.AllAggregates() {
				known[agg.Name] = true
			}
			for _, name := range rel.ExposedAggregates {
				if !known[name] {
					add(SevError, rel.Pos, "The aggregate '%s' is not part of the upstream context '%s'.", name, rel.Upstream)
				}
			}
		}
	}
}

func (rel *Relationship) ends() (string, string) {
	if rel.Kind == RelPartnership || rel.Kind == RelSharedKernel {
		return rel.Participant1, rel.Participant2
	}
	return rel.Upstream, rel.Downstream
}

func (m *Model) checkTeams(add adder) {
	for _, bc := range m.BoundedContexts {
		if len(bc.Realizes) > 0 && bc.Type != "TEAM" {
			add(SevError, bc.Pos, "Only teams can realize bounded contexts. '%s' is not a team!", bc.Name)
		}
		for _, name := range bc.Realizes {
			if m.FindBC(name) == nil {
				add(SevError, bc.Pos, "Couldn't resolve reference to BoundedContext '%s'.", name)
			}
		}
		if bc.Refines != "" && m.FindBC(bc.Refines) == nil {
			add(SevError, bc.Pos, "Couldn't resolve reference to BoundedContext '%s'.", bc.Refines)
		}
		domains := 0
		for _, part := range bc.Implements {
			domain := m.FindDomain(part)
			_, sub := m.FindSubdomain(part)
			if domain == nil && sub == nil {
				add(SevError, bc.Pos, "Couldn't resolve reference to Domain '%s'.", part)
				continue
			}
			if domain != nil {
				domains++
			}
			if sub != nil {
				parent, _ := m.FindSubdomain(part)
				implemented := false
				for _, other := range bc.Implements {
					if other == parent.Name {
						implemented = true
					}
				}
				if implemented {
					add(SevError, bc.Pos, "The subdomain '%s' is already implemented through its domain '%s'.", sub.Name, parent.Name)
				}
			}
		}
		if domains > 1 {
			add(SevWarning, bc.Pos, "Are you sure you want to implement multiple Domains within one Bounded Context? A Bounded Context should typically implement only a part of your Domain; one or multiple Subdomains.")
		}
	}
}

func (m *Model) checkDomains(add adder) {
	for _, d := range m.Domains {
		for _, sd := range d.Subdomains {
			if sd.Name == d.Name {
				add(SevError, sd.Pos, "The subdomain '%s' has the same name as the domain. Please define a unique subdomain name.", sd.Name)
			}
			for _, name := range sd.Supports {
				if m.findRequirement(name) == nil {
					add(SevError, sd.Pos, "Couldn't resolve reference to UserRequirement '%s'.", name)
				}
			}
		}
	}
}

func (m *Model) checkAggregates(add adder) {
	for _, bc := range m.BoundedContexts {
		for _, agg := range bc.AllAggregates() {
			var roots []*DomainObject
			var states []*DomainObject
			for _, obj := range agg.Objects {
				if obj.AggregateRoot && obj.Kind != KindEnum {
					roots = append(roots, obj)
				}
				if obj.Kind == KindEnum && obj.AggregateLifecycle {
					states = append(states, obj)
				}
			}
			if len(roots) > 1 {
				for _, obj := range roots {
					add(SevError, obj.Pos, "Your aggregate '%s' contains multiple aggregate roots. An aggregate must only contain one root.", agg.Name)
				}
			}
			if len(states) > 1 {
				for _, obj := range states {
					add(SevError, obj.Pos, "Your aggregate '%s' contains multiple enums that define its states. Ensure that only one enum declares the states of the Aggregate.", agg.Name)
				}
			}
			if agg.Owner != "" {
				owner := m.FindBC(agg.Owner)
				if owner == nil {
					add(SevError, agg.Pos, "Couldn't resolve reference to BoundedContext '%s'.", agg.Owner)
				} else if owner.Type != "TEAM" {
					add(SevError, agg.Pos, "'%s' is not a team. The owner attribute must refer to a Bounded Context representing a team (type = TEAM)!", owner.Name)
				}
			}
			for _, name := range agg.RequirementRefs {
				if m.findRequirement(name) == nil {
					add(SevError, agg.Pos, "Couldn't resolve reference to UserRequirement '%s'.", name)
				}
			}
		}
	}
}

func (m *Model) checkRequirements(add adder) {
	for _, req := range m.Requirements {
		story, _ := req.(*UserStory)
		for _, feature := range req.ReqFeatures() {
			if feature.Entity != "" && !idPattern.MatchString(feature.Entity) {
				add(SevWarning, feature.Pos, "The entity name should consist of the following characters: a-z, A-Z, 0-9, _. Please rename the entity accordingly before applying transformations.")
			}
			if feature.Verb != "" && !idPattern.MatchString(feature.Verb) {
				add(SevWarning, feature.Pos, "The verb should consist of the following characters: a-z, A-Z, 0-9, _. Please rename it accordingly before applying transformations.")
			}
			if story != nil && story.SplitBy == "" && len(story.Features) == 1 && feature.Verb != "" {
				add(SevInfo, feature.Pos, "You could split your story by its operation/verb.")
			}
		}
		if story != nil && story.SplitBy != "" && m.findRequirement(story.SplitBy) == nil {
			add(SevError, story.Pos, "Couldn't resolve reference to UserStory '%s'.", story.SplitBy)
		}
	}
}

func (m *Model) checkFlows(add adder) {
	for _, bc := range m.BoundedContexts {
		if bc.Application == nil {
			continue
		}
		for _, flow := range bc.Application.Flows {
			for _, step := range flow.Steps {
				if step.Command != "" && !bc.hasCommand(step.Command) {
					add(SevError, step.Pos, "The command or operation '%s' is not part of the '%s' Bounded Context. Please ensure that your workflow only uses commands, operations and events that are part of the same Bounded Context.", step.Command, bc.Name)
				}
				if step.Operation != "" && !bc.hasOperation(step.Operation) {
					add(SevError, step.Pos, "The command or operation '%s' is not part of the '%s' Bounded Context. Please ensure that your workflow only uses commands, operations and events that are part of the same Bounded Context.", step.Operation, bc.Name)
				}
				for _, name := range step.Commands {
					if !bc.hasCommand(name) {
						add(SevError, step.Pos, "The command or operation '%s' is not part of the '%s' Bounded Context. Please ensure that your workflow only uses commands, operations and events that are part of the same Bounded Context.", name, bc.Name)
					}
				}
				for _, name := range step.Operations {
					if !bc.hasOperation(name) {
						add(SevError, step.Pos, "The command or operation '%s' is not part of the '%s' Bounded Context. Please ensure that your workflow only uses commands, operations and events that are part of the same Bounded Context.", name, bc.Name)
					}
				}
				for _, name := range step.Events {
					if !bc.hasEvent(name) {
						add(SevError, step.Pos, "Couldn't resolve reference to DomainEvent '%s'.", name)
					}
				}
				if step.Aggregate != "" {
					_, _, agg := m.FindAggregate(step.Aggregate)
					if agg == nil || !bc.ownsAggregate(agg) {
						add(SevError, step.Pos, "Couldn't resolve reference to Aggregate '%s'.", step.Aggregate)
					} else if step.Transition != nil {
						enum := lifecycleEnum(agg)
						for _, value := range transitionValues(step.Transition) {
							if enum == nil || !enum.hasValue(value) {
								add(SevError, step.Pos, "'%s' is not a state of the Aggregate '%s'.", value, agg.Name)
							}
						}
					}
				}
			}
		}
		for _, coord := range bc.Application.Coordinations {
			for _, step := range coord.Steps {
				target := m.FindBC(step.BoundedContext)
				if target == nil {
					add(SevError, step.Pos, "Couldn't resolve reference to BoundedContext '%s'.", step.BoundedContext)
					continue
				}
				if target != bc && !m.reachable(bc.Name, target.Name) {
					add(SevError, step.Pos, "The Bounded Context '%s' is not reachable through Context Map relationships. Maybe you want to create a relationship with this context?", target.Name)
				}
				svc := target.applicationService(step.Service)
				if svc == nil {
					add(SevError, step.Pos, "The Service '%s' is not part of the Bounded Context '%s' Application.", step.Service, target.Name)
					continue
				}
				count := 0
				for _, op := range svc.Operations {
					if op.Name == step.Operation {
						count++
					}
				}
				if count == 0 {
					add(SevError, step.Pos, "The operation '%s' is not part of the Service '%s' of the Bounded Context '%s' Application.", step.Operation, step.Service, target.Name)
				} else if count > 1 {
					add(SevWarning, step.Pos, "The reference to operation '%s' is ambiguous, since there exist multiple operations with that name in Service '%s'. We suggest keeping the operation names distinct when referencing them through Coordinations.", step.Operation, step.Service)
				}
			}
		}
	}
}

func (m *Model) checkOperationStates(add adder) {
	for _, bc := range m.BoundedContexts {
		for _, agg := range bc.AllAggregates() {
			enum := lifecycleEnum(agg)
			check := func(ops []*Operation) {
				for _, op := range ops {
					if op.Transition == nil {
						continue
					}
					for _, value := range transitionValues(op.Transition) {
						if enum == nil || !enum.hasValue(value) {
							add(SevError, op.Pos, "'%s' is not a state of the Aggregate '%s'.", value, agg.Name)
						}
					}
				}
			}
			for _, obj := range agg.Objects {
				check(obj.Operations)
			}
			for _, svc := range agg.Services {
				check(svc.Operations)
			}
		}
	}
}

func (m *Model) checkReferences(add adder) {
	index := map[string][]*DomainObject{}
	owners := map[*DomainObject]*BoundedContext{}
	m.WalkObjects(func(bc *BoundedContext, _ *Aggregate, obj *DomainObject) {
		index[obj.Name] = append(index[obj.Name], obj)
		owners[obj] = bc
	})
	checkName := func(pos Position, name string, from *BoundedContext, ref bool) {
		if name == "" || primitiveTypes[name] {
			return
		}
		var matches []*DomainObject
		for _, obj := range index[name] {
			if ref && crossesContextAndDomain(from, owners[obj]) {
				continue
			}
			matches = append(matches, obj)
		}
		if len(matches) == 0 {
			if ref {
				add(SevError, pos, "Couldn't resolve reference to '%s'.", name)
			}
			return
		}
		if len(matches) > 1 {
			add(SevWarning, pos, "The reference to the type '%s' is ambiguous, since there exist multiple domain objects with that name in your model. We suggest to keep the names of domain objects distinct.", name)
		}
		targetBC := owners[matches[0]]
		if from == nil || targetBC == nil || targetBC == from {
			return
		}
		if !m.reachable(from.Name, targetBC.Name) {
			add(SevWarning, pos, "The object '%s' is not part of this Bounded Context and not reachable through Context Map relationships. Maybe you want to create a relationship with the context that contains that object?", name)
		}
	}
	m.WalkObjects(func(bc *BoundedContext, _ *Aggregate, obj *DomainObject) {
		for _, ref := range obj.References {
			checkName(ref.Pos, ref.Type.RefName(), bc, true)
		}
		for _, assoc := range obj.Associations {
			checkName(obj.Pos, assoc.Target, bc, true)
		}
		for _, attr := range obj.Attributes {
			if name, ok := attr.Type.LinkedName(); ok {
				checkName(attr.Pos, name, bc, true)
			}
			if hasMod(attr.Mods, "key") && attr.Type != nil && primitiveTypes[attr.Type.TypeName] {
				add(SevWarning, attr.Pos, "This attribute seems to be an identifier. Have you thought about creating a Value Object?")
			}
			if hasMod(attr.Mods, "nullable") && attr.Type != nil && primitiveTypes[attr.Type.TypeName] && !hasNot(attr.Mods, "nullable") {
				add(SevError, attr.Pos, "nullable is not relevant for primitive types")
			}
		}
		for _, tr := range obj.Traits {
			if len(index[tr.Name]) == 0 {
				add(SevError, obj.Pos, "Couldn't resolve reference to Trait '%s'.", tr.Name)
			}
		}
		if obj.ExtendsAt {
			checkName(obj.Pos, obj.Extends, bc, true)
		}
		for _, op := range obj.Operations {
			m.checkOpTypes(add, bc, op, checkName)
		}
		if obj.Repository != nil {
			for _, op := range obj.Repository.Operations {
				m.checkOpTypes(add, bc, op, checkName)
			}
		}
	})
	for _, bc := range m.BoundedContexts {
		for _, svc := range bc.allServices() {
			for _, op := range svc.Operations {
				m.checkOpTypes(add, bc, op, checkName)
			}
		}
	}
}

func (m *Model) checkOpTypes(add adder, bc *BoundedContext, op *Operation, checkName func(Position, string, *BoundedContext, bool)) {
	if name, ok := op.Return.LinkedName(); ok {
		checkName(op.Pos, name, bc, true)
	}
	for _, param := range op.Params {
		if name, ok := param.Type.LinkedName(); ok {
			checkName(op.Pos, name, bc, true)
		}
	}
}

func (m *Model) findRequirement(name string) Requirement {
	for _, req := range m.Requirements {
		if req.ReqName() == name {
			return req
		}
	}
	return nil
}

func (m *Model) reachable(from, to string) bool {
	if from == to {
		return true
	}
	if m.Map == nil {
		return false
	}
	graph := map[string][]string{}
	add := func(a, b string) {
		if a == "" || b == "" {
			return
		}
		graph[a] = append(graph[a], b)
		graph[b] = append(graph[b], a)
	}
	for _, rel := range m.Map.Relationships {
		left, right := rel.ends()
		add(left, right)
	}
	seen := map[string]bool{from: true}
	queue := []string{from}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, next := range graph[cur] {
			if next == to {
				return true
			}
			if !seen[next] {
				seen[next] = true
				queue = append(queue, next)
			}
		}
	}
	return false
}

func (bc *BoundedContext) allServices() []*Service {
	if bc == nil {
		return nil
	}
	out := slices.Clone(bc.Services)
	if bc.Application != nil {
		out = append(out, bc.Application.Services...)
	}
	for _, mod := range bc.Modules {
		out = append(out, mod.Services...)
		for _, agg := range mod.Aggregates {
			out = append(out, agg.Services...)
		}
	}
	for _, agg := range bc.Aggregates {
		out = append(out, agg.Services...)
	}
	return out
}

func (bc *BoundedContext) hasCommand(name string) bool {
	if bc.Application != nil {
		for _, cmd := range bc.Application.Commands {
			if cmd.Name == name {
				return true
			}
		}
	}
	for _, agg := range bc.AllAggregates() {
		for _, obj := range agg.Objects {
			if obj.Kind == KindCommand && obj.Name == name {
				return true
			}
		}
	}
	return false
}

func (bc *BoundedContext) hasEvent(name string) bool {
	if bc.Application != nil {
		for _, ev := range bc.Application.Events {
			if ev.Name == name {
				return true
			}
		}
	}
	for _, agg := range bc.AllAggregates() {
		for _, obj := range agg.Objects {
			if obj.Kind == KindEvent && obj.Name == name {
				return true
			}
		}
	}
	return false
}

func (bc *BoundedContext) hasOperation(name string) bool {
	for _, svc := range bc.allServices() {
		for _, op := range svc.Operations {
			if op.Name == name {
				return true
			}
		}
	}
	return false
}

func (bc *BoundedContext) applicationService(name string) *Service {
	if bc.Application == nil {
		return nil
	}
	for _, svc := range bc.Application.Services {
		if svc.Name == name {
			return svc
		}
	}
	return nil
}

func (bc *BoundedContext) ownsAggregate(agg *Aggregate) bool {
	for _, other := range bc.AllAggregates() {
		if other == agg {
			return true
		}
	}
	return false
}

func lifecycleEnum(agg *Aggregate) *DomainObject {
	for _, obj := range agg.Objects {
		if obj.Kind == KindEnum && obj.AggregateLifecycle {
			return obj
		}
	}
	return nil
}

func (o *DomainObject) hasValue(name string) bool {
	for _, v := range o.EnumValues {
		if v.Name == name {
			return true
		}
	}
	return false
}

func transitionValues(st *StateTransition) []string {
	var out []string
	out = append(out, st.From...)
	for _, to := range st.To {
		out = append(out, to.Value)
	}
	return out
}

func crossesContextAndDomain(from, target *BoundedContext) bool {
	return (from != nil && target == nil) || (from == nil && target != nil)
}

func hasMod(mods []Modifier, name string) bool {
	for _, mod := range mods {
		if mod.Name == name && !mod.Not {
			return true
		}
	}
	return false
}

func hasNot(mods []Modifier, name string) bool {
	for _, mod := range mods {
		if mod.Name == name && mod.Not {
			return true
		}
	}
	return false
}
