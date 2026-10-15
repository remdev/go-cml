package gen

import (
	"fmt"
	"strings"

	"github.com/remdev/go-cml/cml"
)

// File is one generated artifact.
type File struct {
	Name string
	Body string
}

// PlantUML builds the class, component, state and use-case diagrams of a model.
func PlantUML(model *cml.Model, base string) ([]File, error) {
	if model == nil {
		return nil, fmt.Errorf("no CML model")
	}
	if model.Map == nil && len(model.BoundedContexts) == 0 && len(model.Requirements) == 0 && !hasSubdomainEntities(model) {
		return nil, fmt.Errorf("the model has no context map, bounded contexts, subdomains or user requirements to draw")
	}
	var files []File
	if model.Map != nil {
		files = append(files, File{Name: base + "_ContextMap.puml", Body: componentDiagram(model.Map)})
	}
	for _, bc := range model.BoundedContexts {
		files = append(files, File{Name: base + "_BC_" + bc.Name + ".puml", Body: classDiagram(bc.Name, bc.AllAggregates())})
		for _, agg := range bc.AllAggregates() {
			files = append(files, File{Name: base + "_BC_" + bc.Name + "_" + agg.Name + ".puml", Body: classDiagram(agg.Name, []*cml.Aggregate{agg})})
			if diagram := aggregateState(agg); diagram != "" {
				files = append(files, File{Name: base + "_BC_" + bc.Name + "_" + agg.Name + "_StateDiagram.puml", Body: diagram})
			}
		}
		for _, mod := range bc.Modules {
			files = append(files, File{Name: base + "_BC_" + bc.Name + "_" + mod.Name + ".puml", Body: classDiagram(mod.Name, mod.Aggregates)})
		}
		if bc.Application != nil {
			for _, flow := range bc.Application.Flows {
				if diagram := flowState(flow); diagram != "" {
					files = append(files, File{Name: base + "_BC_" + bc.Name + "_" + flow.Name + "_StateDiagram.puml", Body: diagram})
				}
			}
		}
	}
	for _, domain := range model.Domains {
		for _, sub := range domain.Subdomains {
			if len(sub.Entities) == 0 {
				continue
			}
			files = append(files, File{Name: base + "_SD_" + sub.Name + ".puml", Body: objectDiagram(domain.Name+"."+sub.Name, sub.Entities)})
		}
	}
	if len(model.Requirements) > 0 {
		files = append(files, File{Name: base + "_UseCases.puml", Body: useCaseDiagram(model)})
	}
	return files, nil
}

func hasSubdomainEntities(model *cml.Model) bool {
	for _, domain := range model.Domains {
		for _, sub := range domain.Subdomains {
			if len(sub.Entities) > 0 {
				return true
			}
		}
	}
	return false
}

func componentDiagram(cm *cml.ContextMap) string {
	var b strings.Builder
	b.WriteString("@startuml\nskinparam componentStyle rectangle\ntitle " + safe(cm.Name) + "\n")
	names := append([]string{}, cm.Contains...)
	for _, rel := range cm.Relationships {
		left, right := relEnds(rel)
		names = append(names, left, right)
	}
	seen := map[string]bool{}
	for _, name := range names {
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		fmt.Fprintf(&b, "component %q as %s\n", name, id(name))
	}
	for _, rel := range cm.Relationships {
		left, right := relEnds(rel)
		arrow := "-->"
		label := strings.Join(rel.UpstreamRoles, ",")
		switch rel.Kind {
		case cml.RelPartnership:
			arrow = "<->"
			label = "P"
		case cml.RelSharedKernel:
			arrow = "<->"
			label = "SK"
		case cml.RelCustomerSupplier:
			label = "S " + strings.Join(rel.UpstreamRoles, ",") + " / C " + strings.Join(rel.DownstreamRoles, ",")
		default:
			if label == "" {
				label = "U"
			}
			if len(rel.DownstreamRoles) > 0 {
				label += " / " + strings.Join(rel.DownstreamRoles, ",")
			}
		}
		if rel.Name != "" {
			label = rel.Name + " " + label
		}
		fmt.Fprintf(&b, "%s %s %s : %s\n", id(left), arrow, id(right), strings.TrimSpace(label))
	}
	b.WriteString("@enduml\n")
	return b.String()
}

func classDiagram(title string, aggs []*cml.Aggregate) string {
	var objs []*cml.DomainObject
	for _, agg := range aggs {
		objs = append(objs, agg.Objects...)
	}
	return objectDiagram(title, objs)
}

