package cml

import (
	"regexp"
	"slices"
)

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
	add := func(code RuleCode, pos Position, args ...any) {
		ds = append(ds, makeDiagnostic(code, pos, args...))
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

type adder func(RuleCode, Position, ...any)

func (m *Model) checkUnique(add adder) {
	countBC := map[string]int{}
	for _, bc := range m.BoundedContexts {
		countBC[bc.Name]++
	}
	for _, bc := range m.BoundedContexts {
		if countBC[bc.Name] > 1 {
			add(RuleDuplicateBoundedContext, bc.Pos, bc.Name)
		}
		names := map[string]bool{}
		for _, svc := range bc.allServices() {
			if names[svc.Name] {
				add(RuleDuplicateContextService, svc.Pos, svc.Name)
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
			add(RuleDuplicateDomain, d.Pos, d.Name)
		}
		for _, sd := range d.Subdomains {
			if countSub[sd.Name] > 1 {
				add(RuleDuplicateSubdomain, sd.Pos, sd.Name)
			}
			svcNames := map[string]bool{}
			for _, svc := range sd.Services {
				if svcNames[svc.Name] {
					add(RuleDuplicateSubdomainService, svc.Pos, svc.Name)
				}
				svcNames[svc.Name] = true
			}
			seen := map[string]bool{}
			for _, obj := range sd.Entities {
				if obj.Name == "" {
					continue
				}
				if seen[obj.Name] {
					add(RuleDuplicateDomainObject, obj.Pos, obj.Name)
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
			add(RuleDuplicateModule, mod.Pos, mod.Name)
		}
		seen := map[string]bool{}
		for _, obj := range mod.Objects {
			if seen[obj.Name] {
				add(RuleDuplicateDomainObject, obj.Pos, obj.Name)
			}
			seen[obj.Name] = true
		}
		if len(mod.Services) > 0 || hasEntity(mod.Objects) {
			add(RuleUngroupedModuleMembers, mod.Pos, mod.Name)
		}
	}
	for _, agg := range aggs {
		if countAgg[agg.Name] > 1 {
			add(RuleDuplicateAggregate, agg.Pos, agg.Name)
		}
		seen := map[string]bool{}
		for _, obj := range agg.Objects {
			if obj.Name == "" {
				continue
			}
			if seen[obj.Name] {
				add(RuleDuplicateDomainObject, obj.Pos, obj.Name)
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
			add(RuleDuplicateRequirement, req.ReqPos(), req.ReqName())
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
			add(RuleDuplicateFlow, f.pos, f.name)
		}
	}
	for _, c := range coords {
		if countCoord[c.name] > 1 {
			add(RuleDuplicateCoordination, c.pos, c.name)
		}
	}
	objCount := map[string]int{}
	m.WalkObjects(func(_ *BoundedContext, _ *Aggregate, obj *DomainObject) {
		objCount[obj.Name]++
	})
	m.WalkObjects(func(_ *BoundedContext, _ *Aggregate, obj *DomainObject) {
		if objCount[obj.Name] > 1 {
			add(RuleReusedDomainObjectName, obj.Pos, obj.Name)
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
			add(RuleUnknownBoundedContext, m.Map.Pos, name)
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
		add(RuleOrganizationalMapWithoutTeams, m.Map.Pos)
	}
	if m.Map.Type == "SYSTEM_LANDSCAPE" {
		for _, name := range m.Map.Contains {
			bc := m.FindBC(name)
			if bc != nil && bc.Type == "TEAM" {
				add(RuleTeamOnSystemLandscape, m.Map.Pos)
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
				add(RuleUnknownBoundedContext, rel.Pos, name)
				continue
			}
			if !contains[name] {
				add(RuleContextNotInMap, rel.Pos, name)
			}
		}
		if left != "" && left == right {
			add(RuleSelfRelationship, rel.Pos)
		}
		if rel.Kind == RelCustomerSupplier {
			if slices.Contains(rel.UpstreamRoles, "OHS") {
				add(RuleCustomerSupplierOHS, rel.Pos)
			}
			if slices.Contains(rel.DownstreamRoles, "ACL") {
				add(RuleCustomerSupplierACL, rel.Pos)
			}
			if slices.Contains(rel.DownstreamRoles, "CF") {
				add(RuleCustomerSupplierConformist, rel.Pos)
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
					add(RuleExposedAggregateOwnership, rel.Pos, name, rel.Upstream)
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
			add(RuleRealizesRequiresTeam, bc.Pos, bc.Name)
		}
		for _, name := range bc.Realizes {
			if m.FindBC(name) == nil {
				add(RuleUnknownBoundedContext, bc.Pos, name)
			}
		}
		if bc.Refines != "" && m.FindBC(bc.Refines) == nil {
			add(RuleUnknownBoundedContext, bc.Pos, bc.Refines)
		}
		domains := 0
		for _, part := range bc.Implements {
			domain := m.FindDomain(part)
			_, sub := m.FindSubdomain(part)
			if domain == nil && sub == nil {
				add(RuleUnknownDomain, bc.Pos, part)
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
					add(RuleRedundantDomainImplementation, bc.Pos, sub.Name, parent.Name)
				}
			}
		}
		if domains > 1 {
			add(RuleMultipleImplementedDomains, bc.Pos)
		}
	}
}

func (m *Model) checkDomains(add adder) {
	for _, d := range m.Domains {
		for _, sd := range d.Subdomains {
			if sd.Name == d.Name {
				add(RuleSubdomainMatchesDomain, sd.Pos, sd.Name)
			}
			for _, name := range sd.Supports {
				if m.findRequirement(name) == nil {
					add(RuleUnknownRequirement, sd.Pos, name)
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
					add(RuleMultipleAggregateRoots, obj.Pos, agg.Name)
				}
			}
			if len(states) > 1 {
				for _, obj := range states {
					add(RuleMultipleLifecycleEnums, obj.Pos, agg.Name)
				}
			}
			if agg.Owner != "" {
				owner := m.FindBC(agg.Owner)
				if owner == nil {
					add(RuleUnknownBoundedContext, agg.Pos, agg.Owner)
				} else if owner.Type != "TEAM" {
					add(RuleOwnerRequiresTeam, agg.Pos, owner.Name)
				}
			}
			for _, name := range agg.RequirementRefs {
				if m.findRequirement(name) == nil {
					add(RuleUnknownRequirement, agg.Pos, name)
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
				add(RuleFeatureEntityIdentifier, feature.Pos)
			}
			if feature.Verb != "" && !idPattern.MatchString(feature.Verb) {
				add(RuleFeatureVerbIdentifier, feature.Pos)
			}
			if story != nil && story.SplitBy == "" && len(story.Features) == 1 && feature.Verb != "" {
				add(RuleStorySplitSuggestion, feature.Pos)
			}
		}
		if story != nil && story.SplitBy != "" && m.findRequirement(story.SplitBy) == nil {
			add(RuleUnknownUserStory, story.Pos, story.SplitBy)
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
					add(RuleFlowMemberScope, step.Pos, step.Command, bc.Name)
				}
				if step.Operation != "" && !bc.hasOperation(step.Operation) {
					add(RuleFlowMemberScope, step.Pos, step.Operation, bc.Name)
				}
				for _, name := range step.Commands {
					if !bc.hasCommand(name) {
						add(RuleFlowMemberScope, step.Pos, name, bc.Name)
					}
				}
				for _, name := range step.Operations {
					if !bc.hasOperation(name) {
						add(RuleFlowMemberScope, step.Pos, name, bc.Name)
					}
				}
				for _, name := range step.Events {
					if !bc.hasEvent(name) {
						add(RuleUnknownDomainEvent, step.Pos, name)
					}
				}
				if step.Aggregate != "" {
					_, _, agg := m.FindAggregate(step.Aggregate)
					if agg == nil || !bc.ownsAggregate(agg) {
						add(RuleUnknownAggregate, step.Pos, step.Aggregate)
					} else if step.Transition != nil {
						enum := lifecycleEnum(agg)
						for _, value := range transitionValues(step.Transition) {
							if enum == nil || !enum.hasValue(value) {
								add(RuleUnknownAggregateState, step.Pos, value, agg.Name)
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
					add(RuleUnknownBoundedContext, step.Pos, step.BoundedContext)
					continue
				}
				if target != bc && !m.reachable(bc.Name, target.Name) {
					add(RuleCoordinationReachability, step.Pos, target.Name)
				}
				svc := target.applicationService(step.Service)
				if svc == nil {
					add(RuleUnknownApplicationService, step.Pos, step.Service, target.Name)
					continue
				}
				count := 0
				for _, op := range svc.Operations {
					if op.Name == step.Operation {
						count++
					}
				}
				if count == 0 {
					add(RuleUnknownApplicationOperation, step.Pos, step.Operation, step.Service, target.Name)
				} else if count > 1 {
					add(RuleAmbiguousApplicationOperation, step.Pos, step.Operation, step.Service)
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
							add(RuleUnknownAggregateState, op.Pos, value, agg.Name)
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
				add(RuleUnknownDomainObject, pos, name)
			}
			return
		}
		if len(matches) > 1 {
			add(RuleAmbiguousDomainObject, pos, name)
		}
		targetBC := owners[matches[0]]
		if from == nil || targetBC == nil || targetBC == from {
			return
		}
		if !m.reachable(from.Name, targetBC.Name) {
			add(RuleObjectReachability, pos, name)
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
				add(RulePrimitiveIdentifier, attr.Pos)
			}
			if hasMod(attr.Mods, "nullable") && attr.Type != nil && primitiveTypes[attr.Type.TypeName] && !hasNot(attr.Mods, "nullable") {
				add(RuleNullablePrimitive, attr.Pos)
			}
		}
		for _, tr := range obj.Traits {
			if len(index[tr.Name]) == 0 {
				add(RuleUnknownTrait, obj.Pos, tr.Name)
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
