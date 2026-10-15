package cml

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// SyntaxError is a CML syntax error.
type SyntaxError struct {
	File string
	Line int
	Col  int
	Msg  string
}

func (e *SyntaxError) Error() string {
	if e.File == "" {
		return fmt.Sprintf("%d:%d: %s", e.Line, e.Col, e.Msg)
	}
	return fmt.Sprintf("%s:%d:%d: %s", e.File, e.Line, e.Col, e.Msg)
}

type syntaxPanic struct{ err *SyntaxError }

type parser struct {
	file string
	toks []Token
	i    int
}

// Parse parses CML source. Imports are not loaded; use ParseFile for that.
func Parse(filename string, src []byte) (*Model, error) {
	toks, err := lexAll(filename, string(src))
	if err != nil {
		return nil, err
	}
	p := &parser{file: filename, toks: toks}
	var model *Model
	func() {
		defer func() {
			if r := recover(); r != nil {
				if sp, ok := r.(syntaxPanic); ok {
					err = sp.err
					model = nil
					return
				}
				panic(r)
			}
		}()
		model = p.parseModel()
	}()
	if err != nil {
		return nil, err
	}
	return model, nil
}

// ParseFile parses a CML file and recursively loads its imports.
func ParseFile(path string) (*Model, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	return parseFile(abs, map[string]bool{})
}

func parseFile(abs string, stack map[string]bool) (*Model, error) {
	if stack[abs] {
		return nil, fmt.Errorf("%s: cyclic import", abs)
	}
	src, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	model, err := Parse(abs, src)
	if err != nil {
		return nil, err
	}
	stack[abs] = true
	var imported []*Model
	for _, imp := range model.Imports {
		next := imp.URI
		if !filepath.IsAbs(next) {
			next = filepath.Join(filepath.Dir(abs), next)
		}
		next = filepath.Clean(next)
		child, err := parseFile(next, stack)
		if err != nil {
			return nil, err
		}
		imported = append(imported, child)
	}
	delete(stack, abs)
	if len(imported) == 0 {
		return model, nil
	}
	merged := &Model{File: abs, Comment: model.Comment}
	for _, child := range imported {
		if err := merged.absorb(child); err != nil {
			return nil, err
		}
	}
	if err := merged.absorb(model); err != nil {
		return nil, err
	}
	merged.Imports = model.Imports
	merged.File = abs
	return merged, nil
}

func (m *Model) absorb(other *Model) error {
	if other == nil {
		return nil
	}
	if other.Map != nil {
		if m.Map != nil {
			return fmt.Errorf("%s: imported model also defines a Context Map", other.File)
		}
		m.Map = other.Map
	}
	m.BoundedContexts = append(m.BoundedContexts, other.BoundedContexts...)
	m.Domains = append(m.Domains, other.Domains...)
	m.Requirements = append(m.Requirements, other.Requirements...)
	m.StakeholderBlocks = append(m.StakeholderBlocks, other.StakeholderBlocks...)
	m.ValueRegisters = append(m.ValueRegisters, other.ValueRegisters...)
	if m.Comment == "" {
		m.Comment = other.Comment
	}
	return nil
}

func (p *parser) parseModel() *Model {
	m := &Model{File: p.file}
	if len(p.toks) > 0 {
		m.Comment = p.toks[0].Lead
	}
	for !p.atEOF() {
		switch {
		case p.atKw("import"):
			pos := p.pos()
			p.next()
			uri := p.parseString()
			m.Imports = append(m.Imports, Import{Pos: pos, URI: uri})
		case p.atKw("ContextMap"):
			if m.Map != nil {
				p.die("only one Context Map is allowed")
			}
			m.Map = p.parseContextMap()
		case p.atKw("BoundedContext"):
			m.BoundedContexts = append(m.BoundedContexts, p.parseBoundedContext())
		case p.atKw("Domain"):
			m.Domains = append(m.Domains, p.parseDomain())
		case p.atKw("UseCase"):
			m.Requirements = append(m.Requirements, p.parseUseCase())
		case p.atKw("UserStory"):
			m.Requirements = append(m.Requirements, p.parseUserStory())
		case p.atKw("Stakeholders"):
			m.StakeholderBlocks = append(m.StakeholderBlocks, p.parseStakeholders())
		case p.atKw("ValueRegister"):
			m.ValueRegisters = append(m.ValueRegisters, p.parseValueRegister())
		default:
			p.die("unexpected %s", p.tok())
		}
	}
	return m
}

