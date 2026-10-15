// Package refactor applies Context Mapper architectural refactorings to a CML model.
package refactor

import (
	"fmt"
	"slices"

	"github.com/remdev/go-cml/cml"
)

// Command is one architectural refactoring.
type Command struct {
	Name string
	Use  string
	Help string
	Min  int
	Run  func(*cml.Model, []string) error
}

// Commands returns the refactorings supported by the CLI.
func Commands() []Command {
	return []Command{
		{Name: "merge-bounded-contexts", Use: "<a> <b> [newName]", Min: 2, Help: "Объединить два ограниченных контекста.", Run: mergeBoundedContexts},
		{Name: "merge-aggregates", Use: "<a> <b> <newName>", Min: 3, Help: "Объединить два агрегата в один.", Run: mergeAggregates},
		{Name: "split-aggregate-by-entities", Use: "<aggregate> <newName> <entity...>", Min: 3, Help: "Вынести указанные сущности в новый агрегат.", Run: splitAggregate},
		{Name: "extract-shared-kernel", Use: "<bc1> <bc2> <newBc>", Min: 3, Help: "Заменить Shared Kernel новым вышестоящим контекстом.", Run: extractSharedKernel},
		{Name: "partnership-to-upstream-downstream", Use: "<upstream> <downstream>", Min: 2, Help: "Превратить Partnership в Upstream-Downstream.", Run: partnershipToUpstream},
		{Name: "suspend-partnership", Use: "<upstream> <downstream>", Min: 2, Help: "Приостановить Partnership, заменив его Upstream-Downstream.", Run: suspendPartnership},
		{Name: "partnership-to-shared-kernel", Use: "<bc1> <bc2>", Min: 2, Help: "Заменить Partnership на Shared Kernel.", Run: partnershipToShared},
		{Name: "shared-kernel-to-partnership", Use: "<bc1> <bc2>", Min: 2, Help: "Заменить Shared Kernel на Partnership.", Run: sharedToPartnership},
		{Name: "derive-bounded-contexts-from-subdomains", Use: "", Min: 0, Help: "Создать ограниченный контекст для каждого поддомена.", Run: deriveFromSubdomains},
		{Name: "split-bounded-context-by-owner", Use: "<boundedContext>", Min: 1, Help: "Разнести агрегаты по контекстам их владельцев.", Run: splitByOwner},
	}
}

// Apply runs a refactoring by name.
func Apply(model *cml.Model, name string, args []string) error {
	for _, cmd := range Commands() {
		if cmd.Name != name {
			continue
		}
		if len(args) < cmd.Min {
			return fmt.Errorf("usage: cm refactor -r %s %s", cmd.Name, cmd.Use)
		}
		return cmd.Run(model, args)
	}
	return fmt.Errorf("unknown refactoring %q", name)
}

func mergeBoundedContexts(model *cml.Model, args []string) error {
	a := model.FindBC(args[0])
	b := model.FindBC(args[1])
	if a == nil || b == nil {
		return fmt.Errorf("both bounded contexts must exist")
	}
	if a == b {
		return fmt.Errorf("cannot merge a bounded context with itself")
	}
	targetName := a.Name
	if len(args) > 2 && args[2] != "" {
		targetName = args[2]
	}
	target := a
	var remove []string
	if targetName != a.Name && targetName != b.Name {
		if model.FindBC(targetName) != nil {
			return fmt.Errorf("bounded context %s already exists", targetName)
		}
		target = &cml.BoundedContext{Name: targetName, Type: a.Type, Implements: slices.Clone(a.Implements)}
		model.BoundedContexts = append(model.BoundedContexts, target)
		remove = []string{a.Name, b.Name}
	} else if targetName == b.Name {
		target = b
		remove = []string{a.Name}
	} else {
		remove = []string{b.Name}
	}
	for _, src := range []*cml.BoundedContext{a, b} {
		if src == target {
			continue
		}
		absorbBC(target, src)
	}
	for _, name := range remove {
		dropBC(model, name)
		rewrite(model, name, target.Name)
	}
	return nil
}

