package cml

import (
	"fmt"
	"strconv"
	"strings"
)

// String prints the model as CML.
func (m *Model) String() string {
	if m == nil {
		return ""
	}
	var p printer
	for _, imp := range m.Imports {
		p.linef("import %s", strconv.Quote(imp.URI))
	}
	if m.Map != nil {
		p.comment(m.Map.Comment)
		p.contextMap(m.Map)
		p.blank()
	}
	for _, bc := range m.BoundedContexts {
		p.comment(bc.Comment)
		p.boundedContext(bc)
		p.blank()
	}
	for _, d := range m.Domains {
		p.domain(d)
		p.blank()
	}
	for _, req := range m.Requirements {
		switch item := req.(type) {
		case *UseCase:
			p.comment(item.Comment)
			p.useCase(item)
		case *UserStory:
			p.comment(item.Comment)
			p.userStory(item)
		}
		p.blank()
	}
	for _, block := range m.StakeholderBlocks {
		p.stakeholders(block)
		p.blank()
	}
	for _, reg := range m.ValueRegisters {
		p.valueRegister(reg)
		p.blank()
	}
	return strings.TrimRight(p.b.String(), "\n") + "\n"
}

type printer struct {
	b      strings.Builder
	indent int
}

func (p *printer) linef(format string, args ...any) {
	for range p.indent {
		p.b.WriteByte('\t')
	}
	fmt.Fprintf(&p.b, format, args...)
	p.b.WriteByte('\n')
}

func (p *printer) blank() { p.b.WriteByte('\n') }

func (p *printer) comment(raw string) {
	if strings.TrimSpace(raw) == "" {
		return
	}
	p.b.WriteString(raw)
	if !strings.HasSuffix(raw, "\n") {
		p.b.WriteByte('\n')
	}
}

func (p *printer) contextMap(cm *ContextMap) {
	if cm.Name == "" {
		p.linef("ContextMap {")
	} else {
		p.linef("ContextMap %s {", cm.Name)
	}
	p.indent++
	if cm.Type != "" {
		p.linef("type = %s", cm.Type)
	}
	if cm.State != "" {
		p.linef("state = %s", cm.State)
	}
	if len(cm.Contains) > 0 {
		p.linef("contains %s", strings.Join(cm.Contains, ", "))
	}
	for _, rel := range cm.Relationships {
		p.comment(rel.Comment)
		p.relationship(rel)
	}
	p.indent--
	p.linef("}")
}

func (p *printer) relationship(rel *Relationship) {
	var line string
	switch rel.Kind {
	case RelPartnership:
		line = fmt.Sprintf("%s [P] <-> [P] %s", rel.Participant1, rel.Participant2)
	case RelSharedKernel:
		line = fmt.Sprintf("%s [SK] <-> [SK] %s", rel.Participant1, rel.Participant2)
	case RelCustomerSupplier:
		line = fmt.Sprintf("%s [%s] -> [%s] %s", rel.Upstream, joinRoles("U", "S", rel.UpstreamRoles), joinRoles("D", "C", rel.DownstreamRoles), rel.Downstream)
	default:
		line = fmt.Sprintf("%s [%s] -> [%s] %s", rel.Upstream, joinRoles("U", "", rel.UpstreamRoles), joinRoles("D", "", rel.DownstreamRoles), rel.Downstream)
	}
	if rel.Name != "" {
		line += " : " + rel.Name
	}
	body := rel.ImplementationTechnology != "" || len(rel.ExposedAggregates) > 0 || rel.DownstreamRights != ""
	if !body {
		p.linef("%s", line)
		return
	}
	p.linef("%s {", line)
	p.indent++
	if rel.ImplementationTechnology != "" {
		p.linef("implementationTechnology = %s", strconv.Quote(rel.ImplementationTechnology))
	}
	if len(rel.ExposedAggregates) > 0 {
		p.linef("exposedAggregates = %s", strings.Join(rel.ExposedAggregates, ", "))
	}
	if rel.DownstreamRights != "" {
		p.linef("downstreamRights = %s", rel.DownstreamRights)
	}
	p.indent--
	p.linef("}")
}