func (p *parser) parseContextMap() *ContextMap {
	cm := &ContextMap{Pos: p.pos(), Comment: p.tok().Lead}
	p.expectKw("ContextMap")
	if p.atKind(kIdent) {
		cm.Name = p.parseID()
	}
	p.expectKind(kLBrace)
	for !p.atKind(kRBrace) {
		prev := p.i
		switch {
		case p.atKw("type"):
			p.next()
			cm.Type = p.parseOneOf("UNDEFINED", "SYSTEM_LANDSCAPE", "ORGANIZATIONAL")
		case p.atKw("state"):
			p.next()
			cm.State = p.parseOneOf("UNDEFINED", "AS_IS", "TO_BE")
		case p.atKw("contains"):
			p.next()
			cm.Contains = append(cm.Contains, p.parseIDList()...)
		case p.atKind(kIdent) || p.atKind(kLBrack):
			cm.Relationships = append(cm.Relationships, p.parseRelationship())
		default:
			p.die("unexpected %s in Context Map", p.tok())
		}
		p.guard(prev)
	}
	p.expectKind(kRBrace)
	return cm
}

func (p *parser) parseRelationship() *Relationship {
	rel := &Relationship{Pos: p.pos(), Comment: p.tok().Lead}
	left := p.parseEndpoint()
	conn := p.parseConnector()
	right := p.parseEndpoint()
	switch conn {
	case "<->":
		rel.Participant1, rel.Participant2 = left.name, right.name
		if hasRole(left.roles, "P") || hasRole(right.roles, "P") {
			rel.Kind = RelPartnership
		} else {
			rel.Kind = RelSharedKernel
		}
	case "Partnership":
		rel.Kind = RelPartnership
		rel.Participant1, rel.Participant2 = left.name, right.name
	case "Shared-Kernel":
		rel.Kind = RelSharedKernel
		rel.Participant1, rel.Participant2 = left.name, right.name
	case "->":
		p.fillUpstream(rel, left, right, false)
	case "<-":
		p.fillUpstream(rel, right, left, false)
	case "Upstream-Downstream":
		p.fillUpstream(rel, left, right, false)
	case "Downstream-Upstream":
		p.fillUpstream(rel, right, left, false)
	case "Customer-Supplier":
		// Left is the customer (downstream), right is the supplier (upstream).
		p.fillUpstream(rel, right, left, true)
	case "Supplier-Customer":
		p.fillUpstream(rel, left, right, true)
	default:
		p.die("unknown relationship %q", conn)
	}
	if p.acceptKind(kColon) {
		rel.Name = p.parseID()
	}
	if p.acceptKind(kLBrace) {
		p.parseRelationshipBody(rel)
		p.expectKind(kRBrace)
	}
	return rel
}

func (p *parser) fillUpstream(rel *Relationship, up, down endpoint, forceCS bool) {
	rel.Upstream = up.name
	rel.Downstream = down.name
	rel.UpstreamRoles = patternRoles(up.roles)
	rel.DownstreamRoles = patternRoles(down.roles)
	cs := forceCS || hasRole(up.roles, "S") || hasRole(down.roles, "C")
	if cs {
		rel.Kind = RelCustomerSupplier
	} else {
		rel.Kind = RelUpstreamDownstream
	}
}

type endpoint struct {
	name  string
	roles []string
}

func (p *parser) parseEndpoint() endpoint {
	var ep endpoint
	if p.atKind(kLBrack) {
		ep.roles = append(ep.roles, p.parseRoleList()...)
	}
	if p.atKind(kIdent) {
		ep.name = p.parseID()
	}
	if p.atKind(kLBrack) {
		ep.roles = append(ep.roles, p.parseRoleList()...)
	}
	if ep.name == "" {
		p.die("expected bounded context name, got %s", p.tok())
	}
	return ep
}

func (p *parser) parseRoleList() []string {
	p.expectKind(kLBrack)
	if p.atKind(kRBrack) {
		p.next()
		return nil
	}
	roles := []string{p.parseID()}
	for p.acceptKind(kComma) {
		if p.atKind(kRBrack) {
			break
		}
		roles = append(roles, p.parseID())
	}
	p.expectKind(kRBrack)
	return roles
}

func (p *parser) parseConnector() string {
	t := p.tok()
	switch t.Kind {
	case kArrowB, kArrowR, kArrowL:
		p.next()
		return t.Lit
	}
	switch {
	case p.atKw("Partnership"), p.atKw("Shared-Kernel"), p.atKw("Upstream-Downstream"),
		p.atKw("Downstream-Upstream"), p.atKw("Customer-Supplier"), p.atKw("Supplier-Customer"):
		s := t.Lit
		p.next()
		return s
	default:
		p.die("expected a relationship arrow or keyword, got %s", t)
		return ""
	}
}

