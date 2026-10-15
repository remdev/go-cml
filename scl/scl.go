// Package scl parses the Service Cutter user-representation language used by Context Mapper.
package scl

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Model is a Service Cutter configuration.
type Model struct {
	UseCases               []*UseCase
	Aggregates             []*Group
	Entities               []*Group
	PredefinedServices     []*Group
	SecurityAccessGroups   []*Group
	SeparatedSecurityZones []*Group
	SharedOwnerGroups      []*Group
	Characteristics        []*Characteristic
}

// UseCase lists the nanoentities a use case reads and writes.
type UseCase struct {
	Name            string
	LatencyCritical bool
	Reads           []string
	Writes          []string
}

// Group is a named set of nanoentities.
type Group struct {
	Kind         string
	Name         string
	Nanoentities []string
}

// Characteristic is one compatibilities entry.
type Characteristic struct {
	Kind         string
	Name         string
	Nanoentities []string
}

// Parse parses an SCL document.
func Parse(filename, src string) (*Model, error) {
	p := &parser{file: filename, s: src, line: 1, col: 1}
	m := &Model{}
	for {
		p.skip()
		if p.eof() {
			return m, nil
		}
		kw := p.ident()
		switch kw {
		case "UseCase":
			m.UseCases = append(m.UseCases, p.useCase())
		case "Compatibilities":
			m.Characteristics = append(m.Characteristics, p.compatibilities()...)
		case "Aggregate", "Entity", "PredefinedService", "SecurityAccessGroup", "SeparatedSecurityZone", "SharedOwnerGroup":
			m.addGroup(p.group(kw))
		default:
			p.fail("unexpected %q", kw)
		}
		if p.err != nil {
			return nil, p.err
		}
	}
}

func (m *Model) addGroup(g *Group) {
	switch g.Kind {
	case "Aggregate":
		m.Aggregates = append(m.Aggregates, g)
	case "Entity":
		m.Entities = append(m.Entities, g)
	case "PredefinedService":
		m.PredefinedServices = append(m.PredefinedServices, g)
	case "SecurityAccessGroup":
		m.SecurityAccessGroups = append(m.SecurityAccessGroups, g)
	case "SeparatedSecurityZone":
		m.SeparatedSecurityZones = append(m.SeparatedSecurityZones, g)
	default:
		m.SharedOwnerGroups = append(m.SharedOwnerGroups, g)
	}
}

// String prints the model back to SCL.
func (m *Model) String() string {
	var b strings.Builder
	for _, uc := range m.UseCases {
		fmt.Fprintf(&b, "UseCase %s {\n", uc.Name)
		if uc.LatencyCritical {
			b.WriteString("\tisLatencyCritical = true\n")
		}
		if len(uc.Reads) > 0 {
			fmt.Fprintf(&b, "\treads %s\n", quoteJoin(uc.Reads))
		}
		if len(uc.Writes) > 0 {
			fmt.Fprintf(&b, "\twrites %s\n", quoteJoin(uc.Writes))
		}
		b.WriteString("}\n\n")
	}
	if len(m.Characteristics) > 0 {
		b.WriteString("Compatibilities {\n")
		for _, ch := range m.Characteristics {
			fmt.Fprintf(&b, "\t%s {\n\t\tcharacteristic %s\n", ch.Kind, ch.Name)
			if len(ch.Nanoentities) > 0 {
				fmt.Fprintf(&b, "\t\t%s\n", quoteJoin(ch.Nanoentities))
			}
			b.WriteString("\t}\n")
		}
		b.WriteString("}\n\n")
	}
	writeGroups := func(items []*Group) {
		for _, g := range items {
			fmt.Fprintf(&b, "%s %q {\n", g.Kind, g.Name)
			if len(g.Nanoentities) > 0 {
				fmt.Fprintf(&b, "\t%s\n", quoteJoin(g.Nanoentities))
			}
			b.WriteString("}\n\n")
		}
	}
	writeGroups(m.Aggregates)
	writeGroups(m.Entities)
	writeGroups(m.PredefinedServices)
	writeGroups(m.SecurityAccessGroups)
	writeGroups(m.SeparatedSecurityZones)
	writeGroups(m.SharedOwnerGroups)
	return b.String()
}

func quoteJoin(items []string) string {
	parts := make([]string, len(items))
	for i, item := range items {
		parts[i] = fmt.Sprintf("%q", item)
	}
	return strings.Join(parts, ", ")
}

type parser struct {
	file string
	s    string
	i    int
	line int
	col  int
	err  error
}