func joinRoles(marker, extra string, roles []string) string {
	parts := []string{marker}
	if extra != "" {
		parts = append(parts, extra)
	}
	parts = append(parts, roles...)
	return strings.Join(parts, ", ")
}

func (p *printer) boundedContext(bc *BoundedContext) {
	head := "BoundedContext " + bc.Name
	if len(bc.Implements) > 0 {
		head += " implements " + strings.Join(bc.Implements, ", ")
	}
	if len(bc.Realizes) > 0 {
		head += " realizes " + strings.Join(bc.Realizes, ", ")
	}
	if bc.Refines != "" {
		head += " refines " + bc.Refines
	}
	if !bc.hasBody() {
		p.linef("%s", head)
		return
	}
	p.linef("%s {", head)
	p.indent++
	if bc.DomainVisionStatement != "" {
		p.linef("domainVisionStatement = %s", strconv.Quote(bc.DomainVisionStatement))
	}
	if bc.Type != "" {
		p.linef("type = %s", bc.Type)
	}
	if len(bc.Responsibilities) > 0 {
		p.linef("responsibilities = %s", quoteList(bc.Responsibilities))
	}
	if bc.ImplementationTechnology != "" {
		p.linef("implementationTechnology = %s", strconv.Quote(bc.ImplementationTechnology))
	}
	if bc.KnowledgeLevel != "" {
		p.linef("knowledgeLevel = %s", bc.KnowledgeLevel)
	}
	if bc.BusinessModel != "" {
		p.linef("businessModel = %s", strconv.Quote(bc.BusinessModel))
	}
	if bc.Evolution != "" {
		p.linef("evolution = %s", bc.Evolution)
	}
	if bc.Application != nil {
		p.application(bc.Application)
	}
	for _, mod := range bc.Modules {
		p.module(mod)
	}
	for _, agg := range bc.Aggregates {
		p.aggregate(agg)
	}
	for _, svc := range bc.Services {
		p.service(svc)
	}
	p.indent--
	p.linef("}")
}

func (bc *BoundedContext) hasBody() bool {
	return bc.DomainVisionStatement != "" || bc.Type != "" || len(bc.Responsibilities) > 0 ||
		bc.ImplementationTechnology != "" || bc.KnowledgeLevel != "" || bc.BusinessModel != "" ||
		bc.Evolution != "" || bc.Application != nil || len(bc.Modules) > 0 || len(bc.Aggregates) > 0 ||
		len(bc.Services) > 0
}

func (p *printer) application(app *Application) {
	if app.Name == "" {
		p.linef("Application {")
	} else {
		p.linef("Application %s {", app.Name)
	}
	p.indent++
	for _, cmd := range app.Commands {
		p.domainObject(cmd)
	}
	for _, ev := range app.Events {
		p.domainObject(ev)
	}
	for _, svc := range app.Services {
		p.service(svc)
	}
	for _, flow := range app.Flows {
		p.flow(flow)
	}
	for _, coord := range app.Coordinations {
		p.coordination(coord)
	}
	p.indent--
	p.linef("}")
}

func (p *printer) flow(flow *Flow) {
	p.linef("Flow %s {", flow.Name)
	p.indent++
	for _, step := range flow.Steps {
		p.linef("%s", stepText(step))
	}
	p.indent--
	p.linef("}")
}

func stepText(step *FlowStep) string {
	var b strings.Builder
	if step.Kind == "trigger" {
		b.WriteString("event ")
		b.WriteString(strings.Join(step.Events, " + "))
		b.WriteString(" triggers ")
		if len(step.Commands) > 0 {
			b.WriteString("command ")
			b.WriteString(altText(step.TriggerAlt, step.Commands))
		} else {
			b.WriteString("operation ")
			b.WriteString(altText(step.TriggerAlt, step.Operations))
		}
		return b.String()
	}
	if step.Command != "" {
		b.WriteString("command " + step.Command)
	} else {
		b.WriteString("operation " + step.Operation)
	}
	if step.Actor != "" {
		b.WriteString(" [initiated by " + strconv.Quote(step.Actor) + "]")
	}
	if step.Aggregate != "" {
		b.WriteString(" delegates to " + step.Aggregate)
		if step.Transition != nil {
			b.WriteByte(' ')
			b.WriteString(transitionText(step.Transition))
		}
	}
	b.WriteString(" emits event ")
	b.WriteString(altText(step.EventAlt, step.Events))
	return b.String()
}