func objectDiagram(title string, objs []*cml.DomainObject) string {
	var b strings.Builder
	fmt.Fprintf(&b, "@startuml\ntitle %s\n", safe(title))
	for _, obj := range objs {
		stereo := obj.Kind
		if obj.AggregateRoot {
			stereo += ", Aggregate Root"
		}
		fmt.Fprintf(&b, "class %s <<%s>> {\n", id(obj.Name), stereo)
		for _, attr := range obj.Attributes {
			typeName := ""
			if attr.Type != nil {
				typeName = attr.Type.String()
			}
			fmt.Fprintf(&b, "  %s %s\n", typeName, attr.Name)
		}
		for _, ref := range obj.References {
			fmt.Fprintf(&b, "  %s %s\n", ref.Type.RefName(), ref.Name)
		}
		b.WriteString("}\n")
	}
	for _, obj := range objs {
		for _, ref := range obj.References {
			target := ref.Type.RefName()
			if target == "" {
				continue
			}
			fmt.Fprintf(&b, "%s --> %s : %s\n", id(obj.Name), id(target), ref.Name)
		}
	}
	b.WriteString("@enduml\n")
	return b.String()
}

func aggregateState(agg *cml.Aggregate) string {
	enum := lifecycle(agg)
	var lines []string
	for _, obj := range agg.Objects {
		for _, op := range obj.Operations {
			if op.Transition != nil {
				lines = append(lines, transitionLines(op.Name, op.Transition)...)
			}
		}
	}
	if enum == nil || len(lines) == 0 {
		return ""
	}
	return stateDiagram(agg.Name, enum, lines)
}

func flowState(flow *cml.Flow) string {
	var lines []string
	for _, step := range flow.Steps {
		if step.Transition == nil {
			continue
		}
		label := step.Command
		if label == "" {
			label = step.Operation
		}
		lines = append(lines, transitionLines(label, step.Transition)...)
	}
	if len(lines) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "@startuml\ntitle %s\n", safe(flow.Name))
	for _, line := range lines {
		b.WriteString(line + "\n")
	}
	b.WriteString("@enduml\n")
	return b.String()
}

func stateDiagram(title string, enum *cml.DomainObject, lines []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "@startuml\ntitle %s\n", safe(title))
	for _, value := range enum.EnumValues {
		fmt.Fprintf(&b, "state %s\n", id(value.Name))
	}
	for _, line := range lines {
		b.WriteString(line + "\n")
	}
	b.WriteString("@enduml\n")
	return b.String()
}

func transitionLines(label string, tr *cml.StateTransition) []string {
	var lines []string
	froms := tr.From
	if len(froms) == 0 {
		froms = []string{"[*]"}
	}
	for _, from := range froms {
		for _, to := range tr.To {
			lines = append(lines, fmt.Sprintf("%s --> %s : %s", id(from), id(to.Value), safe(label)))
		}
	}
	return lines
}

func lifecycle(agg *cml.Aggregate) *cml.DomainObject {
	for _, obj := range agg.Objects {
		if obj.Kind == cml.KindEnum && obj.AggregateLifecycle {
			return obj
		}
	}
	return nil
}

func useCaseDiagram(model *cml.Model) string {
	var b strings.Builder
	b.WriteString("@startuml\nleft to right direction\n")
	for _, req := range model.Requirements {
		actor := "User"
		if uc, ok := req.(*cml.UseCase); ok && uc.Actor != "" {
			actor = uc.Actor
		}
		if story, ok := req.(*cml.UserStory); ok && story.Role != "" {
			actor = story.Role
		}
		fmt.Fprintf(&b, "actor %q\n", actor)
		fmt.Fprintf(&b, "usecase %q as %s\n", req.ReqName(), id(req.ReqName()))
		fmt.Fprintf(&b, "%q --> %s\n", actor, id(req.ReqName()))
	}
	b.WriteString("@enduml\n")
	return b.String()
}

