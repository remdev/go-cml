package cml

var coreValues = []string{
	"UNDEFINED", "AUTONOMY", "CARE", "CONTROL", "FAIRNESS", "INCLUSIVENESS", "INNOVATION",
	"PERFECTION", "PRIVACY", "RESPECT", "SUSTAINABILITY", "TRANSPARENCY", "TRUST",
}

var priorityValues = []string{"UNDEFINED", "HIGH", "MEDIUM", "LOW"}

func (p *parser) parseUseCase() *UseCase {
	uc := &UseCase{Pos: p.pos(), Comment: p.tok().Lead}
	p.expectKw("UseCase")
	uc.Name = p.parseID()
	if !p.acceptKind(kLBrace) {
		return uc
	}
	for !p.atKind(kRBrace) {
		prev := p.i
		switch {
		case p.atKw("actor"):
			p.next()
			p.acceptKind(kEq)
			uc.Actor = p.parseString()
		case p.atKw("secondaryActors"):
			p.next()
			uc.SecondaryActors = p.parseStringList()
		case p.atKw("interactions"):
			p.next()
			p.acceptKind(kEq)
			uc.Features = p.parseFeatureList()
		case p.atKw("benefit"):
			p.next()
			p.acceptKind(kEq)
			uc.Benefit = p.parseString()
		case p.atKw("scope"):
			p.next()
			p.acceptKind(kEq)
			uc.Scope = p.parseString()
		case p.atKw("level"):
			p.next()
			p.acceptKind(kEq)
			uc.Level = p.parseString()
		default:
			p.die("unexpected %s in UseCase %s", p.tok(), uc.Name)
		}
		p.guard(prev)
	}
	p.expectKind(kRBrace)
	return uc
}

func (p *parser) parseUserStory() *UserStory {
	story := &UserStory{Pos: p.pos(), Comment: p.tok().Lead}
	p.expectKw("UserStory")
	story.Name = p.parseID()
	if p.acceptKw("split") {
		p.expectKw("by")
		story.SplitBy = p.parseID()
	}
	if !p.acceptKind(kLBrace) {
		return story
	}
	if p.atKw("As") {
		p.next()
		if !p.acceptKw("a") && !p.acceptKw("an") {
			p.die("expected 'a' or 'an' after As")
		}
		story.Role = p.parseString()
		for !p.atKind(kRBrace) && (!p.atKw("so") || p.peek(1).Lit != "that") {
			story.Features = append(story.Features, p.parseFeature())
		}
		if p.atKw("so") {
			p.next()
			p.expectKw("that")
			story.Benefit = p.parseString()
		}
		if p.atKw("and") && p.peek(1).Lit == "that" {
			story.Valuation = p.parseValuation()
		}
	}
	p.expectKind(kRBrace)
	return story
}

// continuesAttribute reports whether a comma belongs to the attribute list.
// A following string is the next interaction when it is itself a verb: it is
// followed by an article or by the entity string. A following keyword such as
// "for" still belongs to the current interaction.
func (p *parser) continuesAttribute() bool {
	if !p.atKind(kComma) || p.peek(1).Kind != kString {
		return false
	}
	after := p.peek(2)
	if after.Kind == kString {
		return false
	}
	if after.Kind == kIdent && !after.Esc && (after.Lit == "a" || after.Lit == "an" || after.Lit == "the") {
		return false
	}
	return true
}

func (p *parser) parseFeatureList() []*Feature {
	features := []*Feature{p.parseFeature()}
	for p.acceptKind(kComma) {
		features = append(features, p.parseFeature())
	}
	return features
}

func (p *parser) parseFeature() *Feature {
	f := &Feature{Pos: p.pos()}
	if p.atKw("I") && p.peek(1).Lit == "want" && p.peek(2).Lit == "to" {
		p.next()
		p.next()
		p.next()
		f.WantTo = true
	}
	switch {
	case p.atKind(kString):
		f.Verb = p.parseString()
	case p.atKw("create"), p.atKw("read"), p.atKw("update"), p.atKw("delete"):
		f.Verb = p.parseID()
	default:
		p.die("expected a verb, got %s", p.tok())
	}
	if p.atKw("a") || p.atKw("an") || p.atKw("the") {
		f.Article = p.parseID()
	}
	f.Entity = p.parseString()
	if p.atKw("with") && (p.peek(1).Lit == "its" || p.peek(1).Lit == "their") {
		f.AttrPrep = p.parseID() + " " + p.parseID()
		f.Attrs = []string{p.parseString()}
		for p.continuesAttribute() {
			p.next()
			f.Attrs = append(f.Attrs, p.parseString())
		}
	}
	if p.atKw("in") || p.atKw("for") || p.atKw("to") {
		f.ContainerPrep = p.parseID()
		if p.atKw("a") || p.atKw("an") {
			f.ContainerArticle = p.parseID()
		}
		f.Container = p.parseString()
	}
	return f
}