func (p *parser) useCase() *UseCase {
	uc := &UseCase{Name: p.name()}
	p.expect("{")
	for !p.consume("}") && !p.eof() && p.err == nil {
		switch p.ident() {
		case "isLatencyCritical":
			p.expect("=")
			uc.LatencyCritical = p.ident() == "true"
		case "reads":
			uc.Reads = p.stringList()
		case "writes":
			uc.Writes = p.stringList()
		default:
			p.fail("unexpected token in UseCase")
			return uc
		}
	}
	return uc
}

func (p *parser) group(kind string) *Group {
	g := &Group{Kind: kind, Name: p.name()}
	p.expect("{")
	g.Nanoentities = p.stringList()
	p.expect("}")
	return g
}

func (p *parser) compatibilities() []*Characteristic {
	p.expect("{")
	var out []*Characteristic
	for !p.consume("}") && !p.eof() && p.err == nil {
		kind := p.ident()
		p.expect("{")
		ch := &Characteristic{Kind: kind}
		for !p.consume("}") && !p.eof() && p.err == nil {
			if p.peekIdent() == "characteristic" {
				p.ident()
				ch.Name = p.ident()
				continue
			}
			items := p.stringList()
			if len(items) == 0 {
				p.fail("expected nanoentity or characteristic")
				break
			}
			ch.Nanoentities = append(ch.Nanoentities, items...)
		}
		out = append(out, ch)
	}
	return out
}

func (p *parser) stringList() []string {
	p.skip()
	var out []string
	if p.peekRune() != '"' && p.peekRune() != '\'' {
		return nil
	}
	for {
		out = append(out, p.str())
		p.skip()
		if !p.consume(",") {
			break
		}
	}
	return out
}

func (p *parser) name() string {
	p.skip()
	if p.peekRune() == '"' || p.peekRune() == '\'' {
		return p.str()
	}
	return p.ident()
}

func (p *parser) ident() string {
	p.skip()
	start := p.i
	r := p.bump()
	if !isLetter(r) {
		p.fail("expected identifier")
		return ""
	}
	for isLetter(p.peekRune()) || unicode.IsDigit(p.peekRune()) {
		p.bump()
	}
	return p.s[start:p.i]
}

func (p *parser) peekIdent() string {
	saved := *p
	id := p.ident()
	*p = saved
	return id
}

func (p *parser) str() string {
	p.skip()
	quote := p.bump()
	var b strings.Builder
	for {
		if p.eof() {
			p.fail("unterminated string")
			return ""
		}
		r := p.bump()
		if r == quote {
			return b.String()
		}
		if r == '\\' {
			b.WriteRune(p.bump())
			continue
		}
		b.WriteRune(r)
	}
}

func (p *parser) expect(tok string) {
	p.skip()
	if strings.HasPrefix(p.s[p.i:], tok) {
		for _, r := range tok {
			if p.bump() != r {
				break
			}
		}
		return
	}
	p.fail("expected %s", tok)
}

func (p *parser) consume(tok string) bool {
	p.skip()
	if strings.HasPrefix(p.s[p.i:], tok) {
		p.expect(tok)
		return true
	}
	return false
}

func (p *parser) skip() {
	for !p.eof() {
		r := p.peekRune()
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f' {
			p.bump()
			continue
		}
		if r == '/' && p.peekAt(1) == '/' {
			for !p.eof() && p.peekRune() != '\n' {
				p.bump()
			}
			continue
		}
		if r == '/' && p.peekAt(1) == '*' {
			p.bump()
			p.bump()
			for !p.eof() && (p.peekRune() != '*' || p.peekAt(1) != '/') {
				p.bump()
			}
			p.bump()
			p.bump()
			continue
		}
		return
	}
}

func (p *parser) eof() bool { return p.err != nil || p.i >= len(p.s) }

func (p *parser) peekRune() rune {
	if p.i >= len(p.s) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(p.s[p.i:])
	return r
}

func (p *parser) peekAt(n int) rune {
	j := p.i
	for range n {
		if j >= len(p.s) {
			return 0
		}
		_, w := utf8.DecodeRuneInString(p.s[j:])
		j += w
	}
	if j >= len(p.s) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(p.s[j:])
	return r
}

func (p *parser) bump() rune {
	if p.i >= len(p.s) {
		return 0
	}
	r, w := utf8.DecodeRuneInString(p.s[p.i:])
	p.i += w
	if r == '\n' {
		p.line++
		p.col = 1
	} else {
		p.col++
	}
	return r
}

func (p *parser) fail(format string, args ...any) {
	if p.err != nil {
		return
	}
	p.err = fmt.Errorf("%s:%d:%d: %s", p.file, p.line, p.col, fmt.Sprintf(format, args...))
}

func isLetter(r rune) bool { return r == '_' || unicode.IsLetter(r) }