func altText(kind string, names []string) string {
	sep := ""
	switch kind {
	case "concurrent":
		sep = " + "
	case "exclusive":
		sep = " X "
	case "inclusive":
		sep = " O "
	default:
		if len(names) == 1 {
			return names[0]
		}
		sep = " + "
	}
	return strings.Join(names, sep)
}

func transitionText(st *StateTransition) string {
	var b strings.Builder
	b.WriteByte('[')
	b.WriteString(strings.Join(st.From, ", "))
	if len(st.From) > 0 {
		b.WriteByte(' ')
	}
	b.WriteString("-> ")
	var targets []string
	for _, to := range st.To {
		s := to.Value
		if to.End {
			s += "*"
		}
		targets = append(targets, s)
	}
	sep := ", "
	if st.Exclusive {
		sep = " X "
	}
	b.WriteString(strings.Join(targets, sep))
	b.WriteByte(']')
	return b.String()
}

func (p *printer) coordination(c *Coordination) {
	p.linef("Coordination %s {", c.Name)
	p.indent++
	for _, step := range c.Steps {
		p.linef("%s :: %s :: %s;", step.BoundedContext, step.Service, step.Operation)
	}
	p.indent--
	p.linef("}")
}

func (p *printer) module(m *Module) {
	if m.Doc != "" {
		p.linef("%s", strconv.Quote(m.Doc))
	}
	p.linef("Module %s {", m.Name)
	p.indent++
	if m.External {
		p.linef("external")
	}
	if m.BasePackage != "" {
		p.linef("basePackage = %s", m.BasePackage)
	}
	if m.Hint != "" {
		p.linef("hint = %s", strconv.Quote(m.Hint))
	}
	for _, agg := range m.Aggregates {
		p.aggregate(agg)
	}
	for _, obj := range m.Objects {
		p.domainObject(obj)
	}
	for _, svc := range m.Services {
		p.service(svc)
	}
	for _, res := range m.Resources {
		p.resource(res)
	}
	for _, cons := range m.Consumers {
		p.consumer(cons)
	}
	p.indent--
	p.linef("}")
}

func (p *printer) aggregate(a *Aggregate) {
	p.comment(a.Comment)
	if a.Doc != "" {
		p.linef("%s", strconv.Quote(a.Doc))
	}
	if !aggregateHasBody(a) {
		p.linef("Aggregate %s", a.Name)
		return
	}
	p.linef("Aggregate %s {", a.Name)
	p.indent++
	if len(a.Responsibilities) > 0 {
		p.linef("responsibilities = %s", quoteList(a.Responsibilities))
	}
	if len(a.RequirementRefs) > 0 {
		kind := a.RequirementKind
		if kind == "" {
			kind = "userRequirements"
		}
		p.linef("%s = %s", kind, strings.Join(a.RequirementRefs, ", "))
	}
	if a.Owner != "" {
		p.linef("owner = %s", a.Owner)
	}
	if a.KnowledgeLevel != "" {
		p.linef("knowledgeLevel = %s", a.KnowledgeLevel)
	}
	if a.LikelihoodForChange != "" {
		p.linef("likelihoodForChange = %s", a.LikelihoodForChange)
	}
	if a.ContentVolatility != "" {
		p.linef("contentVolatility = %s", a.ContentVolatility)
	}
	if a.AvailabilityCriticality != "" {
		p.linef("availabilityCriticality = %s", a.AvailabilityCriticality)
	}
	if a.ConsistencyCriticality != "" {
		p.linef("consistencyCriticality = %s", a.ConsistencyCriticality)
	}
	if a.StorageSimilarity != "" {
		p.linef("storageSimilarity = %s", a.StorageSimilarity)
	}
	if a.SecurityCriticality != "" {
		p.linef("securityCriticality = %s", a.SecurityCriticality)
	}
	if a.SecurityZone != "" {
		p.linef("securityZone = %s", strconv.Quote(a.SecurityZone))
	}
	if a.SecurityAccessGroup != "" {
		p.linef("securityAccessGroup = %s", strconv.Quote(a.SecurityAccessGroup))
	}
	for _, obj := range a.Objects {
		p.domainObject(obj)
	}
	for _, svc := range a.Services {
		p.service(svc)
	}
	for _, res := range a.Resources {
		p.resource(res)
	}
	for _, cons := range a.Consumers {
		p.consumer(cons)
	}
	p.indent--
	p.linef("}")
}

