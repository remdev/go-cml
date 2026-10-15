package cml

import (
	"slices"
	"strings"
)

const (
	opService = iota
	opRepo
	opResource
	opDomain
)

func (p *parser) atModuleStart() bool    { return p.keywordAhead("Module") }
func (p *parser) atAggregateStart() bool { return p.keywordAhead("Aggregate") }
func (p *parser) atServiceStart() bool   { return p.keywordAhead("Service") }
func (p *parser) atResourceStart() bool  { return p.keywordAhead("Resource") }
func (p *parser) atConsumerStart() bool  { return p.keywordAhead("Consumer") }

func (p *parser) atEntityStart() bool { return p.domainKeyword() == "Entity" }

func (p *parser) atDomainObjectStart() bool { return isDomainKeyword(p.domainKeyword()) }

func (p *parser) keywordAhead(kw string) bool {
	if p.atKw(kw) {
		return true
	}
	next := p.peek(1)
	return p.atKind(kString) && next.Kind == kIdent && !next.Esc && next.Lit == kw
}

func (p *parser) domainKeyword() string {
	idx := 0
	if p.atKind(kString) {
		idx = 1
	}
	t := p.peek(idx)
	if t.Kind == kIdent && !t.Esc && t.Lit == "abstract" {
		idx++
		t = p.peek(idx)
	}
	if t.Kind == kIdent && !t.Esc && isDomainKeyword(t.Lit) {
		return t.Lit
	}
	return ""
}

func isDomainKeyword(s string) bool {
	switch s {
	case "Entity", "ValueObject", "DomainEvent", "Event", "Command", "CommandEvent",
		"DataTransferObject", "Trait", "BasicType", "enum":
		return true
	default:
		return false
	}
}

func (p *parser) parseModule() *Module {
	m := &Module{Pos: p.pos()}
	if p.atKind(kString) {
		m.Doc = p.parseString()
	}
	p.expectKw("Module")
	m.Name = p.parseID()
	if !p.acceptKind(kLBrace) {
		return m
	}
	for !p.atKind(kRBrace) {
		prev := p.i
		switch {
		case p.acceptKw("external"):
			m.External = true
		case p.atKw("basePackage"):
			p.next()
			p.expectKind(kEq)
			m.BasePackage = p.parseQualifiedName()
		case p.atKw("hint") && p.peek(1).Kind == kEq:
			p.next()
			p.next()
			m.Hint = p.parseString()
		case p.atServiceStart():
			m.Services = append(m.Services, p.parseService())
		case p.atResourceStart():
			m.Resources = append(m.Resources, p.parseResource())
		case p.atConsumerStart():
			m.Consumers = append(m.Consumers, p.parseConsumer())
		case p.atAggregateStart():
			m.Aggregates = append(m.Aggregates, p.parseAggregate())
		case p.atDomainObjectStart():
			m.Objects = append(m.Objects, p.parseDomainObject())
		default:
			p.die("unexpected %s in Module %s", p.tok(), m.Name)
		}
		p.guard(prev)
	}
	p.expectKind(kRBrace)
	return m
}

