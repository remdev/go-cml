package cml

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Token kinds produced by the CML lexer.
type Kind int

const (
	kEOF Kind = iota
	kIdent
	kString
	kInt
	kLBrace
	kRBrace
	kLParen
	kRParen
	kLBrack
	kRBrack
	kComma
	kSemi
	kDot
	kColon
	kEq
	kLt
	kGt
	kAt
	kPlus
	kStar
	kSlash
	kMinus
	kArrowR
	kArrowL
	kArrowB
	kAssoc
	kDelegate
	kDColon
	kNot
)

// Token is one CML lexical token. Lead is the raw comment text immediately
// preceding the token.
type Token struct {
	Kind Kind
	Lit  string
	Lead string
	Line int
	Col  int
	Esc  bool
}

func (t Token) String() string {
	switch t.Kind {
	case kEOF:
		return "end of file"
	case kIdent:
		if t.Esc {
			return "^" + t.Lit
		}
		return t.Lit
	case kString:
		return strconvQuote(t.Lit)
	case kInt:
		return t.Lit
	default:
		if t.Lit != "" {
			return t.Lit
		}
		return kindName(t.Kind)
	}
}

func kindName(k Kind) string {
	switch k {
	case kEOF:
		return "end of file"
	case kIdent:
		return "identifier"
	case kString:
		return "string"
	case kInt:
		return "integer"
	case kLBrace:
		return "'{'"
	case kRBrace:
		return "'}'"
	case kLParen:
		return "'('"
	case kRParen:
		return "')'"
	case kLBrack:
		return "'['"
	case kRBrack:
		return "']'"
	case kComma:
		return "','"
	case kSemi:
		return "';'"
	case kDot:
		return "'.'"
	case kColon:
		return "':'"
	case kEq:
		return "'='"
	case kLt:
		return "'<'"
	case kGt:
		return "'>'"
	case kAt:
		return "'@'"
	case kPlus:
		return "'+'"
	case kStar:
		return "'*'"
	case kSlash:
		return "'/'"
	case kMinus:
		return "'-'"
	case kArrowR:
		return "'->'"
	case kArrowL:
		return "'<-'"
	case kArrowB:
		return "'<->'"
	case kAssoc:
		return "'--'"
	case kDelegate:
		return "'=>'"
	case kDColon:
		return "'::'"
	case kNot:
		return "'!'"
	default:
		return "token"
	}
}

