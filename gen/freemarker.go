// Package gen renders CML models to PlantUML, Graphviz context maps, MDSL and text.
package gen

import (
	"fmt"
	"strconv"
	"strings"
)

// Render evaluates a Freemarker-like template against data.
// Supported constructs are ${expr}, <#if>, <#elseif>, <#else>, <#list>, <#assign>
// and comments. Expressions support dotted paths, ==, !=, ??, !default, ?size and ?has_content.
func Render(template string, data map[string]any) (string, error) {
	template = strings.NewReplacer("<#lt>", "", "<#rt>", "", "<#t>", "", "<#nt>", "").Replace(template)
	p := &ft{s: template, root: data}
	nodes, stop, err := p.parse(nil)
	if err != nil {
		return "", err
	}
	if stop != "" {
		return "", fmt.Errorf("unexpected %s", stop)
	}
	var b strings.Builder
	if err := exec(&b, nodes, []map[string]any{data}); err != nil {
		return "", err
	}
	return b.String(), nil
}

type ft struct {
	s    string
	i    int
	root map[string]any
}

type node any

type textNode struct{ text string }
type interpNode struct{ expr string }
type ifNode struct {
	cond string
	then []node
	alt  []node
}
type listNode struct {
	expr string
	item string
	body []node
}
type assignNode struct{ name, expr string }

func (p *ft) parse(stop []string) ([]node, string, error) {
	var nodes []node
	for p.i < len(p.s) {
		if tag, ok := p.matchStop(stop); ok {
			return nodes, tag, nil
		}
		if strings.HasPrefix(p.s[p.i:], "<#--") {
			end := strings.Index(p.s[p.i+4:], "-->")
			if end < 0 {
				return nil, "", fmt.Errorf("unterminated comment")
			}
			p.i += 4 + end + 3
			continue
		}
		if strings.HasPrefix(p.s[p.i:], "${") {
			end := strings.IndexByte(p.s[p.i+2:], '}')
			if end < 0 {
				return nil, "", fmt.Errorf("unterminated interpolation")
			}
			nodes = append(nodes, interpNode{expr: strings.TrimSpace(p.s[p.i+2 : p.i+2+end])})
			p.i += 2 + end + 1
			continue
		}
		if strings.HasPrefix(p.s[p.i:], "<#") {
			raw, err := p.readDirective()
			if err != nil {
				return nil, "", err
			}
			fields := strings.Fields(raw)
			if len(fields) == 0 {
				return nil, "", fmt.Errorf("empty directive")
			}
			switch fields[0] {
			case "if":
				n, err := p.parseIf(strings.TrimSpace(strings.TrimPrefix(raw, "if")))
				if err != nil {
					return nil, "", err
				}
				nodes = append(nodes, n)
			case "list":
				n, err := p.parseList(fields)
				if err != nil {
					return nil, "", err
				}
				nodes = append(nodes, n)
			case "assign":
				name, expr, ok := strings.Cut(strings.TrimSpace(strings.TrimPrefix(raw, "assign")), "=")
				if !ok {
					return nil, "", fmt.Errorf("assign requires '='")
				}
				nodes = append(nodes, assignNode{name: strings.TrimSpace(name), expr: strings.TrimSpace(expr)})
			default:
				return nil, "", fmt.Errorf("unsupported directive <%s>", fields[0])
			}
			continue
		}
		next := p.i + 1
		if idx := strings.Index(p.s[p.i:], "<#"); idx >= 0 {
			next = p.i + idx
		}
		if idx := strings.Index(p.s[p.i:], "${"); idx >= 0 && p.i+idx < next {
			next = p.i + idx
		}
		nodes = append(nodes, textNode{text: p.s[p.i:next]})
		p.i = next
	}
	return nodes, "", nil
}

func (p *ft) matchStop(stop []string) (string, bool) {
	for _, tag := range stop {
		if strings.HasPrefix(p.s[p.i:], tag) {
			p.i += len(tag)
			return tag, true
		}
	}
	return "", false
}

func (p *ft) readDirective() (string, error) {
	end := strings.IndexByte(p.s[p.i:], '>')
	if end < 0 {
		return "", fmt.Errorf("unterminated directive")
	}
	raw := strings.TrimSpace(p.s[p.i+2 : p.i+end])
	p.i += end + 1
	return raw, nil
}

func (p *ft) parseIf(cond string) (ifNode, error) {
	then, stop, err := p.parse([]string{"<#elseif ", "<#else>", "</#if>"})
	if err != nil {
		return ifNode{}, err
	}
	n := ifNode{cond: cond, then: then}
	switch stop {
	case "</#if>":
		return n, nil
	case "<#else>":
		alt, stop, err := p.parse([]string{"</#if>"})
		if err != nil {
			return ifNode{}, err
		}
		if stop != "</#if>" {
			return ifNode{}, fmt.Errorf("unclosed if")
		}
		n.alt = alt
		return n, nil
	case "<#elseif ":
		rest := p.s[p.i:]
		end := strings.IndexByte(rest, '>')
		if end < 0 {
			return ifNode{}, fmt.Errorf("unterminated elseif")
		}
		inner := strings.TrimSpace(strings.TrimPrefix(rest[:end], "elseif"))
		p.i += end + 1
		nested, err := p.parseIf(inner)
		if err != nil {
			return ifNode{}, err
		}
		n.alt = []node{nested}
		return n, nil
	default:
		return ifNode{}, fmt.Errorf("unclosed if")
	}
}

