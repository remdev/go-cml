package cml

// TemplateData exposes the model to the generic text generator.
// Property names follow the CML semantic model: boundedContexts, map, domains,
// userRequirements, stakeholders and valueRegisters.
func (m *Model) TemplateData() map[string]any {
	if m == nil {
		return map[string]any{}
	}
	bcs := make([]any, 0, len(m.BoundedContexts))
	for _, bc := range m.BoundedContexts {
		bcs = append(bcs, bc.templateData())
	}
	domains := make([]any, 0, len(m.Domains))
	for _, d := range m.Domains {
		subs := make([]any, 0, len(d.Subdomains))
		for _, sd := range d.Subdomains {
			subs = append(subs, map[string]any{
				"name":                  sd.Name,
				"type":                  sd.Type,
				"domainVisionStatement": sd.DomainVisionStatement,
				"supports":              anyStrings(sd.Supports),
			})
		}
		domains = append(domains, map[string]any{
			"name":                  d.Name,
			"domainVisionStatement": d.DomainVisionStatement,
			"subdomains":            subs,
		})
	}
	reqs := make([]any, 0, len(m.Requirements))
	for _, req := range m.Requirements {
		kind := "UseCase"
		role := ""
		benefit := ""
		if story, ok := req.(*UserStory); ok {
			kind = "UserStory"
			role = story.Role
			benefit = story.Benefit
		}
		if uc, ok := req.(*UseCase); ok {
			role = uc.Actor
			benefit = uc.Benefit
		}
		features := make([]any, 0, len(req.ReqFeatures()))
		for _, f := range req.ReqFeatures() {
			features = append(features, map[string]any{
				"verb":   f.Verb,
				"entity": f.Entity,
			})
		}
		reqs = append(reqs, map[string]any{
			"name":     req.ReqName(),
			"kind":     kind,
			"role":     role,
			"benefit":  benefit,
			"features": features,
		})
	}
	var contextMap any
	if m.Map != nil {
		rels := make([]any, 0, len(m.Map.Relationships))
		for _, rel := range m.Map.Relationships {
			rels = append(rels, map[string]any{
				"kind":            rel.Kind,
				"name":            rel.Name,
				"participant1":    rel.Participant1,
				"participant2":    rel.Participant2,
				"upstream":        rel.Upstream,
				"downstream":      rel.Downstream,
				"upstreamRoles":   anyStrings(rel.UpstreamRoles),
				"downstreamRoles": anyStrings(rel.DownstreamRoles),
				"technology":      rel.ImplementationTechnology,
			})
		}
		contextMap = map[string]any{
			"name":          m.Map.Name,
			"type":          m.Map.Type,
			"state":         m.Map.State,
			"contains":      anyStrings(m.Map.Contains),
			"relationships": rels,
		}
	}
	return map[string]any{
		"boundedContexts":  bcs,
		"domains":          domains,
		"userRequirements": reqs,
		"map":              contextMap,
		"stakeholders":     len(m.StakeholderBlocks),
		"valueRegisters":   len(m.ValueRegisters),
	}
}

func (bc *BoundedContext) templateData() map[string]any {
	aggs := make([]any, 0)
	for _, agg := range bc.AllAggregates() {
		objs := make([]any, 0, len(agg.Objects))
		for _, obj := range agg.Objects {
			attrs := make([]any, 0, len(obj.Attributes))
			for _, attr := range obj.Attributes {
				typeName := ""
				if attr.Type != nil {
					typeName = attr.Type.String()
				}
				attrs = append(attrs, map[string]any{"name": attr.Name, "type": typeName})
			}
			refs := make([]any, 0, len(obj.References))
			for _, ref := range obj.References {
				refs = append(refs, map[string]any{"name": ref.Name, "type": ref.Type.RefName()})
			}
			objs = append(objs, map[string]any{
				"name":       obj.Name,
				"kind":       obj.Kind,
				"attributes": attrs,
				"references": refs,
			})
		}
		aggs = append(aggs, map[string]any{
			"name":          agg.Name,
			"owner":         agg.Owner,
			"domainObjects": objs,
		})
	}
	return map[string]any{
		"name":                     bc.Name,
		"type":                     bc.Type,
		"domainVisionStatement":    bc.DomainVisionStatement,
		"responsibilities":         anyStrings(bc.Responsibilities),
		"implementationTechnology": bc.ImplementationTechnology,
		"implements":               anyStrings(bc.Implements),
		"realizes":                 anyStrings(bc.Realizes),
		"aggregates":               aggs,
	}
}

func anyStrings(items []string) []any {
	if items == nil {
		items = []string{}
	}
	out := make([]any, len(items))
	for i, item := range items {
		out[i] = item
	}
	return out
}