func (p *parser) parseAggregate() *Aggregate {
	a := &Aggregate{Pos: p.pos()}
	if p.atKind(kString) {
		a.Comment = p.tok().Lead
		a.Doc = p.parseString()
	}
	if a.Comment == "" {
		a.Comment = p.tok().Lead
	}
	p.expectKw("Aggregate")
	a.Name = p.parseID()
	if !p.acceptKind(kLBrace) {
		return a
	}
	for !p.atKind(kRBrace) {
		prev := p.i
		switch {
		case p.atKw("responsibilities"):
			p.next()
			a.Responsibilities = p.parseStringList()
		case p.atKw("useCases"), p.atKw("userStories"), p.atKw("features"), p.atKw("userRequirements"):
			a.RequirementKind = p.parseID()
			a.RequirementRefs = append(a.RequirementRefs, p.parseIDList()...)
		case p.atKw("owner"):
			p.next()
			p.acceptKind(kEq)
			a.Owner = p.parseID()
		case p.atKw("knowledgeLevel"):
			p.next()
			a.KnowledgeLevel = p.parseOneOf("META", "CONCRETE")
		case p.atKw("likelihoodForChange"), p.atKw("structuralVolatility"):
			p.next()
			a.LikelihoodForChange = p.parseOneOf("UNDEFINED", "NORMAL", "RARELY", "OFTEN")
		case p.atKw("contentVolatility"):
			p.next()
			a.ContentVolatility = p.parseOneOf("UNDEFINED", "NORMAL", "RARELY", "OFTEN")
		case p.atKw("availabilityCriticality"):
			p.next()
			a.AvailabilityCriticality = p.parseOneOf("UNDEFINED", "NORMAL", "HIGH", "LOW")
		case p.atKw("consistencyCriticality"):
			p.next()
			a.ConsistencyCriticality = p.parseOneOf("UNDEFINED", "NORMAL", "HIGH", "LOW")
		case p.atKw("securityCriticality"):
			p.next()
			a.SecurityCriticality = p.parseOneOf("UNDEFINED", "NORMAL", "HIGH", "LOW")
		case p.atKw("storageSimilarity"):
			p.next()
			a.StorageSimilarity = p.parseOneOf("UNDEFINED", "NORMAL", "HUGE", "TINY")
		case p.atKw("securityZone"):
			p.next()
			p.acceptKind(kEq)
			a.SecurityZone = p.parseString()
		case p.atKw("securityAccessGroup"):
			p.next()
			p.acceptKind(kEq)
			a.SecurityAccessGroup = p.parseString()
		case p.atServiceStart():
			a.Services = append(a.Services, p.parseService())
		case p.atResourceStart():
			a.Resources = append(a.Resources, p.parseResource())
		case p.atConsumerStart():
			a.Consumers = append(a.Consumers, p.parseConsumer())
		case p.atDomainObjectStart():
			a.Objects = append(a.Objects, p.parseDomainObject())
		default:
			p.die("unexpected %s in Aggregate %s", p.tok(), a.Name)
		}
		p.guard(prev)
	}
	p.expectKind(kRBrace)
	return a
}

func (p *parser) parseDomainObject() *DomainObject {
	o := &DomainObject{Pos: p.pos()}
	if p.atKind(kString) {
		o.Comment = p.tok().Lead
		o.Doc = p.parseString()
	}
	if o.Comment == "" {
		o.Comment = p.tok().Lead
	}
	if p.acceptKw("abstract") {
		o.Abstract = true
	}
	switch {
	case p.acceptKw("Entity"):
		o.Kind = KindEntity
	case p.acceptKw("ValueObject"):
		o.Kind = KindValue
	case p.acceptKw("DomainEvent"), p.acceptKw("Event"):
		o.Kind = KindEvent
	case p.acceptKw("Command"), p.acceptKw("CommandEvent"):
		o.Kind = KindCommand
	case p.acceptKw("DataTransferObject"):
		o.Kind = KindDTO
	case p.acceptKw("Trait"):
		o.Kind = KindTrait
	case p.acceptKw("BasicType"):
		o.Kind = KindBasic
	case p.acceptKw("enum"):
		o.Kind = KindEnum
	default:
		p.die("expected a domain object, got %s", p.tok())
	}
	o.Name = p.parseID()
	if o.Kind != KindEnum && o.Kind != KindTrait {
		if p.acceptKw("extends") {
			if p.acceptKind(kAt) {
				o.ExtendsAt = true
				o.Extends = p.parseID()
			} else {
				o.ExtendsJava = true
				o.Extends = p.parseQualifiedName()
			}
		}
	}
	if o.Kind != KindEnum && o.Kind != KindDTO {
		for p.acceptKw("with") {
			tr := TraitRef{}
			if p.acceptKind(kAt) {
				tr.ExplicitAt = true
			}
			tr.Name = p.parseID()
			o.Traits = append(o.Traits, tr)
		}
	}
	if !p.acceptKind(kLBrace) {
		return o
	}
	if o.Kind == KindEnum {
		p.parseEnumBody(o)
	} else {
		p.parseObjectBody(o)
	}
	p.expectKind(kRBrace)
	return o
}