func absorbBC(dst, src *cml.BoundedContext) {
	for _, agg := range src.Aggregates {
		if _, _, existing := findAgg(dst, agg.Name); existing != nil {
			agg.Name = src.Name + "_" + agg.Name
		}
		dst.Aggregates = append(dst.Aggregates, agg)
	}
	dst.Modules = append(dst.Modules, src.Modules...)
	dst.Services = append(dst.Services, src.Services...)
	dst.Implements = unique(append(dst.Implements, src.Implements...))
	dst.Realizes = unique(append(dst.Realizes, src.Realizes...))
	if src.Application == nil {
		return
	}
	if dst.Application == nil {
		dst.Application = src.Application
		return
	}
	dst.Application.Commands = append(dst.Application.Commands, src.Application.Commands...)
	dst.Application.Events = append(dst.Application.Events, src.Application.Events...)
	dst.Application.Services = append(dst.Application.Services, src.Application.Services...)
	dst.Application.Flows = append(dst.Application.Flows, src.Application.Flows...)
	dst.Application.Coordinations = append(dst.Application.Coordinations, src.Application.Coordinations...)
}

func mergeAggregates(model *cml.Model, args []string) error {
	_, _, a := model.FindAggregate(args[0])
	_, _, b := model.FindAggregate(args[1])
	if a == nil || b == nil {
		return fmt.Errorf("both aggregates must exist")
	}
	if a == b {
		return fmt.Errorf("cannot merge an aggregate with itself")
	}
	owner, module, _ := model.FindAggregate(args[0])
	created := &cml.Aggregate{Name: args[2], Objects: append(append([]*cml.DomainObject{}, a.Objects...), b.Objects...), Services: append(append([]*cml.Service{}, a.Services...), b.Services...)}
	detach(owner, module, a.Name)
	owner, module, _ = model.FindAggregate(args[1])
	detach(owner, module, b.Name)
	if module != nil {
		module.Aggregates = append(module.Aggregates, created)
	} else if owner != nil {
		owner.Aggregates = append(owner.Aggregates, created)
	}
	for _, rel := range mapRels(model) {
		for i, name := range rel.ExposedAggregates {
			if name == a.Name || name == b.Name {
				rel.ExposedAggregates[i] = created.Name
			}
		}
	}
	return nil
}

func splitAggregate(model *cml.Model, args []string) error {
	owner, module, agg := model.FindAggregate(args[0])
	if agg == nil {
		return fmt.Errorf("aggregate %s not found", args[0])
	}
	if _, _, existing := model.FindAggregate(args[1]); existing != nil {
		return fmt.Errorf("aggregate %s already exists", args[1])
	}
	next := &cml.Aggregate{Name: args[1]}
	wanted := map[string]bool{}
	for _, name := range args[2:] {
		wanted[name] = true
	}
	var stay, move []*cml.DomainObject
	for _, obj := range agg.Objects {
		if wanted[obj.Name] {
			move = append(move, obj)
			delete(wanted, obj.Name)
		} else {
			stay = append(stay, obj)
		}
	}
	if len(wanted) > 0 {
		return fmt.Errorf("entity %s is not in aggregate %s", firstKey(wanted), agg.Name)
	}
	agg.Objects = stay
	next.Objects = move
	if module != nil {
		module.Aggregates = append(module.Aggregates, next)
	} else {
		owner.Aggregates = append(owner.Aggregates, next)
	}
	return nil
}