func (p *parser) parseRelationshipBody(rel *Relationship) {
	for !p.atKind(kRBrace) {
		prev := p.i
		switch {
		case p.atKw("implementationTechnology"):
			p.next()
			p.acceptKind(kEq)
			rel.ImplementationTechnology = p.parseString()
		case p.atKw("exposedAggregates"):
			p.next()
			rel.ExposedAggregates = p.parseIDList()
		case p.atKw("downstreamRights"):
			p.next()
			rel.DownstreamRights = p.parseOneOf("INFLUENCER", "OPINION_LEADER", "VETO_RIGHT", "DECISION_MAKER", "MONOPOLIST")
		default:
			p.die("unexpected %s in relationship", p.tok())
		}
		p.guard(prev)
	}
}

func (p *parser) parseBoundedContext() *BoundedContext {
	bc := &BoundedContext{Pos: p.pos(), Comment: p.tok().Lead}
	p.expectKw("BoundedContext")
	bc.Name = p.parseID()
	for p.atKw("implements") || p.atKw("realizes") || p.atKw("refines") {
		switch {
		case p.acceptKw("implements"):
			bc.Implements = append(bc.Implements, p.parseIDList()...)
		case p.acceptKw("realizes"):
			bc.Realizes = append(bc.Realizes, p.parseIDList()...)
		case p.acceptKw("refines"):
			if bc.Refines != "" {
				p.die("duplicate refines clause")
			}
			bc.Refines = p.parseID()
		}
	}
	if !p.acceptKind(kLBrace) {
		return bc
	}
	for !p.atKind(kRBrace) {
		prev := p.i
		switch {
		case p.atKw("domainVisionStatement"):
			p.next()
			p.acceptKind(kEq)
			bc.DomainVisionStatement = p.parseString()
		case p.atKw("type"):
			p.next()
			bc.Type = p.parseOneOf("UNDEFINED", "FEATURE", "APPLICATION", "SYSTEM", "TEAM")
		case p.atKw("responsibilities"):
			p.next()
			bc.Responsibilities = p.parseStringList()
		case p.atKw("implementationTechnology"):
			p.next()
			p.acceptKind(kEq)
			bc.ImplementationTechnology = p.parseString()
		case p.atKw("knowledgeLevel"):
			p.next()
			bc.KnowledgeLevel = p.parseOneOf("META", "CONCRETE")
		case p.atKw("businessModel"):
			p.next()
			p.acceptKind(kEq)
			bc.BusinessModel = p.parseString()
		case p.atKw("evolution"):
			p.next()
			bc.Evolution = p.parseOneOf("UNDEFINED", "GENESIS", "CUSTOM_BUILT", "PRODUCT", "COMMODITY")
		case p.atKw("Application"):
			bc.Application = p.parseApplication()
		case p.atModuleStart():
			bc.Modules = append(bc.Modules, p.parseModule())
		case p.atAggregateStart():
			bc.Aggregates = append(bc.Aggregates, p.parseAggregate())
		case p.atServiceStart():
			bc.Services = append(bc.Services, p.parseService())
		default:
			p.die("unexpected %s in Bounded Context %s", p.tok(), bc.Name)
		}
		p.guard(prev)
	}
	p.expectKind(kRBrace)
	return bc
}

func (p *parser) parseDomain() *Domain {
	d := &Domain{Pos: p.pos()}
	p.expectKw("Domain")
	d.Name = p.parseID()
	if !p.acceptKind(kLBrace) {
		return d
	}
	for !p.atKind(kRBrace) {
		prev := p.i
		switch {
		case p.atKw("domainVisionStatement"):
			p.next()
			p.acceptKind(kEq)
			d.DomainVisionStatement = p.parseString()
		case p.atKw("Subdomain"):
			d.Subdomains = append(d.Subdomains, p.parseSubdomain())
		default:
			p.die("unexpected %s in Domain %s", p.tok(), d.Name)
		}
		p.guard(prev)
	}
	p.expectKind(kRBrace)
	return d
}

func (p *parser) parseSubdomain() *Subdomain {
	sd := &Subdomain{Pos: p.pos()}
	p.expectKw("Subdomain")
	sd.Name = p.parseID()
	if p.acceptKw("supports") {
		sd.Supports = p.parseIDList()
	}
	if !p.acceptKind(kLBrace) {
		return sd
	}
	for !p.atKind(kRBrace) {
		prev := p.i
		switch {
		case p.atKw("type"):
			p.next()
			sd.Type = p.parseOneOf("UNDEFINED", "CORE_DOMAIN", "SUPPORTING_DOMAIN", "GENERIC_SUBDOMAIN")
		case p.atKw("domainVisionStatement"):
			p.next()
			p.acceptKind(kEq)
			sd.DomainVisionStatement = p.parseString()
		case p.atEntityStart():
			obj := p.parseDomainObject()
			if obj.Kind != KindEntity {
				p.die("only Entity is allowed in a Subdomain")
			}
			sd.Entities = append(sd.Entities, obj)
		case p.atServiceStart():
			sd.Services = append(sd.Services, p.parseService())
		default:
			p.die("unexpected %s in Subdomain %s", p.tok(), sd.Name)
		}
		p.guard(prev)
	}
	p.expectKind(kRBrace)
	return sd
}