func (p *parser) parseObjectBody(o *DomainObject) {
	for !p.atKind(kRBrace) {
		prev := p.i
		if p.atNot() && p.negatableAhead() {
			p.next()
			p.setFlag(o, p.parseID(), false)
			continue
		}
		switch {
		case p.atKw("package") && p.peek(1).Kind == kEq:
			p.next()
			p.next()
			o.Package = p.parseQualifiedName()
		case p.atKw("hint") && p.peek(1).Kind == kEq:
			p.next()
			p.next()
			o.Hint = p.parseString()
		case p.acceptKw("optimisticLocking"):
			o.OptimisticLocking = new(true)
		case p.acceptKw("auditable"):
			o.Auditable = new(true)
		case p.acceptKw("cache"):
			o.Cache = new(true)
		case p.acceptKw("gap"):
			o.Gap = new(true)
		case p.acceptKw("nogap"):
			o.Gap = new(false)
		case p.acceptKw("scaffold"):
			o.Scaffold = true
		case p.atKw("databaseTable") && p.peek(1).Kind == kEq:
			p.next()
			p.next()
			o.DatabaseTable = p.parseString()
		case p.atKw("discriminatorValue") && p.peek(1).Kind == kEq:
			p.next()
			p.next()
			o.DiscriminatorValue = p.parseString()
		case p.atKw("discriminatorColumn") && p.peek(1).Kind == kEq:
			p.next()
			p.next()
			o.DiscriminatorColumn = p.parseString()
		case p.atKw("discriminatorType") && p.peek(1).Kind == kEq:
			p.next()
			p.next()
			o.DiscriminatorType = p.expectOneOf("STRING", "CHAR", "INTEGER")
		case p.atKw("discriminatorLength") && p.peek(1).Kind == kEq:
			p.next()
			p.next()
			o.DiscriminatorLength = p.parseString()
		case p.atKw("inheritanceType") && p.peek(1).Kind == kEq:
			p.next()
			p.next()
			o.InheritanceType = p.expectOneOf("JOINED", "SINGLE_TABLE")
		case p.atKw("validate") && p.peek(1).Kind == kEq:
			p.next()
			p.next()
			o.ValidateRule = p.parseString()
		case p.acceptKw("aggregateRoot"):
			o.AggregateRoot = true
		case p.acceptKw("persistent"):
			o.Persistent = new(true)
		case p.acceptKw("immutable"):
			o.Immutable = new(true)
		case p.atKw("belongsTo"):
			p.next()
			if p.acceptKind(kAt) {
				o.BelongsToAt = true
			}
			o.BelongsTo = p.parseQualifiedName()
		case p.atKw("Repository"):
			o.Repository = p.parseRepository()
		case p.atKind(kMinus) || p.atKw("reference") || (p.atKind(kString) && (p.peek(1).Kind == kMinus || p.peek(1).Lit == "reference")):
			o.References = append(o.References, p.parseReference())
		case p.atKind(kAssoc) || p.atKw("association") || (p.atKind(kString) && (p.peek(1).Kind == kAssoc || p.peek(1).Lit == "association")):
			o.Associations = append(o.Associations, p.parseAssociation())
		case p.atKind(kStar) || p.atKw("def") || (p.atKind(kString) && (p.peek(1).Kind == kStar || p.peek(1).Lit == "def")):
			o.Operations = append(o.Operations, p.parseOperation(opDomain))
		default:
			o.Attributes = append(o.Attributes, p.parseAttribute())
		}
		p.guard(prev)
	}
}