func (p *parser) parseValuation() *Valuation {
	v := &Valuation{}
	p.expectKw("and")
	p.expectKw("that")
	v.Promoted = []string{p.parseString()}
	for p.atKind(kComma) && p.peek(1).Kind == kString {
		p.next()
		v.Promoted = append(v.Promoted, p.parseString())
	}
	if p.acceptKw("are") {
		v.PromotedPlural = true
	} else {
		p.expectKw("is")
	}
	p.expectKw("promoted")
	p.acceptKind(kComma)
	p.expectKw("accepting")
	p.expectKw("that")
	v.Harmed = []string{p.parseString()}
	for p.atKind(kComma) && p.peek(1).Kind == kString {
		p.next()
		v.Harmed = append(v.Harmed, p.parseString())
	}
	if p.acceptKw("are") {
		v.HarmedPlural = true
	} else {
		p.expectKw("is")
	}
	if p.atKw("reduced") || p.atKw("harmed") {
		v.HarmWord = p.parseID()
	} else {
		p.die("expected reduced or harmed, got %s", p.tok())
	}
	return v
}

func (p *parser) parseStakeholders() *StakeholderBlock {
	block := &StakeholderBlock{Pos: p.pos()}
	p.expectKw("Stakeholders")
	if p.acceptKw("of") {
		block.Contexts = p.parseIDList()
	}
	if !p.acceptKind(kLBrace) {
		return block
	}
	for !p.atKind(kRBrace) {
		prev := p.i
		switch {
		case p.atKw("StakeholderGroup"):
			block.Items = append(block.Items, StakeholderItem{Group: p.parseStakeholderGroup()})
		case p.atKw("Stakeholder"):
			block.Items = append(block.Items, StakeholderItem{Person: p.parseStakeholder()})
		default:
			p.die("unexpected %s in Stakeholders", p.tok())
		}
		p.guard(prev)
	}
	p.expectKind(kRBrace)
	return block
}

func (p *parser) parseStakeholderGroup() *StakeholderGroup {
	g := &StakeholderGroup{Pos: p.pos()}
	p.expectKw("StakeholderGroup")
	g.Name = p.parseID()
	if !p.acceptKind(kLBrace) {
		return g
	}
	for !p.atKind(kRBrace) {
		g.Members = append(g.Members, p.parseStakeholder())
	}
	p.expectKind(kRBrace)
	return g
}

func (p *parser) parseStakeholder() *Stakeholder {
	s := &Stakeholder{Pos: p.pos()}
	p.expectKw("Stakeholder")
	s.Name = p.parseID()
	if !p.acceptKind(kLBrace) {
		return s
	}
	for !p.atKind(kRBrace) {
		prev := p.i
		switch {
		case p.atKw("influence"):
			p.next()
			s.Influence = p.parseOneOf(priorityValues...)
		case p.atKw("interest"):
			p.next()
			s.Interest = p.parseOneOf(priorityValues...)
		case p.atKw("description"):
			p.next()
			p.acceptKind(kEq)
			s.Description = p.parseString()
		default:
			p.die("unexpected %s in Stakeholder %s", p.tok(), s.Name)
		}
		p.guard(prev)
	}
	p.expectKind(kRBrace)
	return s
}

func (p *parser) parseValueRegister() *ValueRegister {
	reg := &ValueRegister{Pos: p.pos()}
	p.expectKw("ValueRegister")
	reg.Name = p.parseID()
	if p.acceptKw("for") {
		reg.Context = p.parseID()
	}
	if !p.acceptKind(kLBrace) {
		return reg
	}
	for !p.atKind(kRBrace) {
		prev := p.i
		switch {
		case p.atKw("ValueCluster"):
			reg.Clusters = append(reg.Clusters, p.parseValueCluster())
		case p.atKw("Value"):
			reg.Values = append(reg.Values, p.parseValue())
		case p.atKw("ValueEpic"):
			reg.Epics = append(reg.Epics, p.parseValueEpic())
		case p.atKw("ValueNarrative"):
			reg.Narratives = append(reg.Narratives, p.parseValueNarrative())
		case p.atKw("ValueWeigthing"), p.atKw("ValueWeighting"):
			reg.Weightings = append(reg.Weightings, p.parseValueWeighting())
		default:
			p.die("unexpected %s in ValueRegister %s", p.tok(), reg.Name)
		}
		p.guard(prev)
	}
	p.expectKind(kRBrace)
	return reg
}