func (p *parser) tok() Token {
	if p.i >= len(p.toks) {
		line, col := 1, 1
		if n := len(p.toks); n > 0 {
			last := p.toks[n-1]
			line, col = last.Line, last.Col
		}
		return Token{Kind: kEOF, Line: line, Col: col}
	}
	return p.toks[p.i]
}

func (p *parser) next() Token {
	t := p.tok()
	if p.i < len(p.toks) {
		p.i++
	}
	return t
}

func (p *parser) peek(n int) Token {
	j := p.i + n
	if j < 0 || j >= len(p.toks) {
		return Token{Kind: kEOF}
	}
	return p.toks[j]
}

func (p *parser) pos() Position {
	t := p.tok()
	return Position{File: p.file, Line: t.Line, Col: t.Col}
}

func (p *parser) die(format string, args ...any) {
	t := p.tok()
	panic(syntaxPanic{err: &SyntaxError{
		File: p.file,
		Line: t.Line,
		Col:  t.Col,
		Msg:  fmt.Sprintf(format, args...),
	}})
}

func (p *parser) guard(prev int) {
	if p.i == prev {
		p.die("internal parser stall at %s", p.tok())
	}
}

func (p *parser) atEOF() bool { return p.tok().Kind == kEOF }

func (p *parser) atKind(k Kind) bool { return p.tok().Kind == k }

func (p *parser) atKw(s string) bool {
	t := p.tok()
	return t.Kind == kIdent && !t.Esc && t.Lit == s
}

func (p *parser) acceptKind(k Kind) bool {
	if p.atKind(k) {
		p.next()
		return true
	}
	return false
}

func (p *parser) acceptKw(s string) bool {
	if p.atKw(s) {
		p.next()
		return true
	}
	return false
}

func (p *parser) expectKind(k Kind) {
	if !p.acceptKind(k) {
		p.die("expected %s, got %s", kindName(k), p.tok())
	}
}

func (p *parser) expectKw(s string) {
	if !p.acceptKw(s) {
		p.die("expected '%s', got %s", s, p.tok())
	}
}

func (p *parser) parseID() string {
	t := p.tok()
	if t.Kind != kIdent {
		p.die("expected identifier, got %s", t)
	}
	p.next()
	return t.Lit
}

func (p *parser) parseString() string {
	t := p.tok()
	if t.Kind != kString {
		p.die("expected string, got %s", t)
	}
	p.next()
	return t.Lit
}

func (p *parser) parseIDList() []string {
	p.acceptKind(kEq)
	ids := []string{p.parseID()}
	for p.acceptKind(kComma) {
		if !p.atKind(kIdent) {
			break
		}
		ids = append(ids, p.parseID())
	}
	return ids
}

func (p *parser) parseStringList() []string {
	p.acceptKind(kEq)
	out := []string{p.parseString()}
	for p.acceptKind(kComma) {
		out = append(out, p.parseString())
	}
	return out
}

func (p *parser) parseOneOf(vals ...string) string {
	p.acceptKind(kEq)
	id := p.parseID()
	if !slices.Contains(vals, id) {
		p.die("expected one of %s, got %q", strings.Join(vals, ", "), id)
	}
	return id
}

func (p *parser) parseQualifiedName() string {
	name := p.parseID()
	for p.atKind(kDot) && p.peek(1).Kind == kIdent {
		p.next()
		name += "." + p.parseID()
	}
	return name
}

func (p *parser) parseChannel() string {
	name := p.parseID()
	for {
		k := p.tok().Kind
		if (k == kDot || k == kSlash || k == kColon) && p.peek(1).Kind == kIdent {
			sep := p.tok().Lit
			p.next()
			name += sep + p.parseID()
			continue
		}
		break
	}
	return name
}

func hasRole(roles []string, role string) bool {
	return slices.Contains(roles, role)
}

func patternRoles(roles []string) []string {
	var out []string
	for _, r := range roles {
		switch r {
		case "U", "D", "P", "SK", "S", "C":
			continue
		default:
			out = append(out, r)
		}
	}
	return out
}
