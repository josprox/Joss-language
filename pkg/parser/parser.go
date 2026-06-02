package parser

import (
	"fmt"
)

const (
	_ int = iota
	LOWEST
	ASSIGNMENT  // =
	TERNARY     // ? :
	COALESCE    // ??
	LOGICAL     // && or ||
	EQUALS      // ==
	LESSGREATER // > or <
	PIPE_OP     // |>
	SUM         // +
	SHIFT       // << or >>
	PRODUCT     // *
	MODULO      // %
	PREFIX      // -X or !X
	CALL        // myFunction(X)
	INDEX       // array[index]
)

var precedences = map[TokenType]int{
	ASSIGN:        ASSIGNMENT,
	QUESTION:      TERNARY,
	NULL_COALESCE: COALESCE,
	PIPE:          PIPE_OP,
	PLUS:          SUM,
	MINUS:         SUM,
	DOT:           SUM,
	SLASH:         PRODUCT,
	ASTERISK:      PRODUCT,
	PERCENT:       MODULO,
	AND:           LOGICAL,
	OR:            LOGICAL,
	LT:            LESSGREATER,
	GT:            LESSGREATER,
	EQ:            EQUALS,
	NOT_EQ:        EQUALS,
	LTE:           LESSGREATER,
	GTE:           LESSGREATER,
	SHIFT_LEFT:    SHIFT,
	SHIFT_RIGHT:   SHIFT,
	LPAREN:        CALL,
	LBRACKET:      INDEX,
	ARROW:         INDEX,
	DOUBLE_COLON:  INDEX,
	INCREMENT:     INDEX,
}

type (
	prefixParseFn func() Expression
	infixParseFn  func(Expression) Expression
)

type Parser struct {
	l         *Lexer
	curToken  Token
	peekToken Token
	errors    []string

	prefixParseFns map[TokenType]prefixParseFn
	infixParseFns  map[TokenType]infixParseFn
}

func NewParser(l *Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}

	p.prefixParseFns = make(map[TokenType]prefixParseFn)
	p.registerPrefix(IDENT, p.parseIdentifier)
	p.registerPrefix(VAR, p.parseVarExpression) // Handle $name
	p.registerPrefix(INT, p.parseIntegerLiteral)
	p.registerPrefix(FLOAT, p.parseFloatLiteral)
	p.registerPrefix(STRING, p.parseStringLiteral)
	p.registerPrefix(TRUE, p.parseBoolean)
	p.registerPrefix(FALSE, p.parseBoolean)
	p.registerPrefix(LPAREN, p.parseGroupedExpression)
	p.registerPrefix(LBRACKET, p.parseArrayLiteral)
	p.registerPrefix(LBRACE, p.parseBraceExpression) // Maps { key: val } or Blocks { stmt; }
	p.registerPrefix(NEW, p.parseNewExpression)
	p.registerPrefix(THIS, p.parseIdentifier)
	p.registerPrefix(ISSET, p.parseIssetExpression)
	p.registerPrefix(EMPTY, p.parseEmptyExpression)
	p.registerPrefix(BANG, p.parsePrefixExpression)
	p.registerPrefix(MINUS, p.parsePrefixExpression)
	p.registerPrefix(FUNCTION, p.parseFunctionLiteral)
	p.registerPrefix(MATCH, p.parseMatchExpression)

	p.infixParseFns = make(map[TokenType]infixParseFn)
	p.registerInfix(PLUS, p.parseInfixExpression)
	p.registerInfix(MINUS, p.parseInfixExpression)
	p.registerInfix(SLASH, p.parseInfixExpression)
	p.registerInfix(ASTERISK, p.parseInfixExpression)
	p.registerInfix(PERCENT, p.parseInfixExpression)
	p.registerInfix(AND, p.parseInfixExpression)
	p.registerInfix(OR, p.parseInfixExpression)
	p.registerInfix(LT, p.parseInfixExpression)
	p.registerInfix(GT, p.parseInfixExpression)
	p.registerInfix(EQ, p.parseInfixExpression)
	p.registerInfix(NOT_EQ, p.parseInfixExpression)
	p.registerInfix(LTE, p.parseInfixExpression)
	p.registerInfix(GTE, p.parseInfixExpression)
	p.registerInfix(SHIFT_LEFT, p.parseInfixExpression)
	p.registerInfix(SHIFT_RIGHT, p.parseInfixExpression)
	p.registerInfix(PIPE, p.parseInfixExpression)
	p.registerInfix(DOT, p.parseInfixExpression)
	p.registerInfix(LPAREN, p.parseCallExpression)
	p.registerInfix(QUESTION, p.parseTernaryExpression)
	p.registerInfix(NULL_COALESCE, p.parseInfixExpression)
	p.registerInfix(LBRACKET, p.parseIndexExpression)
	p.registerInfix(ARROW, p.parseMemberExpression)
	p.registerInfix(DOUBLE_COLON, p.parseMemberExpression)
	p.registerInfix(ASSIGN, p.parseAssignExpression)
	p.registerInfix(INCREMENT, p.parsePostfixExpression)

	// Read two tokens, so curToken and peekToken are both set
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) ParseProgram() *Program {
	program := &Program{}
	program.Statements = []Statement{}

	for p.curToken.Type != EOF {
		if p.curToken.Type == NEWLINE {
			p.nextToken()
			continue
		}
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}
	return program
}

func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) noPrefixParseFnError(t TokenType) {
	msg := fmt.Sprintf("line %d: no prefix parse function for %s found", p.curToken.Line, t)
	p.errors = append(p.errors, msg)
}

func (p *Parser) expectPeek(t TokenType) bool {
	if p.peekToken.Type == t {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

func (p *Parser) peekError(t TokenType) {
	msg := fmt.Sprintf("line %d: expected next token to be %s, got %s instead", p.peekToken.Line, t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) peekTokenIs(t TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) curTokenIs(t TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) registerPrefix(tokenType TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}
