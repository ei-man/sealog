package main

import (
	"fmt"
	"regexp"
	"strings"
)

type QueryError struct {
	Message  string `json:"message"`
	Position int    `json:"position"`
}

func (e *QueryError) Error() string {
	return fmt.Sprintf("%s at position %d", e.Message, e.Position+1)
}

type tokenType string

const (
	tokenIdentifier  tokenType = "identifier"
	tokenString      tokenType = "string"
	tokenEquals      tokenType = "equals"
	tokenRegexEquals tokenType = "regexEquals"
	tokenAnd         tokenType = "and"
	tokenOr          tokenType = "or"
	tokenNot         tokenType = "not"
	tokenLeftParen   tokenType = "leftParen"
	tokenRightParen  tokenType = "rightParen"
	tokenEOF         tokenType = "eof"
)

type token struct {
	typ      tokenType
	value    string
	position int
}

func tokenizeQuery(input string) ([]token, *QueryError) {
	tokens := make([]token, 0)
	index := 0

	for index < len(input) {
		char := input[index]

		if isWhitespace(char) {
			index++
			continue
		}

		switch char {
		case '(':
			tokens = append(tokens, token{typ: tokenLeftParen, value: "(", position: index})
			index++
			continue
		case ')':
			tokens = append(tokens, token{typ: tokenRightParen, value: ")", position: index})
			index++
			continue
		case '=':
			tokens = append(tokens, token{typ: tokenEquals, value: "=", position: index})
			index++
			continue
		case '~':
			if index+1 < len(input) && input[index+1] == '=' {
				tokens = append(tokens, token{typ: tokenRegexEquals, value: "~=", position: index})
				index += 2
				continue
			}
			return nil, &QueryError{Message: `Unexpected character "~"`, Position: index}
		case '"':
			stringToken, next, err := readQuotedString(input, index)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, stringToken)
			index = next
			continue
		}

		start := index
		var builder strings.Builder
		for index < len(input) && !isWhitespace(input[index]) && !isReservedQueryChar(input[index]) {
			builder.WriteByte(input[index])
			index++
		}
		value := builder.String()
		if value == "" {
			return nil, &QueryError{Message: fmt.Sprintf(`Unexpected character "%c"`, char), Position: index}
		}

		tokens = append(tokens, token{
			typ:      keywordTokenType(value),
			value:    value,
			position: start,
		})
	}

	tokens = append(tokens, token{typ: tokenEOF, position: len(input)})
	return tokens, nil
}

func readQuotedString(input string, start int) (token, int, *QueryError) {
	index := start + 1
	var builder strings.Builder
	for index < len(input) {
		current := input[index]
		if current == '\\' {
			if index+1 >= len(input) {
				return token{}, index, &QueryError{Message: "Unterminated escape sequence", Position: index}
			}
			builder.WriteByte(input[index+1])
			index += 2
			continue
		}
		if current == '"' {
			return token{typ: tokenString, value: builder.String(), position: start}, index + 1, nil
		}
		builder.WriteByte(current)
		index++
	}
	return token{}, index, &QueryError{Message: "Unterminated quoted string", Position: start}
}

func isWhitespace(char byte) bool {
	return char == ' ' || char == '\t' || char == '\n' || char == '\r'
}

func isReservedQueryChar(char byte) bool {
	return char == '(' || char == ')' || char == '=' || char == '~' || char == '"'
}

func keywordTokenType(value string) tokenType {
	switch strings.ToUpper(value) {
	case "AND":
		return tokenAnd
	case "OR":
		return tokenOr
	case "NOT":
		return tokenNot
	default:
		return tokenIdentifier
	}
}

type QueryNode interface {
	queryNode()
}

type ConditionNode struct {
	Field    string
	Operator string
	Value    string
	Regex    *regexp.Regexp
}

func (*ConditionNode) queryNode() {}

type AndNode struct {
	Left  QueryNode
	Right QueryNode
}

func (*AndNode) queryNode() {}

type OrNode struct {
	Left  QueryNode
	Right QueryNode
}

func (*OrNode) queryNode() {}

type NotNode struct {
	Expr QueryNode
}

func (*NotNode) queryNode() {}

type queryParser struct {
	tokens []token
	index  int
}

func ParseQuery(input string) (QueryNode, *QueryError) {
	tokens, err := tokenizeQuery(input)
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 || tokens[0].typ == tokenEOF {
		return nil, &QueryError{Message: "Enter a filter expression", Position: 0}
	}
	parser := queryParser{tokens: tokens}
	return parser.parse()
}

func (p *queryParser) parse() (QueryNode, *QueryError) {
	expr, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if token := p.peek(); token.typ != tokenEOF {
		return nil, &QueryError{Message: "Unexpected token after query", Position: token.position}
	}
	return expr, nil
}

func (p *queryParser) parseOr() (QueryNode, *QueryError) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.match(tokenOr) {
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &OrNode{Left: left, Right: right}
	}
	return left, nil
}

func (p *queryParser) parseAnd() (QueryNode, *QueryError) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for p.match(tokenAnd) {
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &AndNode{Left: left, Right: right}
	}
	return left, nil
}

func (p *queryParser) parseUnary() (QueryNode, *QueryError) {
	if p.match(tokenNot) {
		expr, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &NotNode{Expr: expr}, nil
	}
	return p.parsePrimary()
}