func aggregateHasBody(a *Aggregate) bool {
	return len(a.Responsibilities) > 0 || len(a.RequirementRefs) > 0 || a.Owner != "" ||
		a.KnowledgeLevel != "" || a.LikelihoodForChange != "" || a.ContentVolatility != "" ||
		a.AvailabilityCriticality != "" || a.ConsistencyCriticality != "" || a.StorageSimilarity != "" ||
		a.SecurityCriticality != "" || a.SecurityZone != "" || a.SecurityAccessGroup != "" ||
		len(a.Objects) > 0 || len(a.Services) > 0 || len(a.Resources) > 0 || len(a.Consumers) > 0
}

func (p *printer) domainObject(o *DomainObject) {
	p.comment(o.Comment)
	if o.Doc != "" {
		p.linef("%s", strconv.Quote(o.Doc))
	}
	var head strings.Builder
	if o.Abstract {
		head.WriteString("abstract ")
	}
	switch o.Kind {
	case KindEvent:
		head.WriteString("DomainEvent ")
	case KindCommand:
		head.WriteString("Command ")
	default:
		head.WriteString(o.Kind + " ")
	}
	head.WriteString(o.Name)
	if o.Extends != "" {
		if o.ExtendsAt {
			head.WriteString(" extends @" + o.Extends)
		} else {
			head.WriteString(" extends " + o.Extends)
		}
	}
	for _, tr := range o.Traits {
		if tr.ExplicitAt {
			head.WriteString(" with @" + tr.Name)
		} else {
			head.WriteString(" with " + tr.Name)
		}
	}
	if !objectHasBody(o) {
		p.linef("%s", head.String())
		return
	}
	p.linef("%s {", head.String())
	p.indent++
	p.objectFlags(o)
	if o.Kind == KindEnum {
		if o.AggregateLifecycle {
			p.linef("aggregateLifecycle")
		}
		if o.Ordinal {
			p.linef("ordinal")
		}
		for _, attr := range o.EnumAttributes {
			line := attr.Type + " " + attr.Name
			if attr.Key {
				line += " key"
			}
			p.linef("%s;", line)
		}
		if len(o.EnumValues) > 0 {
			var vals []string
			for _, v := range o.EnumValues {
				vals = append(vals, enumValueText(v))
			}
			p.linef("%s;", strings.Join(vals, ", "))
		}
	}
	for _, attr := range o.Attributes {
		p.attribute(attr)
	}
	for _, ref := range o.References {
		p.reference(ref)
	}
	for _, op := range o.Operations {
		p.writeOp(op, true)
	}
	for _, assoc := range o.Associations {
		p.association(assoc)
	}
	if o.Repository != nil {
		p.repository(o.Repository)
	}
	p.indent--
	p.linef("}")
}