func extractSharedKernel(model *cml.Model, args []string) error {
	if model.Map == nil {
		return fmt.Errorf("a context map is required")
	}
	bc1, bc2, newName := args[0], args[1], args[2]
	if model.FindBC(bc1) == nil || model.FindBC(bc2) == nil {
		return fmt.Errorf("both participants must exist")
	}
	if model.FindBC(newName) != nil {
		return fmt.Errorf("bounded context %s already exists", newName)
	}
	found := false
	var kept []*cml.Relationship
	for _, rel := range model.Map.Relationships {
		if rel.Kind == cml.RelSharedKernel && samePair(rel.Participant1, rel.Participant2, bc1, bc2) {
			found = true
			continue
		}
		kept = append(kept, rel)
	}
	if !found {
		return fmt.Errorf("no shared kernel between %s and %s", bc1, bc2)
	}
	model.BoundedContexts = append(model.BoundedContexts, &cml.BoundedContext{Name: newName, Type: "FEATURE"})
	model.Map.Contains = unique(append(model.Map.Contains, newName))
	model.Map.Relationships = append(kept,
		upstream(newName, bc1),
		upstream(newName, bc2),
	)
	return nil
}

func partnershipToUpstream(model *cml.Model, args []string) error {
	return convertPartnership(model, args[0], args[1], false)
}

func suspendPartnership(model *cml.Model, args []string) error {
	return convertPartnership(model, args[0], args[1], true)
}

func convertPartnership(model *cml.Model, upstream, downstream string, suspend bool) error {
	rel := findSymmetric(model, cml.RelPartnership, upstream, downstream)
	if rel == nil {
		return fmt.Errorf("no partnership between %s and %s", upstream, downstream)
	}
	rel.Kind = cml.RelUpstreamDownstream
	rel.Upstream = upstream
	rel.Downstream = downstream
	rel.Participant1, rel.Participant2 = "", ""
	if suspend && rel.ImplementationTechnology == "" {
		rel.ImplementationTechnology = "suspended partnership"
	}
	return nil
}

func partnershipToShared(model *cml.Model, args []string) error {
	rel := findSymmetric(model, cml.RelPartnership, args[0], args[1])
	if rel == nil {
		return fmt.Errorf("no partnership between %s and %s", args[0], args[1])
	}
	rel.Kind = cml.RelSharedKernel
	return nil
}

func sharedToPartnership(model *cml.Model, args []string) error {
	rel := findSymmetric(model, cml.RelSharedKernel, args[0], args[1])
	if rel == nil {
		return fmt.Errorf("no shared kernel between %s and %s", args[0], args[1])
	}
	rel.Kind = cml.RelPartnership
	return nil
}

func deriveFromSubdomains(model *cml.Model, _ []string) error {
	created := 0
	for _, domain := range model.Domains {
		for _, sub := range domain.Subdomains {
			if subdomainImplemented(model, sub.Name) {
				continue
			}
			name := sub.Name + "Context"
			if model.FindBC(name) != nil {
				name = sub.Name + "BoundedContext"
			}
			model.BoundedContexts = append(model.BoundedContexts, &cml.BoundedContext{
				Name:       name,
				Type:       "FEATURE",
				Implements: []string{sub.Name},
			})
			if model.Map != nil {
				model.Map.Contains = append(model.Map.Contains, name)
			}
			created++
		}
	}
	if created == 0 {
		return fmt.Errorf("every subdomain is already implemented by a bounded context")
	}
	return nil
}

func splitByOwner(model *cml.Model, args []string) error {
	bc := model.FindBC(args[0])
	if bc == nil {
		return fmt.Errorf("bounded context %s not found", args[0])
	}
	groups := map[string][]*cml.Aggregate{}
	for _, agg := range bc.AllAggregates() {
		if agg.Owner == "" {
			continue
		}
		groups[agg.Owner] = append(groups[agg.Owner], agg)
	}
	if len(groups) == 0 {
		return fmt.Errorf("no aggregate in %s has an owner", bc.Name)
	}
	for owner, aggs := range groups {
		for _, agg := range aggs {
			bc.DetachAggregate(agg.Name)
		}
		name := bc.Name + "_" + owner
		model.BoundedContexts = append(model.BoundedContexts, &cml.BoundedContext{
			Name:       name,
			Type:       "FEATURE",
			Aggregates: aggs,
		})
		if model.Map != nil {
			model.Map.Contains = unique(append(model.Map.Contains, name))
		}
		if team := model.FindBC(owner); team != nil && team.Type == "TEAM" && !slices.Contains(team.Realizes, name) {
			team.Realizes = append(team.Realizes, name)
		}
	}
	return nil
}