func (p *parser) parseValueCluster() *ValueCluster {
	c := &ValueCluster{Pos: p.pos()}
	p.expectKw("ValueCluster")
	c.Name = p.parseID()
	if !p.acceptKind(kLBrace) {
		return c
	}
	p.parseValueBody(func(kind string) {
		switch kind {
		case "core":
			p.acceptKind(kEq)
			if p.atKind(kString) {
				c.CoreLiteral = true
				c.Core = p.parseString()
			} else {
				c.Core = p.expectOneOf(coreValues...)
			}
		case "demonstrator":
			p.acceptKind(kEq)
			c.Demonstrators = append(c.Demonstrators, p.parseString())
		case "relatedValue":
			p.acceptKind(kEq)
			c.RelatedValues = append(c.RelatedValues, p.parseString())
		case "opposingValue":
			p.acceptKind(kEq)
			c.OpposingValues = append(c.OpposingValues, p.parseString())
		case "Value":
			c.Values = append(c.Values, p.parseValue())
		case "elicitation":
			c.Elicitations = append(c.Elicitations, p.parseElicitation())
		}
	})
	return c
}

func (p *parser) parseValue() *Value {
	v := &Value{Pos: p.pos()}
	p.expectKw("Value")
	v.Name = p.parseID()
	if !p.acceptKind(kLBrace) {
		return v
	}
	for !p.atKind(kRBrace) {
		prev := p.i
		switch {
		case p.acceptKw("isCore"):
			v.IsCore = true
		case p.atKw("demonstrator"):
			p.next()
			p.acceptKind(kEq)
			v.Demonstrators = append(v.Demonstrators, p.parseString())
		case p.atKw("relatedValue"):
			p.next()
			p.acceptKind(kEq)
			v.RelatedValues = append(v.RelatedValues, p.parseString())
		case p.atKw("opposingValue"):
			p.next()
			p.acceptKind(kEq)
			v.OpposingValues = append(v.OpposingValues, p.parseString())
		case p.atKw("Stakeholder"), p.atKw("Stakeholders"):
			v.Elicitations = append(v.Elicitations, p.parseElicitation())
		default:
			p.die("unexpected %s in Value %s", p.tok(), v.Name)
		}
		p.guard(prev)
	}
	p.expectKind(kRBrace)
	return v
}

func (p *parser) parseValueBody(handle func(kind string)) {
	for !p.atKind(kRBrace) {
		prev := p.i
		switch {
		case p.atKw("core"):
			p.next()
			handle("core")
		case p.atKw("demonstrator"):
			p.next()
			handle("demonstrator")
		case p.atKw("relatedValue"):
			p.next()
			handle("relatedValue")
		case p.atKw("opposingValue"):
			p.next()
			handle("opposingValue")
		case p.atKw("Value"):
			handle("Value")
		case p.atKw("Stakeholder"), p.atKw("Stakeholders"):
			handle("elicitation")
		default:
			p.die("unexpected %s in value cluster", p.tok())
		}
		p.guard(prev)
	}
	p.expectKind(kRBrace)
}

func (p *parser) parseElicitation() *ValueElicitation {
	e := &ValueElicitation{Pos: p.pos()}
	if p.acceptKw("Stakeholders") {
		e.Plural = true
	} else {
		p.expectKw("Stakeholder")
	}
	e.Stakeholder = p.parseID()
	if !p.acceptKind(kLBrace) {
		return e
	}
	for !p.atKind(kRBrace) {
		prev := p.i
		switch {
		case p.atKw("priority"):
			p.next()
			e.Priority = p.parseOneOf(priorityValues...)
		case p.atKw("impact"):
			p.next()
			e.Impact = p.parseOneOf(priorityValues...)
		case p.atKw("consequences"):
			p.next()
			if !p.atKw("good") && !p.atKw("bad") && !p.atKw("neutral") {
				p.die("expected at least one consequence")
			}
			for p.atKw("good") || p.atKw("bad") || p.atKw("neutral") {
				e.Consequences = append(e.Consequences, p.parseConsequence())
			}
		default:
			p.die("unexpected %s in value elicitation", p.tok())
		}
		p.guard(prev)
	}
	p.expectKind(kRBrace)
	return e
}