func (p *parser) atNot() bool {
	t := p.tok()
	return t.Kind == kNot || (t.Kind == kIdent && !t.Esc && t.Lit == "not")
}

func (p *parser) negatableAhead() bool {
	n := p.peek(1)
	if n.Kind != kIdent || n.Esc {
		return false
	}
	switch n.Lit {
	case "optimisticLocking", "auditable", "cache", "persistent", "immutable":
		return true
	default:
		return false
	}
}

func (p *parser) setFlag(o *DomainObject, kw string, value bool) {
	switch kw {
	case "optimisticLocking":
		o.OptimisticLocking = new(value)
	case "auditable":
		o.Auditable = new(value)
	case "cache":
		o.Cache = new(value)
	case "persistent":
		o.Persistent = new(value)
	case "immutable":
		o.Immutable = new(value)
	default:
		p.die("cannot negate %s", kw)
	}
}

func (p *parser) parseEnumBody(o *DomainObject) {
	for !p.atKind(kRBrace) {
		prev := p.i
		switch {
		case p.atKw("package") && p.peek(1).Kind == kEq:
			p.next()
			p.next()
			o.Package = p.parseQualifiedName()
		case p.atKw("hint") && p.peek(1).Kind == kEq:
			p.next()
			p.next()
			o.Hint = p.parseString()
		case p.acceptKw("aggregateLifecycle"):
			o.AggregateLifecycle = true
		case p.acceptKw("ordinal"):
			o.Ordinal = true
		case p.looksLikeEnumAttr():
			o.EnumAttributes = append(o.EnumAttributes, p.parseEnumAttr())
		default:
			for {
				o.EnumValues = append(o.EnumValues, p.parseEnumValue())
				if p.acceptKind(kComma) {
					if p.atKind(kRBrace) || p.atKind(kSemi) {
						p.acceptKind(kSemi)
						break
					}
					continue
				}
				p.acceptKind(kSemi)
				break
			}
		}
		p.guard(prev)
		if len(o.EnumValues) > 0 && p.atKind(kRBrace) {
			break
		}
	}
}

func (p *parser) looksLikeEnumAttr() bool {
	idx := 0
	if p.atKind(kString) {
		idx = 1
	}
	return p.peek(idx).Kind == kIdent && p.peek(idx+1).Kind == kIdent
}

func (p *parser) parseEnumAttr() *EnumAttribute {
	a := &EnumAttribute{}
	if p.atKind(kString) {
		a.Doc = p.parseString()
	}
	a.Type = p.parseQualifiedName()
	a.Name = p.parseID()
	if p.acceptKw("key") {
		a.Key = true
	}
	p.acceptKind(kSemi)
	return a
}

func (p *parser) parseEnumValue() *EnumValue {
	v := &EnumValue{}
	if p.atKind(kString) {
		v.Doc = p.parseString()
	}
	v.Name = p.parseID()
	if p.acceptKind(kLParen) {
		for !p.atKind(kRParen) {
			if p.atKind(kString) {
				v.Params = append(v.Params, EnumParam{String: p.parseString()})
			} else if p.atKind(kInt) {
				v.Params = append(v.Params, EnumParam{Int: p.tok().Lit, IsInt: true})
				p.next()
			} else {
				p.die("expected enum parameter, got %s", p.tok())
			}
			if !p.acceptKind(kComma) {
				break
			}
		}
		p.expectKind(kRParen)
	}
	return v
}