func upstream(up, down string) *cml.Relationship {
	return &cml.Relationship{
		Kind:            cml.RelUpstreamDownstream,
		Upstream:        up,
		Downstream:      down,
		UpstreamRoles:   []string{"OHS", "PL"},
		DownstreamRoles: []string{"CF"},
	}
}

func findSymmetric(model *cml.Model, kind, a, b string) *cml.Relationship {
	if model.Map == nil {
		return nil
	}
	for _, rel := range model.Map.Relationships {
		if rel.Kind == kind && samePair(rel.Participant1, rel.Participant2, a, b) {
			return rel
		}
	}
	return nil
}

func samePair(left, right, a, b string) bool {
	return (left == a && right == b) || (left == b && right == a)
}

func subdomainImplemented(model *cml.Model, name string) bool {
	for _, bc := range model.BoundedContexts {
		if slices.Contains(bc.Implements, name) {
			return true
		}
	}
	return false
}

func dropBC(model *cml.Model, name string) {
	var kept []*cml.BoundedContext
	for _, bc := range model.BoundedContexts {
		if bc.Name != name {
			kept = append(kept, bc)
		}
	}
	model.BoundedContexts = kept
}

func rewrite(model *cml.Model, from, to string) {
	swap := func(name string) string {
		if name == from {
			return to
		}
		return name
	}
	if model.Map != nil {
		for i, name := range model.Map.Contains {
			model.Map.Contains[i] = swap(name)
		}
		model.Map.Contains = unique(model.Map.Contains)
		var rels []*cml.Relationship
		for _, rel := range model.Map.Relationships {
			rel.Participant1 = swap(rel.Participant1)
			rel.Participant2 = swap(rel.Participant2)
			rel.Upstream = swap(rel.Upstream)
			rel.Downstream = swap(rel.Downstream)
			if rel.Participant1 != "" && rel.Participant1 == rel.Participant2 {
				continue
			}
			if rel.Upstream != "" && rel.Upstream == rel.Downstream {
				continue
			}
			rels = append(rels, rel)
		}
		model.Map.Relationships = rels
	}
	for _, bc := range model.BoundedContexts {
		bc.Refines = swap(bc.Refines)
		for i, name := range bc.Realizes {
			bc.Realizes[i] = swap(name)
		}
		bc.Realizes = unique(bc.Realizes)
		for _, agg := range bc.AllAggregates() {
			agg.Owner = swap(agg.Owner)
		}
		if bc.Application == nil {
			continue
		}
		for _, coord := range bc.Application.Coordinations {
			for i := range coord.Steps {
				coord.Steps[i].BoundedContext = swap(coord.Steps[i].BoundedContext)
			}
		}
	}
}

func detach(owner *cml.BoundedContext, module *cml.Module, name string) {
	if module != nil {
		var kept []*cml.Aggregate
		for _, agg := range module.Aggregates {
			if agg.Name != name {
				kept = append(kept, agg)
			}
		}
		module.Aggregates = kept
		return
	}
	if owner != nil {
		owner.DetachAggregate(name)
	}
}

func findAgg(bc *cml.BoundedContext, name string) (*cml.BoundedContext, *cml.Module, *cml.Aggregate) {
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
	return bc, nil, nil
}

func mapRels(model *cml.Model) []*cml.Relationship {
	if model.Map == nil {
		return nil
	}
	return model.Map.Relationships
}

func unique(items []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, item := range items {
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

func firstKey(items map[string]bool) string {
	for key := range items {
		return key
	}
	return ""
}