func objectHasBody(o *DomainObject) bool {
	return o.Package != "" || o.Hint != "" || o.OptimisticLocking != nil || o.Auditable != nil || o.Cache != nil ||
		o.Gap != nil || o.Scaffold || o.DatabaseTable != "" || o.AggregateRoot || o.Persistent != nil ||
		o.Immutable != nil || o.BelongsTo != "" || o.ValidateRule != "" || o.AggregateLifecycle || o.Ordinal ||
		len(o.Attributes) > 0 || len(o.References) > 0 || len(o.Operations) > 0 || len(o.Associations) > 0 ||
		o.Repository != nil || len(o.EnumValues) > 0 || len(o.EnumAttributes) > 0 || o.InheritanceType != "" ||
		o.DiscriminatorType != ""
}

func (p *printer) objectFlags(o *DomainObject) {
	if o.Package != "" {
		p.linef("package = %s", o.Package)
	}
	writeTri := func(v *bool, word string) {
		if v == nil {
			return
		}
		if *v {
			p.linef("%s", word)
		} else {
			p.linef("not %s", word)
		}
	}
	writeTri(o.OptimisticLocking, "optimisticLocking")
	writeTri(o.Auditable, "auditable")
	writeTri(o.Cache, "cache")
	writeTri(o.Immutable, "immutable")
	writeTri(o.Persistent, "persistent")
	if o.Gap != nil {
		if *o.Gap {
			p.linef("gap")
		} else {
			p.linef("nogap")
		}
	}
	if o.Scaffold {
		p.linef("scaffold")
	}
	if o.Hint != "" {
		p.linef("hint = %s", strconv.Quote(o.Hint))
	}
	if o.DatabaseTable != "" {
		p.linef("databaseTable = %s", strconv.Quote(o.DatabaseTable))
	}
	if o.InheritanceType != "" {
		p.linef("inheritanceType = %s", o.InheritanceType)
	}
	if o.DiscriminatorType != "" {
		p.linef("discriminatorType = %s", o.DiscriminatorType)
	}
	if o.ValidateRule != "" {
		p.linef("validate = %s", strconv.Quote(o.ValidateRule))
	}
	if o.AggregateRoot {
		p.linef("aggregateRoot")
	}
	if o.BelongsTo != "" {
		if o.BelongsToAt {
			p.linef("belongsTo @%s", o.BelongsTo)
		} else {
			p.linef("belongsTo %s", o.BelongsTo)
		}
	}
}

func enumValueText(v *EnumValue) string {
	if len(v.Params) == 0 {
		return v.Name
	}
	var args []string
	for _, param := range v.Params {
		if param.IsInt {
			args = append(args, param.Int)
		} else {
			args = append(args, strconv.Quote(param.String))
		}
	}
	return v.Name + "(" + strings.Join(args, ", ") + ")"
}

func (p *printer) attribute(a *Attribute) {
	if a.Doc != "" {
		p.linef("%s", strconv.Quote(a.Doc))
	}
	p.linef("%s;", fieldText(a.Visibility, a.Type.String(), a.Name, a.Mods))
}

func (p *printer) reference(r *Reference) {
	if r.Doc != "" {
		p.linef("%s", strconv.Quote(r.Doc))
	}
	line := "- " + fieldText(r.Visibility, r.Type.String(), r.Name, r.Mods)
	if r.Opposite != "" {
		line += " opposite " + r.Opposite
	}
	p.linef("%s;", line)
}

func fieldText(vis, typ, name string, mods []Modifier) string {
	var parts []string
	if vis != "" {
		parts = append(parts, vis)
	}
	if typ != "" {
		parts = append(parts, typ)
	}
	parts = append(parts, name)
	for _, mod := range mods {
		parts = append(parts, mod.String())
	}
	return strings.Join(parts, " ")
}

func (m Modifier) String() string {
	s := m.Name
	if m.Not {
		s = "not " + s
	}
	if m.HasValue {
		s += " = " + strconv.Quote(m.Value)
	}
	return s
}

func (p *printer) operation(op *Operation) { p.writeOp(op, false) }

