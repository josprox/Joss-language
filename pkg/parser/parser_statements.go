package parser

func (p *Parser) parseStatement() Statement {
	if p.curToken.Type == CLASS {
		return p.parseClassStatement()
	}
	if p.curToken.Type == INIT {
		return p.parseInitStatement()
	}
	if p.curToken.Type == FOREACH {
		return p.parseForeachStatement()
	}
	if p.curToken.Type == FUNCTION {
		return p.parseMethodStatement()
	}
	if p.curToken.Type == IMPORT {
		return p.parseImportStatement()
	}
	if p.curToken.Type == USE {
		return p.parseUseStatement()
	}
	if p.curToken.Type == ECHO || p.curToken.Type == PRINT {
		return p.parseEchoStatement()
	}

	if p.curToken.Type == WHILE {
		return p.parseWhileStatement()
	}
	if p.curToken.Type == DO {
		return p.parseDoWhileStatement()
	}
	if p.curToken.Type == TRY {
		return p.parseTryCatchStatement()
	}
	if p.curToken.Type == THROW {
		return p.parseThrowStatement()
	}
	if p.curToken.Type == RETURN {
		return p.parseReturnStatement()
	}
	if p.curToken.Type == BREAK {
		return p.parseBreakStatement()
	}
	if p.curToken.Type == CONTINUE {
		return p.parseContinueStatement()
	}
	// Check for variable declaration: type $name = value
	if p.curToken.Type == IDENT && p.peekToken.Type == VAR {
		return p.parseLetStatement()
	}
	// Check for Increment: $i++
	// $ is VAR, i is IDENT, ++ is INCREMENT
	// But parseStatement starts at current token.
	// If current is VAR ($), it might be expression statement or assignment.
	// If current is IDENT (variable name after $), wait.
	// In Joss, variables start with $.
	// So `$i` is `VAR` then `IDENT`.
	// `parseExpressionStatement` handles `$i`.
	// `parseExpression` handles `$i` (Identifier).
	// Then `parseExpressionStatement` checks for semicolon.
	// But `++` is postfix?
	// If we have `$i++`, `parseExpression` reads `$i`.
	// Then `peekToken` is `++`.
	// If `++` is registered as infix (postfix), it works.
	// But `++` is usually a statement or expression.
	// Let's register `++` as a postfix operator in `parser.go`?
	// Or handle it here.

	// If we register `++` as infix with high precedence, `$i ++` becomes `InfixExpression($i, ++, nil)`? No.
	// Postfix is `Left -> Operator`.
	// We don't have postfix support in `parser.go` loop yet.
	// Let's add it to `parseExpression` loop or handle as statement.

	// Simpler: Handle as statement if it appears at top level.
	// But it can be used in expression: `$x = $i++`.
	// So it must be an expression.

	// I need to register INCREMENT as a POSTFIX operator in `parser.go`.
	// `parser.go` loop: `infix := p.infixParseFns[p.peekToken.Type]`
	// I can register `INCREMENT` as infix, and the parse function will return `PostfixExpression`.

	return p.parseExpressionStatement()
}

