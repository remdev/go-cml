package cml

import "fmt"

// Severity is the level of a semantic diagnostic.
type Severity int

const (
	// SevError rejects the model.
	SevError Severity = iota
	// SevWarning is reported without failing validation.
	SevWarning
	// SevInfo is a suggestion.
	SevInfo
)

func (s Severity) String() string {
	switch s {
	case SevWarning:
		return "warning"
	case SevInfo:
		return "info"
	default:
		return "error"
	}
}

// Diagnostic is one semantic finding.
type Diagnostic struct {
	Severity Severity
	Pos      Position
	Message  string
}

// RuleDiagnostic adds machine-readable rule metadata without changing the
// legacy Diagnostic value or its serialization. ValidateDetailed returns it.
type RuleDiagnostic struct {
	Diagnostic
	Code     RuleCode
	Category RuleCategory
}

func (d Diagnostic) String() string {
	if d.Pos.File == "" {
		return d.Severity.String() + ": " + d.Message
	}
	return fmt.Sprintf("%s:%d:%d: %s: %s", d.Pos.File, d.Pos.Line, d.Pos.Col, d.Severity, d.Message)
}

// HasErrors reports whether any diagnostic is an error.
func HasErrors(ds []Diagnostic) bool {
	for _, d := range ds {
		if d.Severity == SevError {
			return true
		}
	}
	return false
}

// makeDiagnostic uses the rule's fixed compatibility default. It does not apply
// project policies or permit validation checks to override their own severity.
func makeDiagnostic(code RuleCode, pos Position, args ...any) RuleDiagnostic {
	definition, ok := ruleDefinitions[code]
	if !ok {
		panic(fmt.Sprintf("unregistered CML validation rule %q", code))
	}
	return RuleDiagnostic{
		Severity: definition.DefaultSeverity,
		Pos:      pos,
		Message:  fmt.Sprintf(definition.format, args...),
		Code:     code,
		Category: definition.Category,
	}
}
