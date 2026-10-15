package cml

func (p *parser) parseApplication() *Application {
	app := &Application{Pos: p.pos()}
	p.expectKw("Application")
	if p.atKind(kIdent) && !p.atDomainObjectStart() {
		app.Name = p.parseID()
	}
	if !p.acceptKind(kLBrace) {
		return app
	}
	for !p.atKind(kRBrace) {
		prev := p.i
		switch {
		case p.domainKeyword() == "Command" || p.domainKeyword() == "CommandEvent":
			app.Commands = append(app.Commands, p.parseDomainObject())
		case p.domainKeyword() == "DomainEvent" || p.domainKeyword() == "Event":
			app.Events = append(app.Events, p.parseDomainObject())
		case p.atServiceStart():
			app.Services = append(app.Services, p.parseService())
		case p.atKw("Flow"):
			app.Flows = append(app.Flows, p.parseFlow())
		case p.atKw("Coordination"):
			app.Coordinations = append(app.Coordinations, p.parseCoordination())
		default:
			p.die("unexpected %s in Application", p.tok())
		}
		p.guard(prev)
	}
	p.expectKind(kRBrace)
	return app
}

func (p *parser) parseFlow() *Flow {
	f := &Flow{Pos: p.pos()}
	p.expectKw("Flow")
	f.Name = p.parseID()
	if !p.acceptKind(kLBrace) {
		return f
	}
	for !p.atKind(kRBrace) {
		prev := p.i
		f.Steps = append(f.Steps, p.parseFlowStep())
		p.guard(prev)
	}
	p.expectKind(kRBrace)
	return f
}

func (p *parser) parseFlowStep() *FlowStep {
	step := &FlowStep{Pos: p.pos()}
	switch {
	case p.acceptKw("command"):
		step.Kind = "emit"
		step.Command = p.parseID()
		p.parseEmitTail(step)
	case p.acceptKw("operation"):
		step.Kind = "emit"
		step.Operation = p.parseID()
		p.parseEmitTail(step)
	case p.acceptKw("event"):
		step.Kind = "trigger"
		step.Events = []string{p.parseID()}
		for p.acceptKind(kPlus) {
			step.Events = append(step.Events, p.parseID())
		}
		p.expectKw("triggers")
		switch {
		case p.acceptKw("command"):
			step.TriggerAlt, step.Commands = p.parseAltList()
		case p.acceptKw("operation"):
			step.TriggerAlt, step.Operations = p.parseAltList()
		default:
			p.die("expected command or operation after triggers, got %s", p.tok())
		}
	default:
		p.die("expected command, operation or event, got %s", p.tok())
	}
	return step
}

func (p *parser) parseEmitTail(step *FlowStep) {
	if p.atKind(kLBrack) && p.peek(1).Lit == "initiated" {
		p.next()
		p.expectKw("initiated")
		p.expectKw("by")
		step.Actor = p.parseString()
		p.expectKind(kRBrack)
	}
	if p.atKind(kDelegate) || (p.atKw("delegates") && p.peek(1).Lit == "to") {
		p.consumeDelegate()
		step.Aggregate = p.parseID()
		if p.atKind(kLBrack) {
			step.Transition = p.parseTransition()
		}
	}
	p.expectKw("emits")
	p.expectKw("event")
	step.EventAlt, step.Events = p.parseAltList()
}

func (p *parser) parseAltList() (string, []string) {
	names := []string{p.parseID()}
	alt := ""
	for {
		sep := ""
		switch {
		case p.atKind(kPlus):
			sep = "concurrent"
		case p.atKw("X"), p.atKw("x"):
			sep = "exclusive"
		case p.atKw("O"), p.atKw("o"):
			sep = "inclusive"
		default:
			return alt, names
		}
		if alt != "" && alt != sep {
			p.die("mixed alternative separators in a flow step")
		}
		alt = sep
		p.next()
		names = append(names, p.parseID())
	}
}

func (p *parser) parseTransition() *StateTransition {
	p.expectKind(kLBrack)
	st := &StateTransition{}
	if !p.atKind(kArrowR) {
		st.From = append(st.From, p.parseID())
		for p.acceptKind(kComma) {
			st.From = append(st.From, p.parseID())
		}
	}
	p.expectKind(kArrowR)
	st.To = append(st.To, p.parseTarget())
	for p.atKw("X") || p.atKw("x") {
		st.Exclusive = true
		p.next()
		st.To = append(st.To, p.parseTarget())
	}
	p.expectKind(kRBrack)
	return st
}

func (p *parser) parseTarget() StateTarget {
	target := StateTarget{Value: p.parseID()}
	target.End = p.acceptKind(kStar)
	return target
}

func (p *parser) parseCoordination() *Coordination {
	c := &Coordination{Pos: p.pos()}
	p.expectKw("Coordination")
	c.Name = p.parseID()
	if !p.acceptKind(kLBrace) {
		return c
	}
	for !p.atKind(kRBrace) {
		prev := p.i
		step := CoordinationStep{Pos: p.pos()}
		step.BoundedContext = p.parseID()
		p.expectKind(kDColon)
		step.Service = p.parseID()
		p.expectKind(kDColon)
		step.Operation = p.parseID()
		p.expectKind(kSemi)
		c.Steps = append(c.Steps, step)
		p.guard(prev)
	}
	p.expectKind(kRBrace)
	return c
}