func (p *parser) parseService() *Service {
	s := &Service{Pos: p.pos()}
	if p.atKind(kString) {
		s.Comment = p.tok().Lead
		s.Doc = p.parseString()
	}
	if s.Comment == "" {
		s.Comment = p.tok().Lead
	}
	p.expectKw("Service")
	s.Name = p.parseID()
	if !p.acceptKind(kLBrace) {
		return s
	}
	for !p.atKind(kRBrace) {
		prev := p.i
		switch {
		case p.acceptKw("gap"):
			s.Gap = new(true)
		case p.acceptKw("nogap"):
			s.Gap = new(false)
		case p.atKw("hint") && p.peek(1).Kind == kEq:
			p.next()
			p.next()
			s.Hint = p.parseString()
		case p.acceptKw("webservice"):
			s.WebService = true
		case p.atKw("subscribe"):
			s.Subscribe = p.parseSubscribe()
		case p.atKind(kGt) || p.atKw("inject"):
			s.Dependencies = append(s.Dependencies, p.parseDependency())
		case p.atKind(kAssoc) || p.atKw("association"):
			s.Associations = append(s.Associations, p.parseAssociation())
		default:
			s.Operations = append(s.Operations, p.parseOperation(opService))
		}
		p.guard(prev)
	}
	p.expectKind(kRBrace)
	return s
}

func (p *parser) parseRepository() *Repository {
	r := &Repository{Pos: p.pos()}
	if p.atKind(kString) {
		r.Doc = p.parseString()
	}
	p.expectKw("Repository")
	r.Name = p.parseID()
	if !p.acceptKind(kLBrace) {
		return r
	}
	for !p.atKind(kRBrace) {
		prev := p.i
		switch {
		case p.acceptKw("gap"):
			r.Gap = new(true)
		case p.acceptKw("nogap"):
			r.Gap = new(false)
		case p.atKw("hint") && p.peek(1).Kind == kEq:
			p.next()
			p.next()
			r.Hint = p.parseString()
		case p.atKw("subscribe"):
			r.Subscribe = p.parseSubscribe()
		case p.atKind(kGt) || p.atKw("inject"):
			r.Dependencies = append(r.Dependencies, p.parseDependency())
		default:
			r.Operations = append(r.Operations, p.parseOperation(opRepo))
		}
		p.guard(prev)
	}
	p.expectKind(kRBrace)
	return r
}

func (p *parser) parseResource() *Resource {
	r := &Resource{Pos: p.pos()}
	if p.atKind(kString) {
		r.Doc = p.parseString()
	}
	p.expectKw("Resource")
	r.Name = p.parseID()
	if !p.acceptKind(kLBrace) {
		return r
	}
	for !p.atKind(kRBrace) {
		prev := p.i
		switch {
		case p.acceptKw("gap"):
			r.Gap = new(true)
		case p.acceptKw("nogap"):
			r.Gap = new(false)
		case p.atKw("hint") && p.peek(1).Kind == kEq:
			p.next()
			p.next()
			r.Hint = p.parseString()
		case p.acceptKw("scaffold"):
			r.Scaffold = true
		case p.atKw("path") && p.peek(1).Kind == kEq:
			p.next()
			p.next()
			r.Path = p.parseString()
		case p.atKind(kGt) || p.atKw("inject"):
			r.Dependencies = append(r.Dependencies, p.parseDependency())
		default:
			r.Operations = append(r.Operations, p.parseOperation(opResource))
		}
		p.guard(prev)
	}
	p.expectKind(kRBrace)
	return r
}

func (p *parser) parseConsumer() *Consumer {
	c := &Consumer{Pos: p.pos()}
	if p.atKind(kString) {
		c.Doc = p.parseString()
	}
	p.expectKw("Consumer")
	c.Name = p.parseID()
	p.expectKind(kLBrace)
	for !p.atKind(kRBrace) {
		prev := p.i
		switch {
		case p.atKw("hint") && p.peek(1).Kind == kEq:
			p.next()
			p.next()
			c.Hint = p.parseString()
		case p.atKind(kGt) || p.atKw("inject"):
			c.Dependencies = append(c.Dependencies, p.parseDependency())
		case p.atKw("unmarshall"):
			p.next()
			p.expectKw("to")
			if p.acceptKind(kAt) {
				c.UnmarshallAt = true
			}
			c.Unmarshall = p.parseID()
		case p.atKw("queueName"), p.atKw("topicName"):
			c.ChannelKind = p.parseID()
			p.expectKind(kEq)
			c.Channel = p.parseChannel()
		case p.atKw("subscribe"):
			c.Subscribe = p.parseSubscribe()
		default:
			p.die("unexpected %s in Consumer %s", p.tok(), c.Name)
		}
		p.guard(prev)
	}
	p.expectKind(kRBrace)
	return c
}