func (p *ft) parseList(fields []string) (listNode, error) {
	if len(fields) != 4 || fields[2] != "as" {
		return listNode{}, fmt.Errorf("list syntax is <#list expr as item>")
	}
	body, stop, err := p.parse([]string{"</#list>"})
	if err != nil {
		return listNode{}, err
	}
	if stop != "</#list>" {
		return listNode{}, fmt.Errorf("unclosed list")
	}
	return listNode{expr: fields[1], item: fields[3], body: body}, nil
}

func exec(b *strings.Builder, nodes []node, scope []map[string]any) error {
	for _, n := range nodes {
		switch item := n.(type) {
		case textNode:
			b.WriteString(item.text)
		case interpNode:
			val, err := eval(item.expr, scope)
			if err != nil {
				return err
			}
			b.WriteString(stringify(val))
		case assignNode:
			val, err := eval(item.expr, scope)
			if err != nil {
				return err
			}
			scope[len(scope)-1][item.name] = val
		case ifNode:
			ok, err := truthy(item.cond, scope)
			if err != nil {
				return err
			}
			branch := item.alt
			if ok {
				branch = item.then
			}
			if err := exec(b, branch, scope); err != nil {
				return err
			}
		case listNode:
			val, err := eval(item.expr, scope)
			if err != nil {
				return err
			}
			seq, ok := val.([]any)
			if !ok {
				return fmt.Errorf("%s is not a list", item.expr)
			}
			for _, el := range seq {
				child := make([]map[string]any, len(scope)+1)
				copy(child, scope)
				child[len(scope)] = map[string]any{item.item: el}
				if err := exec(b, item.body, child); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("unknown template node")
		}
	}
	return nil
}

func truthy(expr string, scope []map[string]any) (bool, error) {
	val, err := eval(expr, scope)
	if err != nil {
		return false, err
	}
	switch v := val.(type) {
	case nil:
		return false, nil
	case bool:
		return v, nil
	case string:
		return v != "", nil
	case []any:
		return len(v) > 0, nil
	default:
		return true, nil
	}
}

func eval(expr string, scope []map[string]any) (any, error) {
	expr = strings.TrimSpace(expr)
	if expr == "true" {
		return true, nil
	}
	if expr == "false" {
		return false, nil
	}
	if len(expr) >= 2 && (expr[0] == '"' || expr[0] == '\'') {
		s, err := strconv.Unquote(normalizeQuote(expr))
		if err != nil {
			return nil, fmt.Errorf("bad string %s", expr)
		}
		return s, nil
	}
	if left, right, ok := cutCompare(expr, "=="); ok {
		return compare(left, right, scope, true)
	}
	if left, right, ok := cutCompare(expr, "!="); ok {
		return compare(left, right, scope, false)
	}
	if strings.HasSuffix(expr, "??") {
		val, ok := lookup(scope, strings.TrimSpace(strings.TrimSuffix(expr, "??")))
		return ok && val != nil, nil
	}
	if path, def, ok := strings.Cut(expr, "!"); ok && !strings.Contains(path, " ") {
		if val, found := lookup(scope, strings.TrimSpace(path)); found && val != nil {
			return val, nil
		}
		return eval(def, scope)
	}
	path := expr
	suffix := ""
	if before, after, ok := strings.Cut(expr, "?"); ok {
		path, suffix = before, after
	}
	val, found := lookup(scope, strings.TrimSpace(path))
	if !found {
		return nil, fmt.Errorf("unknown name %s", path)
	}
	switch suffix {
	case "":
		return val, nil
	case "size":
		switch seq := val.(type) {
		case []any:
			return len(seq), nil
		case string:
			return len(seq), nil
		case nil:
			return 0, nil
		default:
			return nil, fmt.Errorf("%s?size is not a sequence", path)
		}
	case "has_content":
		return truthyValue(val), nil
	default:
		return nil, fmt.Errorf("unsupported builtin ?%s", suffix)
	}
}

func compare(left, right string, scope []map[string]any, eq bool) (bool, error) {
	lv, err := eval(left, scope)
	if err != nil {
		return false, err
	}
	rv, err := eval(right, scope)
	if err != nil {
		return false, err
	}
	same := stringify(lv) == stringify(rv)
	if eq {
		return same, nil
	}
	return !same, nil
}

func cutCompare(expr, op string) (string, string, bool) {
	i := strings.Index(expr, op)
	if i < 0 {
		return "", "", false
	}
	return expr[:i], expr[i+len(op):], true
}

func lookup(scope []map[string]any, path string) (any, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, false
	}
	parts := strings.Split(path, ".")
	var cur any
	found := false
	for i := len(scope) - 1; i >= 0; i-- {
		if v, ok := scope[i][parts[0]]; ok {
			cur = v
			found = true
			break
		}
	}
	if !found {
		return nil, false
	}
	for _, part := range parts[1:] {
		obj, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		cur, ok = obj[part]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

func truthyValue(val any) bool {
	switch v := val.(type) {
	case nil:
		return false
	case bool:
		return v
	case string:
		return v != ""
	case []any:
		return len(v) > 0
	default:
		return true
	}
}

func stringify(val any) string {
	switch v := val.(type) {
	case nil:
		return ""
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprint(v)
	}
}

func normalizeQuote(expr string) string {
	if expr[0] == '\'' && strings.HasSuffix(expr, "'") {
		return strconv.Quote(expr[1 : len(expr)-1])
	}
	return expr
}