func (p *printer) writeOp(op *Operation, domain bool) {
	if op.Doc != "" {
		p.linef("%s", strconv.Quote(op.Doc))
	}
	var b strings.Builder
	if domain {
		b.WriteString("def ")
	}
	if op.Abstract {
		b.WriteString("abstract ")
	}
	if op.Visibility != "" {
		b.WriteString(op.Visibility + " ")
	}
	if op.Void {
		b.WriteString("void ")
	} else if op.Return != nil {
		b.WriteString(op.Return.String() + " ")
	}
	b.WriteString(op.Name)
	b.WriteByte('(')
	var params []string
	for _, param := range op.Params {
		params = append(params, strings.TrimSpace(param.Type.String()+" "+param.Name))
	}
	b.WriteString(strings.Join(params, ", "))
	b.WriteByte(')')
	if len(op.Throws) > 0 {
		b.WriteString(" throws " + strings.Join(op.Throws, ", "))
	}
	if op.Hint != "" {
		b.WriteString(" hint = " + strconv.Quote(op.Hint))
	}
	if op.ReadOnly {
		b.WriteString(" : read-only")
	}
	if op.Write {
		b.WriteString(" : write")
		if op.Transition != nil {
			b.WriteByte(' ')
			b.WriteString(transitionText(op.Transition))
		}
	}
	if op.Delegate != nil {
		target := op.Delegate.Target
		if op.Delegate.ExplicitAt {
			target = "@" + target
		}
		b.WriteString(" delegates to " + target + "." + op.Delegate.Operation)
	}
	b.WriteByte(';')
	p.linef("%s", b.String())
}

func (p *printer) association(a *Association) {
	target := a.Target
	if a.ExplicitAt {
		target = "@" + target
	}
	if a.Description != "" {
		p.linef("association %s %s;", strconv.Quote(a.Description), target)
		return
	}
	p.linef("association %s;", target)
}

func (p *printer) repository(r *Repository) {
	p.linef("Repository %s {", r.Name)
	p.indent++
	for _, op := range r.Operations {
		p.operation(op)
	}
	p.indent--
	p.linef("}")
}

func (p *printer) service(s *Service) {
	p.comment(s.Comment)
	if s.Doc != "" {
		p.linef("%s", strconv.Quote(s.Doc))
	}
	if len(s.Operations) == 0 && len(s.Dependencies) == 0 && s.Hint == "" && s.Subscribe == nil && !s.WebService && s.Gap == nil {
		p.linef("Service %s", s.Name)
		return
	}
	p.linef("Service %s {", s.Name)
	p.indent++
	if s.Gap != nil {
		if *s.Gap {
			p.linef("gap")
		} else {
			p.linef("nogap")
		}
	}
	if s.Hint != "" {
		p.linef("hint = %s", strconv.Quote(s.Hint))
	}
	if s.WebService {
		p.linef("webservice")
	}
	if s.Subscribe != nil {
		p.linef("%s", subscribeText(s.Subscribe))
	}
	for _, dep := range s.Dependencies {
		p.linef("%s", depText(dep))
	}
	for _, op := range s.Operations {
		p.operation(op)
	}
	p.indent--
	p.linef("}")
}

func (p *printer) resource(r *Resource) {
	p.linef("Resource %s {", r.Name)
	p.indent++
	if r.Path != "" {
		p.linef("path = %s", strconv.Quote(r.Path))
	}
	for _, op := range r.Operations {
		p.operation(op)
	}
	p.indent--
	p.linef("}")
}

func (p *printer) consumer(c *Consumer) {
	p.linef("Consumer %s {", c.Name)
	p.indent++
	if c.ChannelKind != "" {
		p.linef("%s = %s", c.ChannelKind, c.Channel)
	}
	if c.Unmarshall != "" {
		if c.UnmarshallAt {
			p.linef("unmarshall to @%s", c.Unmarshall)
		} else {
			p.linef("unmarshall to %s", c.Unmarshall)
		}
	}
	p.indent--
	p.linef("}")
}

func subscribeText(s *Subscription) string {
	line := "subscribe to " + s.Topic
	if s.EventBus != "" {
		line += " eventBus = " + s.EventBus
	}
	return line
}