func (p *parser) parseSubscribe() *Subscription {
	p.expectKw("subscribe")
	p.expectKw("to")
	sub := &Subscription{Topic: p.parseChannel()}
	if p.atKw("eventBus") {
		p.next()
		p.expectKind(kEq)
		sub.EventBus = p.parseID()
	}
	return sub
}

func (p *parser) parseDependency() Dependency {
	d := Dependency{}
	switch {
	case p.acceptKind(kGt):
	case p.acceptKw("inject"):
		d.Inject = true
	default:
		p.die("expected '>' or inject, got %s", p.tok())
	}
	if p.acceptKind(kAt) {
		d.ExplicitAt = true
	}
	d.Ref = p.parseID()
	return d
}

func (p *parser) parseOperation(kind int) *Operation {
	op := &Operation{Pos: p.pos()}
	if p.atKind(kString) {
		op.Doc = p.parseString()
	}
	if kind == opDomain {
		if p.atKind(kStar) {
			p.next()
		} else {
			p.expectKw("def")
		}
		if p.acceptKw("abstract") {
			op.Abstract = true
		}
	}
	if p.isVisibility() {
		op.Visibility = p.parseID()
	}
	op.Return, op.Void, op.Name = p.splitReturnAndName()
	if p.acceptKind(kLParen) {
		if !p.atKind(kRParen) {
			for {
				op.Params = append(op.Params, p.parseParam())
				if !p.acceptKind(kComma) {
					break
				}
			}
		}
		p.expectKind(kRParen)
	}
	p.parseOpSuffix(op, kind)
	p.expectKind(kSemi)
	return op
}

func (p *parser) splitReturnAndName() (*ComplexType, bool, string) {
	if p.atKw("void") {
		p.next()
		return nil, true, p.parseID()
	}
	if p.looksLikeReturnType() {
		return p.parseComplexType(), false, p.parseID()
	}
	return nil, false, p.parseID()
}

func (p *parser) looksLikeReturnType() bool {
	if p.atKind(kAt) {
		return true
	}
	if !p.atKind(kIdent) {
		return false
	}
	n := p.peek(1)
	return n.Kind == kLt || n.Kind == kDot || n.Kind == kIdent
}

func (p *parser) parseParam() *Parameter {
	param := &Parameter{}
	if p.atKind(kString) {
		param.Doc = p.parseString()
	}
	param.Type = p.parseComplexType()
	param.Name = p.parseID()
	return param
}

func (p *parser) parseComplexType() *ComplexType {
	if p.atKw("Map") && p.peek(1).Kind == kLt {
		p.next()
		p.expectKind(kLt)
		ct := &ComplexType{Collection: "Map"}
		if p.acceptKind(kAt) {
			ct.MapKeyAt = true
			ct.MapKeyDomain = p.parseID()
		} else {
			ct.MapKey = p.parseQualifiedName()
		}
		p.expectKind(kComma)
		ct.Inner = p.parseComplexType()
		p.expectKind(kGt)
		return ct
	}
	if p.isCollection() && p.peek(1).Kind == kLt {
		col := p.parseID()
		p.expectKind(kLt)
		inner := p.parseComplexType()
		p.expectKind(kGt)
		return &ComplexType{Collection: col, Inner: inner}
	}
	if p.acceptKind(kAt) {
		return &ComplexType{DomainObject: p.parseID(), ExplicitAt: true}
	}
	name := p.parseQualifiedName()
	if p.acceptKind(kLt) {
		p.expectKind(kAt)
		do := p.parseID()
		p.expectKind(kGt)
		return &ComplexType{TypeName: name, DomainObject: do, ExplicitAt: true}
	}
	return &ComplexType{TypeName: name}
}