func strconvQuote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\t", `\t`, "\r", `\r`).Replace(s) + `"`
}

var compounds = map[string]bool{
	"read-only":           true,
	"Shared-Kernel":       true,
	"Upstream-Downstream": true,
	"Downstream-Upstream": true,
	"Customer-Supplier":   true,
	"Supplier-Customer":   true,
}

type scanner struct {
	file        string
	s           string
	i           int
	line        int
	col         int
	lead        strings.Builder
	failComment error
}

func lexAll(file, src string) ([]Token, error) {
	sc := &scanner{file: file, s: src, line: 1, col: 1}
	var toks []Token
	for {
		sc.skipSpaceAndComments()
		if sc.failComment != nil {
			return nil, sc.failComment
		}
		if sc.i >= len(sc.s) {
			break
		}
		tok, err := sc.scan()
		if err != nil {
			return nil, err
		}
		tok.Lead = sc.takeLead()
		toks = append(toks, tok)
	}
	return toks, nil
}

func (sc *scanner) takeLead() string {
	s := sc.lead.String()
	sc.lead.Reset()
	return strings.TrimRight(s, "\n")
}

func (sc *scanner) addLead(raw string) {
	if sc.lead.Len() > 0 {
		sc.lead.WriteByte('\n')
	}
	sc.lead.WriteString(raw)
}

func (sc *scanner) peek() rune {
	if sc.i >= len(sc.s) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(sc.s[sc.i:])
	return r
}

func (sc *scanner) peekAt(off int) rune {
	j := sc.i
	for range off {
		if j >= len(sc.s) {
			return 0
		}
		_, w := utf8.DecodeRuneInString(sc.s[j:])
		j += w
	}
	if j >= len(sc.s) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(sc.s[j:])
	return r
}

func (sc *scanner) bump() rune {
	if sc.i >= len(sc.s) {
		return 0
	}
	r, w := utf8.DecodeRuneInString(sc.s[sc.i:])
	sc.i += w
	if r == '\n' {
		sc.line++
		sc.col = 1
	} else {
		sc.col++
	}
	return r
}

func (sc *scanner) skipSpaceAndComments() {
	for sc.i < len(sc.s) {
		r := sc.peek()
		if r == ' ' || r == '\t' || r == '\r' || r == '\n' || r == '\f' {
			sc.bump()
			continue
		}
		if r == '/' && sc.peekAt(1) == '/' {
			sc.scanLineComment()
			continue
		}
		if r == '/' && sc.peekAt(1) == '*' {
			if err := sc.scanBlockComment(); err != nil {
				sc.failComment = err
				return
			}
			continue
		}
		return
	}
}

func (sc *scanner) scan() (Token, error) {
	if sc.failComment != nil {
		return Token{}, sc.failComment
	}
	startLine, startCol := sc.line, sc.col
	r := sc.bump()
	tok := Token{Line: startLine, Col: startCol}
	switch r {
	case '{':
		tok.Kind, tok.Lit = kLBrace, "{"
	case '}':
		tok.Kind, tok.Lit = kRBrace, "}"
	case '(':
		tok.Kind, tok.Lit = kLParen, "("
	case ')':
		tok.Kind, tok.Lit = kRParen, ")"
	case '[':
		tok.Kind, tok.Lit = kLBrack, "["
	case ']':
		tok.Kind, tok.Lit = kRBrack, "]"
	case ',':
		tok.Kind, tok.Lit = kComma, ","
	case ';':
		tok.Kind, tok.Lit = kSemi, ";"
	case '.':
		tok.Kind, tok.Lit = kDot, "."
	case '@':
		tok.Kind, tok.Lit = kAt, "@"
	case '+':
		tok.Kind, tok.Lit = kPlus, "+"
	case '*':
		tok.Kind, tok.Lit = kStar, "*"
	case '!':
		tok.Kind, tok.Lit = kNot, "!"
	case ':':
		if sc.peek() == ':' {
			sc.bump()
			tok.Kind, tok.Lit = kDColon, "::"
		} else {
			tok.Kind, tok.Lit = kColon, ":"
		}
	case '=':
		if sc.peek() == '>' {
			sc.bump()
			tok.Kind, tok.Lit = kDelegate, "=>"
		} else {
			tok.Kind, tok.Lit = kEq, "="
		}
	case '<':
		if sc.peek() == '-' && sc.peekAt(1) == '>' {
			sc.bump()
			sc.bump()
			tok.Kind, tok.Lit = kArrowB, "<->"
		} else if sc.peek() == '-' {
			sc.bump()
			tok.Kind, tok.Lit = kArrowL, "<-"
		} else {
			tok.Kind, tok.Lit = kLt, "<"
		}
	case '>':
		tok.Kind, tok.Lit = kGt, ">"
	case '-':
		if sc.peek() == '>' {
			sc.bump()
			tok.Kind, tok.Lit = kArrowR, "->"
		} else if sc.peek() == '-' {
			sc.bump()
			tok.Kind, tok.Lit = kAssoc, "--"
		} else {
			tok.Kind, tok.Lit = kMinus, "-"
		}
	case '/':
		tok.Kind, tok.Lit = kSlash, "/"
	case '"', '\'':
		lit, err := sc.scanString(r, startLine, startCol)
		if err != nil {
			return Token{}, err
		}
		tok.Kind, tok.Lit = kString, lit
	default:
		if r == '^' || isIdentStart(r) {
			esc := false
			if r == '^' {
				esc = true
				if !isIdentStart(sc.peek()) {
					return Token{}, sc.syntax(startLine, startCol, "expected identifier after '^'")
				}
				r = sc.bump()
			}
			lit := sc.scanIdent(r)
			if !esc {
				if compound, ok := sc.tryCompound(lit); ok {
					lit = compound
				}
			}
			tok.Kind, tok.Lit, tok.Esc = kIdent, lit, esc
			return tok, nil
		}
		if unicode.IsDigit(r) {
			lit := sc.scanNumber(r)
			tok.Kind, tok.Lit = kInt, lit
			return tok, nil
		}
		return Token{}, sc.syntax(startLine, startCol, "unexpected character %q", r)
	}
	return tok, nil
}

func (sc *scanner) scanIdent(first rune) string {
	var b strings.Builder
	b.WriteRune(first)
	for {
		r := sc.peek()
		if !isIdentPart(r) {
			break
		}
		b.WriteRune(sc.bump())
	}
	return b.String()
}

func (sc *scanner) tryCompound(first string) (string, bool) {
	if sc.peek() != '-' {
		return "", false
	}
	// Save and try to read -ident.
	i, line, col := sc.i, sc.line, sc.col
	sc.bump()
	if !isIdentStart(sc.peek()) {
		sc.i, sc.line, sc.col = i, line, col
		return "", false
	}
	second := sc.scanIdent(sc.bump())
	joined := first + "-" + second
	if compounds[joined] {
		return joined, true
	}
	sc.i, sc.line, sc.col = i, line, col
	return "", false
}

func (sc *scanner) scanNumber(first rune) string {
	var b strings.Builder
	b.WriteRune(first)
	for {
		r := sc.peek()
		if !unicode.IsDigit(r) {
			break
		}
		b.WriteRune(sc.bump())
	}
	return b.String()
}

func (sc *scanner) scanString(quote rune, line, col int) (string, error) {
	var b strings.Builder
	for {
		if sc.i >= len(sc.s) {
			return "", sc.syntax(line, col, "unterminated string")
		}
		r := sc.bump()
		if r == quote {
			return b.String(), nil
		}
		if r == '\n' {
			return "", sc.syntax(line, col, "unterminated string")
		}
		if r == '\\' {
			if sc.i >= len(sc.s) {
				return "", sc.syntax(line, col, "unterminated string")
			}
			esc := sc.bump()
			switch esc {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			case '\\', '"', '\'':
				b.WriteRune(esc)
			default:
				b.WriteRune(esc)
			}
			continue
		}
		b.WriteRune(r)
	}
}

func (sc *scanner) scanLineComment() {
	start := sc.i
	sc.bump()
	sc.bump()
	for sc.i < len(sc.s) && sc.peek() != '\n' {
		sc.bump()
	}
	sc.addLead(sc.s[start:sc.i])
}

func (sc *scanner) scanBlockComment() error {
	start := sc.i
	line, col := sc.line, sc.col
	sc.bump()
	sc.bump()
	for sc.i < len(sc.s) {
		if sc.peek() == '*' && sc.peekAt(1) == '/' {
			sc.bump()
			sc.bump()
			sc.addLead(sc.s[start:sc.i])
			return nil
		}
		sc.bump()
	}
	return sc.syntax(line, col, "unterminated block comment")
}

func (sc *scanner) syntax(line, col int, format string, args ...any) error {
	return &SyntaxError{File: sc.file, Line: line, Col: col, Msg: fmt.Sprintf(format, args...)}
}

func isIdentStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isIdentPart(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