func depText(d Dependency) string {
	kw := ">"
	if d.Inject {
		kw = "inject"
	}
	ref := d.Ref
	if d.ExplicitAt {
		ref = "@" + ref
	}
	return kw + " " + ref
}

func (p *printer) domain(d *Domain) {
	p.linef("Domain %s {", d.Name)
	p.indent++
	if d.DomainVisionStatement != "" {
		p.linef("domainVisionStatement = %s", strconv.Quote(d.DomainVisionStatement))
	}
	for _, sd := range d.Subdomains {
		p.subdomain(sd)
	}
	p.indent--
	p.linef("}")
}

func (p *printer) subdomain(sd *Subdomain) {
	head := "Subdomain " + sd.Name
	if len(sd.Supports) > 0 {
		head += " supports " + strings.Join(sd.Supports, ", ")
	}
	p.linef("%s {", head)
	p.indent++
	if sd.Type != "" {
		p.linef("type = %s", sd.Type)
	}
	if sd.DomainVisionStatement != "" {
		p.linef("domainVisionStatement = %s", strconv.Quote(sd.DomainVisionStatement))
	}
	for _, obj := range sd.Entities {
		p.domainObject(obj)
	}
	for _, svc := range sd.Services {
		p.service(svc)
	}
	p.indent--
	p.linef("}")
}

func (p *printer) useCase(uc *UseCase) {
	p.linef("UseCase %s {", uc.Name)
	p.indent++
	if uc.Actor != "" {
		p.linef("actor = %s", strconv.Quote(uc.Actor))
	}
	if len(uc.SecondaryActors) > 0 {
		p.linef("secondaryActors = %s", quoteList(uc.SecondaryActors))
	}
	if len(uc.Features) > 0 {
		var parts []string
		for _, f := range uc.Features {
			parts = append(parts, featureText(f))
		}
		p.linef("interactions = %s", strings.Join(parts, ", "))
	}
	if uc.Benefit != "" {
		p.linef("benefit = %s", strconv.Quote(uc.Benefit))
	}
	if uc.Scope != "" {
		p.linef("scope = %s", strconv.Quote(uc.Scope))
	}
	if uc.Level != "" {
		p.linef("level = %s", strconv.Quote(uc.Level))
	}
	p.indent--
	p.linef("}")
}

func (p *printer) userStory(story *UserStory) {
	head := "UserStory " + story.Name
	if story.SplitBy != "" {
		head += " split by " + story.SplitBy
	}
	p.linef("%s {", head)
	p.indent++
	if story.Role != "" || len(story.Features) > 0 || story.Benefit != "" {
		var b strings.Builder
		b.WriteString("As a " + strconv.Quote(story.Role))
		for _, f := range story.Features {
			b.WriteByte(' ')
			if f.WantTo {
				b.WriteString(featureText(f))
			} else {
				b.WriteString("I want to " + featureText(f))
			}
		}
		if story.Benefit != "" {
			b.WriteString(" so that " + strconv.Quote(story.Benefit))
		}
		p.linef("%s", b.String())
	}
	if story.Valuation != nil {
		p.linef("%s", valuationText(story.Valuation))
	}
	p.indent--
	p.linef("}")
}

func featureText(f *Feature) string {
	var b strings.Builder
	if f.WantTo {
		b.WriteString("I want to ")
	}
	switch f.Verb {
	case "create", "read", "update", "delete":
		b.WriteString(f.Verb)
	default:
		b.WriteString(strconv.Quote(f.Verb))
	}
	if f.Article != "" {
		b.WriteByte(' ')
		b.WriteString(f.Article)
	}
	b.WriteByte(' ')
	b.WriteString(strconv.Quote(f.Entity))
	if len(f.Attrs) > 0 {
		prep := f.AttrPrep
		if prep == "" {
			prep = "with its"
		}
		b.WriteByte(' ')
		b.WriteString(prep)
		for i, attr := range f.Attrs {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteByte(' ')
			b.WriteString(strconv.Quote(attr))
		}
	}
	if f.Container != "" {
		b.WriteByte(' ')
		b.WriteString(f.ContainerPrep)
		if f.ContainerArticle != "" {
			b.WriteByte(' ')
			b.WriteString(f.ContainerArticle)
		}
		b.WriteByte(' ')
		b.WriteString(strconv.Quote(f.Container))
	}
	return b.String()
}