// ContextMapDOT renders a Graphviz context map.
func ContextMapDOT(model *cml.Model) (string, error) {
	if model == nil || model.Map == nil {
		return "", fmt.Errorf("the model does not contain a Context Map")
	}
	cm := model.Map
	var b strings.Builder
	name := cm.Name
	if name == "" {
		name = "ContextMap"
	}
	fmt.Fprintf(&b, "digraph %s {\n", id(name))
	b.WriteString("  rankdir=LR;\n  node [shape=box, style=\"rounded,filled\", fillcolor=\"#F7F4EF\", fontname=\"Helvetica\"];\n  edge [fontname=\"Helvetica\"];\n")
	teams := map[string]bool{}
	for _, bcName := range cm.Contains {
		if bc := model.FindBC(bcName); bc != nil && bc.Type == "TEAM" {
			teams[bcName] = true
		}
	}
	writeNodes := func(wantTeam bool) {
		for _, bcName := range cm.Contains {
			if teams[bcName] != wantTeam {
				continue
			}
			label := bcName
			if bc := model.FindBC(bcName); bc != nil && bc.Type != "" {
				label = bcName + "\\n" + bc.Type
			}
			fmt.Fprintf(&b, "    %s [label=\"%s\"];\n", id(bcName), label)
		}
	}
	if len(teams) > 0 && len(teams) != len(cm.Contains) {
		b.WriteString("  subgraph cluster_systems {\n    label=\"Bounded Contexts\";\n")
		writeNodes(false)
		b.WriteString("  }\n  subgraph cluster_teams {\n    label=\"Teams\";\n")
		writeNodes(true)
		b.WriteString("  }\n")
	} else {
		writeNodes(false)
		writeNodes(true)
	}
	for _, rel := range cm.Relationships {
		left, right := relEnds(rel)
		style := "dir=forward"
		label := strings.Join(rel.UpstreamRoles, ", ")
		switch rel.Kind {
		case cml.RelPartnership:
			style = "dir=both"
			label = "P"
		case cml.RelSharedKernel:
			style = "dir=both"
			label = "SK"
		case cml.RelCustomerSupplier:
			label = "S"
			if len(rel.UpstreamRoles) > 0 {
				label += " " + strings.Join(rel.UpstreamRoles, ",")
			}
			label += " / C"
			if len(rel.DownstreamRoles) > 0 {
				label += " " + strings.Join(rel.DownstreamRoles, ",")
			}
		default:
			if label == "" {
				label = "U"
			}
			if len(rel.DownstreamRoles) > 0 {
				label += " / " + strings.Join(rel.DownstreamRoles, ",")
			}
		}
		if rel.Name != "" {
			label = rel.Name + ": " + label
		}
		fmt.Fprintf(&b, "  %s -> %s [%s, label=\"%s\"];\n", id(left), id(right), style, label)
	}
	b.WriteString("}\n")
	return b.String(), nil
}

func relEnds(rel *cml.Relationship) (string, string) {
	if rel.Kind == cml.RelPartnership || rel.Kind == cml.RelSharedKernel {
		return rel.Participant1, rel.Participant2
	}
	return rel.Upstream, rel.Downstream
}

// MDSL builds service contracts from aggregates, repositories and services.
func MDSL(model *cml.Model, base string) ([]File, error) {
	if model == nil {
		return nil, fmt.Errorf("no CML model")
	}
	var files []File
	for _, bc := range model.BoundedContexts {
		body := mdslContext(bc)
		if body == "" {
			continue
		}
		files = append(files, File{Name: base + "_" + bc.Name + ".mdsl", Body: body})
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("the model has no aggregates or services to export as MDSL")
	}
	return files, nil
}

func mdslContext(bc *cml.BoundedContext) string {
	var b strings.Builder
	wrote := false
	fmt.Fprintf(&b, "// generated from bounded context %s\nAPI description %sAPI\n\n", bc.Name, id(bc.Name))
	seen := map[string]bool{}
	for _, agg := range bc.AllAggregates() {
		for _, obj := range agg.Objects {
			if obj.Kind == cml.KindEnum || seen[obj.Name] {
				continue
			}
			seen[obj.Name] = true
			fmt.Fprintf(&b, "data type %s {\n", obj.Name)
			for _, attr := range obj.Attributes {
				fmt.Fprintf(&b, "  %q: %s,\n", attr.Name, mdslType(attr.Type))
			}
			for _, ref := range obj.References {
				fmt.Fprintf(&b, "  %q: %s,\n", ref.Name, ref.Type.RefName())
			}
			b.WriteString("}\n\n")
			wrote = true
		}
		for _, obj := range agg.Objects {
			if obj.Repository == nil {
				continue
			}
			writeEndpoint(&b, obj.Repository.Name, obj.Repository.Operations)
			wrote = true
		}
		for _, svc := range agg.Services {
			writeEndpoint(&b, svc.Name, svc.Operations)
			wrote = true
		}
	}
	for _, svc := range bc.Services {
		writeEndpoint(&b, svc.Name, svc.Operations)
		wrote = true
	}
	if !wrote {
		return ""
	}
	fmt.Fprintf(&b, "API provider %sProvider\n  offers %sAPI\n  at endpoint location \"http://localhost:8080\"\n    via protocol \"RESTful HTTP\"\n", id(bc.Name), id(bc.Name))
	return b.String()
}