func (p *parser) isCollection() bool {
	t := p.tok()
	if t.Kind != kIdent || t.Esc {
		return false
	}
	switch t.Lit {
	case "Set", "List", "Bag", "Collection":
		return true
	default:
		return false
	}
}

func (p *parser) parseOpSuffix(op *Operation, kind int) {
	for !p.atKind(kSemi) && !p.atKind(kRBrace) && !p.atEOF() {
		prev := p.i
		switch {
		case p.acceptKw("throws"):
			op.Throws = append(op.Throws, p.parseQualifiedName())
			for p.acceptKind(kComma) {
				op.Throws = append(op.Throws, p.parseQualifiedName())
			}
		case p.atKw("hint") && p.peek(1).Kind == kEq:
			p.next()
			p.next()
			op.Hint = p.parseString()
		case p.atKw("publish"):
			op.Publish = p.parsePublish()
		case p.acceptKw("cache"):
			op.Cache = true
		case p.acceptKw("gap"):
			op.Gap = new(true)
		case p.acceptKw("nogap"):
			op.Gap = new(false)
		case p.stringEq("query"):
			op.Query = p.takeStringAttr()
		case p.stringEq("condition"):
			op.Condition = p.takeStringAttr()
		case p.stringEq("select"):
			op.Select = p.takeStringAttr()
		case p.stringEq("groupBy"):
			op.GroupBy = p.takeStringAttr()
		case p.stringEq("orderBy"):
			op.OrderBy = p.takeStringAttr()
		case p.acceptKw("construct"):
			op.Construct = true
		case p.acceptKw("build"):
			op.Build = true
		case p.acceptKw("map"):
			op.MapFlag = true
		case p.atKw("GET"), p.atKw("POST"), p.atKw("PUT"), p.atKw("DELETE"):
			op.HTTPMethod = p.parseID()
		case p.stringEq("path"):
			op.Path = p.takeStringAttr()
		case p.stringEq("return"):
			op.ReturnString = p.takeStringAttr()
		case p.atKind(kDelegate) || (p.atKw("delegates") && p.peek(1).Lit == "to"):
			if kind == opRepo {
				op.RepoDelegate = p.parseRepoDelegate()
			} else {
				op.Delegate = p.parseDelegate()
			}
		case p.atKind(kColon):
			p.next()
			if p.acceptKw("read-only") {
				op.ReadOnly = true
			} else if p.acceptKw("write") {
				op.Write = true
				if p.atKind(kLBrack) {
					op.Transition = p.parseTransition()
				}
			} else {
				p.die("expected read-only or write, got %s", p.tok())
			}
		default:
			p.die("unexpected %s in operation %s", p.tok(), op.Name)
		}
		p.guard(prev)
	}
}

func (p *parser) stringEq(kw string) bool {
	return p.atKw(kw) && p.peek(1).Kind == kEq
}

func (p *parser) takeStringAttr() string {
	p.next()
	p.next()
	return p.parseString()
}

func (p *parser) parsePublish() *Publish {
	p.expectKw("publish")
	pub := &Publish{}
	if p.atKind(kAt) || (p.atKind(kIdent) && p.tok().Lit != "to") {
		if p.acceptKind(kAt) {
			pub.ExplicitAt = true
		}
		pub.Event = p.parseID()
	}
	p.expectKw("to")
	pub.Topic = p.parseChannel()
	if p.atKw("eventBus") {
		p.next()
		p.expectKind(kEq)
		pub.EventBus = p.parseID()
	}
	return pub
}

func (p *parser) parseDelegate() *Delegate {
	p.consumeDelegate()
	d := &Delegate{}
	if p.acceptKind(kAt) {
		d.ExplicitAt = true
	}
	d.Target = p.parseID()
	p.expectKind(kDot)
	d.Operation = p.parseID()
	return d
}