func (p *Parser) parseReturnStatement() *ReturnStatement {
	stmt := &ReturnStatement{Token: p.curToken}

	p.nextToken()

	if p.curToken.Type == SEMICOLON {
		return stmt
	}

	stmt.ReturnValue = p.parseExpression(LOWEST)

	if p.peekToken.Type == SEMICOLON {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseBreakStatement() *BreakStatement {
	stmt := &BreakStatement{Token: p.curToken}

	if p.peekToken.Type == SEMICOLON {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseContinueStatement() *ContinueStatement {
	stmt := &ContinueStatement{Token: p.curToken}

	if p.peekToken.Type == SEMICOLON {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseClassStatement() *ClassStatement {
	stmt := &ClassStatement{Token: p.curToken}

	if !p.expectPeek(IDENT) {
		return nil
	}

	stmt.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if p.peekToken.Type == EXTENDS {
		p.nextToken()
		p.nextToken()
		stmt.SuperClass = &Identifier{Token: p.curToken, Value: p.curToken.Literal}
	}

	if !p.expectPeek(LBRACE) {
		return nil
	}

	stmt.Body = p.parseClassBody()

	return stmt
}

func (p *Parser) parseClassBody() *BlockStatement {
	block := &BlockStatement{Token: p.curToken}
	block.Statements = []Statement{}

	p.nextToken()

	for p.curToken.Type != RBRACE && p.curToken.Type != EOF {
		if p.curToken.Type == NEWLINE {
			p.nextToken()
			continue
		}

		var stmt Statement
		if p.curToken.Type == FUNCTION {
			stmt = p.parseMethodStatement()
		} else if p.curToken.Type == INIT {
			stmt = p.parseInitStatement()
		} else if p.curToken.Type == IDENT && p.peekToken.Type == VAR { // Property: string $x
			stmt = p.parseLetStatement()
		} else {
			// Skip or error? For now skip to avoid infinite loop if unknown
			p.nextToken()
			continue
		}

		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	return block
}

func (p *Parser) parseInitStatement() *InitStatement {
	stmt := &InitStatement{Token: p.curToken}

	if !p.expectPeek(IDENT) { // main
		return nil
	}
	stmt.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(LPAREN) {
		return nil
	}

	// Parse parameters
	stmt.Parameters = p.parseFunctionParameters()

	if !p.expectPeek(LBRACE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()

	return stmt
}

func (p *Parser) parseBlockStatement() *BlockStatement {
	block := &BlockStatement{Token: p.curToken}
	block.Statements = []Statement{}

	p.nextToken()

	for p.curToken.Type != RBRACE && p.curToken.Type != EOF {
		if p.curToken.Type == NEWLINE {
			p.nextToken()
			continue
		}
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	return block
}

func (p *Parser) parseLetStatement() Statement {
	typeToken := p.curToken

	if !p.expectPeek(VAR) {
		return nil
	}

	if !p.expectPeek(IDENT) {
		return nil
	}

	name := &Identifier{Token: p.curToken, Value: p.curToken.Literal}
	var value Expression

	if p.peekToken.Type == ASSIGN {
		p.nextToken()
		p.nextToken()
		value = p.parseExpression(LOWEST)
	}

	if p.peekToken.Type == COMMA {
		return p.parseMultiLetStatement(typeToken, name, value)
	}

	stmt := &LetStatement{Token: typeToken, Name: name, Value: value}

	if p.peekToken.Type == SEMICOLON || p.peekToken.Type == NEWLINE {
		p.nextToken()
	}

	return stmt
}

// parseMultiLetStatement parses: type $a[=val],$b[=val],...
// Call only when a COMMA is detected after the first var in a LetStatement.
func (p *Parser) parseMultiLetStatement(typeToken Token, firstName *Identifier, firstValue Expression) *MultiLetStatement {
	stmt := &MultiLetStatement{TypeToken: typeToken}

	// Add first declaration
	stmt.Declarations = append(stmt.Declarations, SingleDecl{Name: firstName, Value: firstValue})

	// Consume each comma-separated declaration
	for p.peekToken.Type == COMMA {
		p.nextToken() // consume COMMA
		if !p.expectPeek(VAR) {
			break
		}
		if !p.expectPeek(IDENT) {
			break
		}
		name := &Identifier{Token: p.curToken, Value: p.curToken.Literal}
		var value Expression
		if p.peekToken.Type == ASSIGN {
			p.nextToken() // consume =
			p.nextToken() // move to value
			value = p.parseExpression(LOWEST)
		}
		stmt.Declarations = append(stmt.Declarations, SingleDecl{Name: name, Value: value})
	}

	if p.peekToken.Type == SEMICOLON || p.peekToken.Type == NEWLINE {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseExpressionStatement() *ExpressionStatement {
	stmt := &ExpressionStatement{Token: p.curToken}
	stmt.Expression = p.parseExpression(LOWEST)

	if p.peekToken.Type == SEMICOLON || p.peekToken.Type == NEWLINE {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseForeachStatement() *ForeachStatement {
	stmt := &ForeachStatement{Token: p.curToken}

	if !p.expectPeek(LPAREN) {
		return nil
	}

	p.nextToken()
	stmt.Iterable = p.parseExpression(LOWEST)

	if !p.expectPeek(AS) {
		return nil
	}

	// Expect variable: $val
	// In parser, VAR is '$', then IDENT 'val'
	if !p.expectPeek(VAR) {
		return nil
	}
	if !p.expectPeek(IDENT) {
		return nil
	}
	stmt.Value = p.curToken.Literal

	if !p.expectPeek(RPAREN) {
		return nil
	}

	if !p.expectPeek(LBRACE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()

	return stmt
}

func (p *Parser) parseImportStatement() *ImportStatement {
	stmt := &ImportStatement{Token: p.curToken}

	if !p.expectPeek(STRING) {
		return nil
	}

	stmt.Path = p.curToken.Literal

	if p.peekToken.Type == SEMICOLON || p.peekToken.Type == NEWLINE {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseEchoStatement() *EchoStatement {
	stmt := &EchoStatement{Token: p.curToken}

	p.nextToken() // consume ECHO/PRINT

	// Optional parentheses: echo("foo")
	if p.curToken.Type == LPAREN {
		p.nextToken()
		stmt.Value = p.parseExpression(LOWEST)
		if p.peekToken.Type == RPAREN {
			p.nextToken()
		}
	} else {
		stmt.Value = p.parseExpression(LOWEST)
	}

	if p.peekToken.Type == SEMICOLON || p.peekToken.Type == NEWLINE {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseWhileStatement() *WhileStatement {
	stmt := &WhileStatement{Token: p.curToken}

	if !p.expectPeek(LPAREN) {
		return nil
	}

	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(RPAREN) {
		return nil
	}

	if !p.expectPeek(LBRACE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()

	return stmt
}

func (p *Parser) parseDoWhileStatement() *DoWhileStatement {
	stmt := &DoWhileStatement{Token: p.curToken}

	if !p.expectPeek(LBRACE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()

	if !p.expectPeek(WHILE) {
		return nil
	}

	if !p.expectPeek(LPAREN) {
		return nil
	}

	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(RPAREN) {
		return nil
	}

	if p.peekToken.Type == SEMICOLON {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseTryCatchStatement() *TryCatchStatement {
	stmt := &TryCatchStatement{Token: p.curToken}

	if !p.expectPeek(LBRACE) {
		return nil
	}

	stmt.TryBlock = p.parseBlockStatement()

	if !p.expectPeek(CATCH) {
		return nil
	}
	stmt.CatchToken = p.curToken

	if !p.expectPeek(LPAREN) {
		return nil
	}

	// Expect variable: $e
	if !p.expectPeek(VAR) {
		return nil
	}
	if !p.expectPeek(IDENT) {
		return nil
	}
	stmt.CatchVar = p.curToken.Literal

	if !p.expectPeek(RPAREN) {
		return nil
	}

	if !p.expectPeek(LBRACE) {
		return nil
	}

	stmt.CatchBlock = p.parseBlockStatement()

	return stmt
}

func (p *Parser) parseThrowStatement() *ThrowStatement {
	stmt := &ThrowStatement{Token: p.curToken}

	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	if p.peekToken.Type == SEMICOLON {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseMethodStatement() *MethodStatement {
	stmt := &MethodStatement{Token: p.curToken}

	if !p.expectPeek(IDENT) {
		return nil
	}
	stmt.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(LPAREN) {
		return nil
	}

	stmt.Parameters = p.parseFunctionParameters()

	if !p.expectPeek(LBRACE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()

	return stmt
}

func (p *Parser) parseUseStatement() *ImportStatement {
	stmt := &ImportStatement{Token: p.curToken}

	if !p.expectPeek(IDENT) {
		return nil
	}

	stmt.Path = "package:" + p.curToken.Literal

	if p.peekToken.Type == SEMICOLON || p.peekToken.Type == NEWLINE {
		p.nextToken()
	}

	return stmt
}