func writeEndpoint(b *strings.Builder, name string, ops []*cml.Operation) {
	fmt.Fprintf(b, "endpoint type %s\n  exposes\n", name)
	if len(ops) == 0 {
		b.WriteString("    operation ping\n      delivering\n        payload D<string>\n\n")
		return
	}
	for _, op := range ops {
		fmt.Fprintf(b, "    operation %s\n", op.Name)
		if len(op.Params) > 0 {
			b.WriteString("      expecting\n        payload " + mdslPayload(op.Params) + "\n")
		}
		if op.Void {
			b.WriteString("      delivering\n        payload D<void>\n")
		} else if op.Return != nil {
			b.WriteString("      delivering\n        payload " + mdslType(op.Return) + "\n")
		}
		for _, thr := range op.Throws {
			fmt.Fprintf(b, "      throws %q\n", thr)
		}
	}
	b.WriteByte('\n')
}

func mdslPayload(params []*cml.Parameter) string {
	if len(params) == 1 {
		return mdslType(params[0].Type)
	}
	var parts []string
	for _, param := range params {
		parts = append(parts, fmt.Sprintf("%s: %s", param.Name, mdslType(param.Type)))
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

func mdslType(t *cml.ComplexType) string {
	if t == nil {
		return "D<string>"
	}
	name := t.String()
	switch name {
	case "String", "Date", "DateTime", "Timestamp":
		return "D<string>"
	case "int", "Integer", "long", "Long":
		return "D<int>"
	case "boolean", "Boolean":
		return "D<bool>"
	case "double", "Double", "float", "Float", "BigDecimal":
		return "D<double>"
	default:
		if t.DomainObject != "" {
			return t.DomainObject
		}
		if t.Collection != "" && t.Inner != nil {
			return t.Collection + "<" + mdslType(t.Inner) + ">"
		}
		if name == "" {
			return "D<string>"
		}
		return name
	}
}

// ServiceCutter builds a Service Cutter user-representation file from the CML model.
func ServiceCutter(model *cml.Model) (string, error) {
	if model == nil {
		return "", fmt.Errorf("no CML model")
	}
	var b strings.Builder
	for _, req := range model.Requirements {
		reads, writes := nanoentities(model, req)
		if len(reads) == 0 && len(writes) == 0 {
			continue
		}
		fmt.Fprintf(&b, "UseCase %s {\n", id(req.ReqName()))
		if len(reads) > 0 {
			fmt.Fprintf(&b, "  reads %s\n", quoteCSV(reads))
		}
		if len(writes) > 0 {
			fmt.Fprintf(&b, "  writes %s\n", quoteCSV(writes))
		}
		b.WriteString("}\n\n")
	}
	for _, bc := range model.BoundedContexts {
		for _, agg := range bc.AllAggregates() {
			var nanos []string
			for _, obj := range agg.Objects {
				if obj.Kind == cml.KindEnum {
					continue
				}
				for _, attr := range obj.Attributes {
					nanos = append(nanos, obj.Name+"."+attr.Name)
				}
				for _, ref := range obj.References {
					nanos = append(nanos, obj.Name+"."+ref.Name)
				}
			}
			if len(nanos) == 0 {
				continue
			}
			fmt.Fprintf(&b, "Aggregate %q {\n  %s\n}\n\n", agg.Name, quoteCSV(nanos))
		}
	}
	if b.Len() == 0 {
		return "", fmt.Errorf("the model has no aggregates or user requirements to export")
	}
	return b.String(), nil
}

func nanoentities(model *cml.Model, req cml.Requirement) (reads, writes []string) {
	index := map[string]*cml.DomainObject{}
	model.WalkObjects(func(_ *cml.BoundedContext, _ *cml.Aggregate, obj *cml.DomainObject) {
		index[obj.Name] = obj
	})
	for _, feature := range req.ReqFeatures() {
		obj := index[feature.Entity]
		if obj == nil {
			continue
		}
		var nanos []string
		for _, attr := range obj.Attributes {
			nanos = append(nanos, obj.Name+"."+attr.Name)
		}
		if feature.Verb == "read" {
			reads = append(reads, nanos...)
		} else {
			writes = append(writes, nanos...)
		}
	}
	return reads, writes
}

func quoteCSV(items []string) string {
	quoted := make([]string, len(items))
	for i, item := range items {
		quoted[i] = fmt.Sprintf("%q", item)
	}
	return strings.Join(quoted, ", ")
}

func id(name string) string {
	if name == "" || name == "[*]" {
		return name
	}
	var b strings.Builder
	for _, r := range name {
		if r == '_' || r == '-' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('_')
	}
	out := b.String()
	if out == "" {
		return "id"
	}
	if out[0] >= '0' && out[0] <= '9' {
		return "id_" + out
	}
	return out
}

func safe(s string) string {
	return strings.ReplaceAll(s, "\n", " ")
}