func valuationText(v *Valuation) string {
	be := "is"
	if v.PromotedPlural {
		be = "are"
	}
	harmBe := "is"
	if v.HarmedPlural {
		harmBe = "are"
	}
	word := v.HarmWord
	if word == "" {
		word = "harmed"
	}
	return "and that " + quoteList(v.Promoted) + " " + be + " promoted, accepting that " + quoteList(v.Harmed) + " " + harmBe + " " + word
}

func (p *printer) stakeholders(block *StakeholderBlock) {
	head := "Stakeholders"
	if len(block.Contexts) > 0 {
		head += " of " + strings.Join(block.Contexts, ", ")
	}
	p.linef("%s {", head)
	p.indent++
	for _, item := range block.Items {
		if item.Group != nil {
			p.linef("StakeholderGroup %s {", item.Group.Name)
			p.indent++
			for _, person := range item.Group.Members {
				p.stakeholder(person)
			}
			p.indent--
			p.linef("}")
		}
		if item.Person != nil {
			p.stakeholder(item.Person)
		}
	}
	p.indent--
	p.linef("}")
}

func (p *printer) stakeholder(s *Stakeholder) {
	if s.Influence == "" && s.Interest == "" && s.Description == "" {
		p.linef("Stakeholder %s", s.Name)
		return
	}
	p.linef("Stakeholder %s {", s.Name)
	p.indent++
	if s.Influence != "" {
		p.linef("influence = %s", s.Influence)
	}
	if s.Interest != "" {
		p.linef("interest = %s", s.Interest)
	}
	if s.Description != "" {
		p.linef("description = %s", strconv.Quote(s.Description))
	}
	p.indent--
	p.linef("}")
}

func (p *printer) valueRegister(reg *ValueRegister) {
	head := "ValueRegister " + reg.Name
	if reg.Context != "" {
		head += " for " + reg.Context
	}
	p.linef("%s {", head)
	p.indent++
	for _, cluster := range reg.Clusters {
		p.linef("ValueCluster %s {", cluster.Name)
		p.indent++
		if cluster.Core != "" {
			if cluster.CoreLiteral {
				p.linef("core = %s", strconv.Quote(cluster.Core))
			} else {
				p.linef("core = %s", cluster.Core)
			}
		}
		for _, value := range cluster.Values {
			p.value(value)
		}
		p.indent--
		p.linef("}")
	}
	for _, value := range reg.Values {
		p.value(value)
	}
	p.indent--
	p.linef("}")
}

func (p *printer) value(v *Value) {
	p.linef("Value %s {", v.Name)
	p.indent++
	if v.IsCore {
		p.linef("isCore")
	}
	for _, e := range v.Elicitations {
		kw := "Stakeholder"
		if e.Plural {
			kw = "Stakeholders"
		}
		p.linef("%s %s {", kw, e.Stakeholder)
		p.indent++
		if e.Priority != "" {
			p.linef("priority = %s", e.Priority)
		}
		if e.Impact != "" {
			p.linef("impact = %s", e.Impact)
		}
		if len(e.Consequences) > 0 {
			p.linef("consequences")
			for _, c := range e.Consequences {
				line := c.Type + " " + strconv.Quote(c.Text)
				if c.Action != nil {
					line += " action " + strconv.Quote(c.Action.Text) + " " + c.Action.Type
				}
				p.linef("%s", line)
			}
		}
		p.indent--
		p.linef("}")
	}
	p.indent--
	p.linef("}")
}

func quoteList(items []string) string {
	quoted := make([]string, len(items))
	for i, item := range items {
		quoted[i] = strconv.Quote(item)
	}
	return strings.Join(quoted, ", ")
}