func (p *parser) parseConsequence() Consequence {
	c := Consequence{Type: p.parseID(), Text: p.parseString()}
	if p.atKw("action") {
		p.next()
		action := &Action{Text: p.parseString()}
		if p.atKind(kString) {
			action.Type = p.parseString()
		} else {
			action.Type = p.parseID()
		}
		c.Action = action
	}
	return c
}

func (p *parser) parseValueEpic() *ValueEpic {
	e := &ValueEpic{Pos: p.pos()}
	p.expectKw("ValueEpic")
	e.Name = p.parseID()
	p.expectKind(kLBrace)
	p.expectKw("As")
	p.expectKw("a")
	e.Stakeholder = p.parseID()
	p.expectKw("I")
	p.expectKw("value")
	e.Value = p.parseString()
	p.expectKw("as")
	p.expectKw("demonstrated")
	p.expectKw("in")
	sawRealization := false
	sawReduction := false
	for p.atKw("realization") || p.atKw("reduction") {
		if p.acceptKw("realization") {
			p.expectKw("of")
			e.RealizedValues = append(e.RealizedValues, p.parseString())
			sawRealization = true
			continue
		}
		p.expectKw("reduction")
		p.expectKw("of")
		e.ReducedValues = append(e.ReducedValues, p.parseString())
		sawReduction = true
	}
	if !sawRealization || !sawReduction {
		p.die("value epic requires both 'realization of' and 'reduction of'")
	}
	p.expectKind(kRBrace)
	return e
}

func (p *parser) parseValueNarrative() *ValueNarrative {
	n := &ValueNarrative{Pos: p.pos()}
	p.expectKw("ValueNarrative")
	n.Name = p.parseID()
	p.expectKind(kLBrace)
	p.expectKw("When")
	p.expectKw("the")
	p.expectKw("SOI")
	p.expectKw("executes")
	n.Feature = p.parseString()
	p.expectKind(kComma)
	p.expectKw("stakeholders")
	p.expectKw("expect")
	p.expectKw("it")
	p.expectKw("to")
	p.expectKw("promote")
	p.expectKind(kComma)
	p.expectKw("protect")
	p.expectKw("or")
	p.expectKw("create")
	n.PromotedValues = p.parseString()
	p.expectKind(kComma)
	p.expectKw("possibly")
	p.expectKw("degrading")
	p.expectKw("or")
	p.expectKw("prohibiting")
	n.HarmedValues = p.parseString()
	p.expectKw("with")
	p.expectKw("the")
	p.expectKw("following")
	p.expectKw("externally")
	p.expectKw("observable")
	p.expectKw("and")
	p.expectKind(kSlash)
	p.expectKw("or")
	p.expectKw("internally")
	p.expectKw("auditable")
	p.expectKw("behavior")
	p.expectKind(kColon)
	n.PreAndPostConditions = p.parseString()
	p.expectKind(kRBrace)
	return n
}

func (p *parser) parseValueWeighting() *ValueWeighting {
	w := &ValueWeighting{Pos: p.pos()}
	if p.atKw("ValueWeigthing") || p.atKw("ValueWeighting") {
		p.next()
	} else {
		p.die("expected ValueWeigthing, got %s", p.tok())
	}
	w.Name = p.parseID()
	p.expectKind(kLBrace)
	p.expectKw("In")
	p.expectKw("the")
	p.expectKw("context")
	p.expectKw("of")
	p.expectKw("the")
	p.expectKw("SOI")
	p.expectKind(kComma)
	p.expectKw("stakeholder")
	w.Stakeholder = p.parseID()
	p.expectKw("values")
	w.Value1 = p.parseString()
	p.expectKw("more")
	p.expectKw("than")
	w.Value2 = p.parseString()
	p.expectKw("expecting")
	p.expectKw("benefits")
	p.expectKw("such")
	p.expectKw("as")
	w.Benefits = p.parseString()
	p.expectKw("running")
	p.expectKw("the")
	p.expectKw("risk")
	p.expectKw("of")
	p.expectKw("harms")
	p.expectKw("such")
	p.expectKw("as")
	w.Harms = p.parseString()
	p.expectKind(kRBrace)
	return w
}