func (p *parser) parseRepoDelegate() string {
	p.consumeDelegate()
	if p.acceptKw("AccessObject") {
		return "AccessObject"
	}
	return p.parseID()
}

func (p *parser) consumeDelegate() {
	if p.acceptKind(kDelegate) {
		return
	}
	p.expectKw("delegates")
	p.expectKw("to")
}

func (p *parser) parseAttribute() *Attribute {
	a := &Attribute{Pos: p.pos()}
	if p.atKind(kString) {
		a.Doc = p.parseString()
	}
	if p.isVisibility() {
		a.Visibility = p.parseID()
	}
	a.Type = p.parseComplexType()
	a.Name = p.parseID()
	a.Mods = p.parseModifiers()
	p.acceptKind(kSemi)
	return a
}

func (p *parser) parseReference() *Reference {
	r := &Reference{Pos: p.pos()}
	if p.atKind(kString) {
		r.Doc = p.parseString()
	}
	if p.atKind(kMinus) {
		p.next()
	} else {
		p.expectKw("reference")
	}
	if p.isVisibility() {
		r.Visibility = p.parseID()
	}
	r.Type = p.parseComplexType()
	r.Name = p.parseID()
	r.Mods = p.parseModifiers()
	if p.atKind(kArrowB) || p.atKw("opposite") {
		p.next()
		r.Opposite = p.parseID()
	}
	if p.atKind(kAssoc) || p.atKw("association") {
		p.next()
		if p.atKind(kString) {
			r.Label = p.parseString()
		}
	}
	p.acceptKind(kSemi)
	return r
}

func (p *parser) parseAssociation() *Association {
	a := &Association{}
	if p.atKind(kString) {
		a.Doc = p.parseString()
	}
	if p.atKind(kAssoc) {
		p.next()
	} else {
		p.expectKw("association")
	}
	if p.atKind(kString) {
		a.Description = p.parseString()
	}
	if p.acceptKind(kAt) {
		a.ExplicitAt = true
	}
	a.Target = p.parseID()
	p.acceptKind(kSemi)
	return a
}

func (p *parser) isVisibility() bool {
	t := p.tok()
	if t.Kind != kIdent || t.Esc {
		return false
	}
	switch t.Lit {
	case "public", "protected", "private":
		return true
	case "package":
		return p.peek(1).Kind != kEq
	default:
		return false
	}
}

var modifierNames = map[string]bool{
	"key": true, "changeable": true, "required": true, "unique": true, "nullable": true,
	"index": true, "assertFalse": true, "assertTrue": true, "hint": true, "creditCardNumber": true,
	"digits": true, "email": true, "future": true, "past": true, "max": true, "min": true,
	"decimalMax": true, "decimalMin": true, "notEmpty": true, "notBlank": true, "pattern": true,
	"range": true, "size": true, "length": true, "scriptAssert": true, "url": true, "validate": true,
	"transient": true, "databaseColumn": true, "databaseType": true, "cascade": true, "fetch": true,
	"cache": true, "inverse": true, "databaseJoinTable": true, "databaseJoinColumn": true,
	"valid": true, "orderby": true, "orderColumn": true,
}

func (p *parser) parseModifiers() []Modifier {
	var mods []Modifier
	for {
		not := false
		if p.atNot() {
			n := p.peek(1)
			if n.Kind != kIdent || n.Esc || !modifierNames[n.Lit] {
				break
			}
			p.next()
			not = true
		}
		t := p.tok()
		if t.Kind != kIdent || t.Esc || !modifierNames[t.Lit] {
			break
		}
		p.next()
		mod := Modifier{Not: not, Name: t.Lit}
		if p.acceptKind(kEq) {
			mod.HasValue = true
			mod.Value = p.parseString()
		}
		mods = append(mods, mod)
	}
	return mods
}

func (p *parser) expectOneOf(vals ...string) string {
	id := p.parseID()
	if !slices.Contains(vals, id) {
		p.die("expected one of %s, got %q", strings.Join(vals, ", "), id)
	}
	return id
}