func (p *queryParser) parsePrimary() (QueryNode, *QueryError) {
	if p.match(tokenLeftParen) {
		expr, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(tokenRightParen, "Expected closing parenthesis"); err != nil {
			return nil, err
		}
		return expr, nil
	}
	return p.parseCondition()
}

func (p *queryParser) parseCondition() (QueryNode, *QueryError) {
	field, err := p.expect(tokenIdentifier, "Expected field name")
	if err != nil {
		return nil, err
	}

	operator := "="
	if p.match(tokenRegexEquals) {
		operator = "~="
	} else if _, err := p.expect(tokenEquals, "Expected = or ~="); err != nil {
		return nil, err
	}

	value := p.peek()
	if !isValueToken(value.typ) {
		return nil, &QueryError{Message: "Expected filter value", Position: value.position}
	}
	p.advance()

	condition := &ConditionNode{
		Field:    field.value,
		Operator: operator,
		Value:    value.value,
	}
	if operator == "~=" {
		regex, regexErr := regexp.Compile("(?i)" + value.value)
		if regexErr != nil {
			return nil, &QueryError{Message: regexErr.Error(), Position: value.position}
		}
		condition.Regex = regex
	}
	return condition, nil
}

func isValueToken(typ tokenType) bool {
	return typ == tokenIdentifier || typ == tokenString || typ == tokenAnd || typ == tokenOr || typ == tokenNot
}

func (p *queryParser) match(typ tokenType) bool {
	if p.peek().typ != typ {
		return false
	}
	p.advance()
	return true
}

func (p *queryParser) expect(typ tokenType, message string) (token, *QueryError) {
	token := p.peek()
	if token.typ != typ {
		return token, &QueryError{Message: message, Position: token.position}
	}
	return p.advance(), nil
}

func (p *queryParser) peek() token {
	if p.index >= len(p.tokens) {
		return token{typ: tokenEOF}
	}
	return p.tokens[p.index]
}

func (p *queryParser) advance() token {
	token := p.peek()
	p.index++
	return token
}

func EvaluateQuery(node QueryNode, fields map[string]any) bool {
	if node == nil {
		return true
	}
	switch typed := node.(type) {
	case *AndNode:
		return EvaluateQuery(typed.Left, fields) && EvaluateQuery(typed.Right, fields)
	case *OrNode:
		return EvaluateQuery(typed.Left, fields) || EvaluateQuery(typed.Right, fields)
	case *NotNode:
		return !EvaluateQuery(typed.Expr, fields)
	case *ConditionNode:
		value, ok := fields[typed.Field]
		if !ok {
			return false
		}
		text := valueToText(value)
		if typed.Operator == "~=" {
			return typed.Regex != nil && typed.Regex.MatchString(text)
		}
		return strings.EqualFold(text, typed.Value)
	default:
		return false
	}
}

func QueryToKQL(node QueryNode) string {
	if node == nil {
		return ""
	}
	switch typed := node.(type) {
	case *AndNode:
		return "(" + QueryToKQL(typed.Left) + " and " + QueryToKQL(typed.Right) + ")"
	case *OrNode:
		return "(" + QueryToKQL(typed.Left) + " or " + QueryToKQL(typed.Right) + ")"
	case *NotNode:
		return "not(" + QueryToKQL(typed.Expr) + ")"
	case *ConditionNode:
		expr := kqlFieldExpr(typed.Field)
		if typed.Operator == "~=" {
			return expr + " matches regex " + kqlString("(?i)"+typed.Value)
		}
		return expr + " =~ " + kqlString(typed.Value)
	default:
		return ""
	}
}

var traceColumnExpressions = map[string]string{
	"timestamp":              "tostring(timestamp)",
	"message":                "message",
	"severityLevel":          "tostring(severityLevel)",
	"operation_Id":           "operation_Id",
	"operation_Name":         "operation_Name",
	"operation_ParentId":     "operation_ParentId",
	"cloud_RoleName":         "cloud_RoleName",
	"cloud_RoleInstance":     "cloud_RoleInstance",
	"itemId":                 "itemId",
	"itemType":               "itemType",
	"appName":                "appName",
	"client_Type":            "client_Type",
	"client_City":            "client_City",
	"client_StateOrProvince": "client_StateOrProvince",
	"client_CountryOrRegion": "client_CountryOrRegion",
}

func kqlFieldExpr(field string) string {
	if expr, ok := traceColumnExpressions[field]; ok {
		return expr
	}

	parts := strings.Split(field, ".")
	if len(parts) > 0 && parts[0] == "customDimensions" {
		parts = parts[1:]
	}
	if len(parts) == 0 {
		return `tostring(customDimensions)`
	}
	if len(parts) == 1 {
		return "tostring(customDimensions[" + kqlString(parts[0]) + "])"
	}

	expr := "customDimensions[" + kqlString(parts[0]) + "]"
	for _, part := range parts[1:] {
		expr = "parse_json(tostring(" + expr + "))[" + kqlString(part) + "]"
	}
	direct := "tostring(customDimensions[" + kqlString(strings.Join(parts, ".")) + "])"
	nested := "tostring(" + expr + ")"
	return "iff(isnotempty(" + direct + "), " + direct + ", " + nested + ")"
}

func kqlString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
