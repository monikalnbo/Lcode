package parser

import (
	"fmt"
	"strconv"

	"lcode/pkg/ast"
	"lcode/pkg/lexer"
	"lcode/pkg/token"
)

// 运算符优先级常量定义
const (
	_ int = iota
	LOWEST
	LOGICAL_OR  // ||
	LOGICAL_AND // &&
	EQUALS      // == !=
	LESSGREATER // > < >= <=
	SUM         // + -
	PRODUCT     // * / %
	PREFIX      // -x !x
	CALL        // fn(x)
)

var precedences = map[token.TokenType]int{
	token.LOR:         LOGICAL_OR,
	token.LAND:        LOGICAL_AND,
	token.EQL:         EQUALS,
	token.NEQ:         EQUALS,
	token.LSS:         LESSGREATER,
	token.LEQ:         LESSGREATER,
	token.GTR:         LESSGREATER,
	token.GEQ:         LESSGREATER,
	token.ADD:         SUM,
	token.SUB:         SUM,
	token.MUL:         PRODUCT,
	token.QUO:         PRODUCT,
	token.REM:         PRODUCT,
	token.LPAREN:      CALL,
}

type (
	prefixParseFn func() ast.Expr
	infixParseFn  func(ast.Expr) ast.Expr
)

// Parser 语法分析器
type Parser struct {
	l         *lexer.Lexer
	curToken  token.Token
	peekToken token.Token
	errors    []string

	prefixParseFns map[token.TokenType]prefixParseFn
	infixParseFns  map[token.TokenType]infixParseFn
}

// New 创建语法分析器
func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:              l,
		errors:         make([]string, 0),
		prefixParseFns: make(map[token.TokenType]prefixParseFn),
		infixParseFns:  make(map[token.TokenType]infixParseFn),
	}

	// 注册前缀解析函数
	p.registerPrefix(token.IDENT, p.parseIdentifier)
	p.registerPrefix(token.INT, p.parseIntegerLiteral)
	p.registerPrefix(token.FLOAT, p.parseFloatLiteral)
	p.registerPrefix(token.STRING, p.parseStringLiteral)
	p.registerPrefix(token.BOOL, p.parseBooleanLiteral)
	p.registerPrefix(token.NOT, p.parsePrefixExpr)
	p.registerPrefix(token.SUB, p.parsePrefixExpr)
	p.registerPrefix(token.AMP, p.parsePrefixExpr)
	p.registerPrefix(token.MUL, p.parsePrefixExpr)
	p.registerPrefix(token.LPAREN, p.parseGroupedExpr)

	// 注册中缀解析函数
	p.registerInfix(token.ADD, p.parseInfixExpr)
	p.registerInfix(token.SUB, p.parseInfixExpr)
	p.registerInfix(token.MUL, p.parseInfixExpr)
	p.registerInfix(token.QUO, p.parseInfixExpr)
	p.registerInfix(token.REM, p.parseInfixExpr)
	p.registerInfix(token.EQL, p.parseInfixExpr)
	p.registerInfix(token.NEQ, p.parseInfixExpr)
	p.registerInfix(token.LSS, p.parseInfixExpr)
	p.registerInfix(token.LEQ, p.parseInfixExpr)
	p.registerInfix(token.GTR, p.parseInfixExpr)
	p.registerInfix(token.GEQ, p.parseInfixExpr)
	p.registerInfix(token.LAND, p.parseInfixExpr)
	p.registerInfix(token.LOR, p.parseInfixExpr)
	p.registerInfix(token.LPAREN, p.parseCallExpr)

	// 初始化 curToken 和 peekToken
	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) AppendError(msg string) {
	p.errors = append(p.errors, msg)
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

func (p *Parser) peekError(t token.TokenType) {
	msg := fmt.Sprintf("%s: 语法错误，期望下一个标记为 %s, 实际为 %s (字面量: %q)",
		p.peekToken.Pos, t, p.peekToken.Type, p.peekToken.Literal)
	p.errors = append(p.errors, msg)
}

func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) registerPrefix(tokenType token.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType token.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

// ParseProgram 解析整个文件程序
func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{
		Imports: make([]*ast.ImportDecl, 0),
		Decls:   make([]ast.Decl, 0),
		Stmts:   make([]ast.Stmt, 0),
	}

	for !p.curTokenIs(token.EOF) {
		if p.curTokenIs(token.IMPORT) {
			imp := p.parseImportDecl()
			if imp != nil {
				program.Imports = append(program.Imports, imp)
			}
		} else if p.curTokenIs(token.FN) || p.curTokenIs(token.FUNC) {
			decl := p.parseFuncDecl()
			if decl != nil {
				program.Decls = append(program.Decls, decl)
			}
		} else {
			stmt := p.parseStatement()
			if stmt != nil {
				program.Stmts = append(program.Stmts, stmt)
			}
		}
		p.nextToken()
	}

	return program
}

// parseImportDecl 解析 import "path" [as alias];
func (p *Parser) parseImportDecl() *ast.ImportDecl {
	decl := &ast.ImportDecl{Token: p.curToken}

	if !p.expectPeek(token.STRING) {
		return nil
	}
	decl.Path = p.curToken.Literal

	// 可选别名: import "path" as alias;
	if p.peekTokenIs(token.AS) {
		p.nextToken() // 吃掉 as
		if !p.expectPeek(token.IDENT) {
			return nil
		}
		decl.Alias = p.curToken.Literal
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return decl
}

// parseStatement 解析一条语句
func (p *Parser) parseStatement() ast.Stmt {
	switch p.curToken.Type {
	case token.LET, token.MUT, token.VAR, token.CONST:
		return p.parseVarDeclStmt()
	case token.RETURN:
		return p.parseReturnStmt()
	case token.IF:
		return p.parseIfStmt()
	case token.WHILE:
		return p.parseWhileStmt()
	case token.LBRACE:
		return p.parseBlockStmt()
	default:
		return p.parseExprOrAssignStmt()
	}
}

// parseFuncDecl 解析函数声明: func name(p1: T1, p2: T2) -> RetType { ... }
func (p *Parser) parseFuncDecl() *ast.FuncDecl {
	decl := &ast.FuncDecl{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		return nil
	}
	decl.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	decl.Params = p.parseFuncParams()

	// 可选返回类型标注 -> Type
	if p.peekTokenIs(token.ARROW) {
		p.nextToken() // 吃掉 ->
		if !p.expectPeek(token.IDENT) {
			return nil
		}
		decl.ReturnType = &ast.TypeAnnotation{
			Token: p.curToken,
			Name:  p.curToken.Literal,
		}
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	decl.Body = p.parseBlockStmt()

	return decl
}

func (p *Parser) parseFuncParams() []*ast.Param {
	params := make([]*ast.Param, 0)

	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return params
	}

	p.nextToken() // 移到第一个形参名

	param := &ast.Param{Name: &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}}
	if p.peekTokenIs(token.COLON) {
		p.nextToken() // 吃掉 :
		if p.expectPeek(token.IDENT) {
			param.Type = &ast.TypeAnnotation{Token: p.curToken, Name: p.curToken.Literal}
		}
	}
	params = append(params, param)

	for p.peekTokenIs(token.COMMA) {
		p.nextToken() // 吃掉 ,
		p.nextToken() // 移到下一个形参名
		prm := &ast.Param{Name: &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}}
		if p.peekTokenIs(token.COLON) {
			p.nextToken() // 吃掉 :
			if p.expectPeek(token.IDENT) {
				prm.Type = &ast.TypeAnnotation{Token: p.curToken, Name: p.curToken.Literal}
			}
		}
		params = append(params, prm)
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return params
}

// parseVarDeclStmt 解析变量定义: let x: int = 10; 或 mut x = 5;
func (p *Parser) parseVarDeclStmt() *ast.VarDeclStmt {
	stmt := &ast.VarDeclStmt{
		Token:   p.curToken,
		IsMut:   p.curTokenIs(token.MUT) || p.curTokenIs(token.VAR),
		IsConst: p.curTokenIs(token.CONST) || p.curTokenIs(token.LET),
	}

	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// 检查可选类型标注 : Type
	if p.peekTokenIs(token.COLON) {
		p.nextToken() // 吃掉 :
		if !p.expectPeek(token.IDENT) {
			return nil
		}
		stmt.Type = &ast.TypeAnnotation{Token: p.curToken, Name: p.curToken.Literal}
	}

	// 检查初始赋值 =
	if p.peekTokenIs(token.ASSIGN) {
		p.nextToken() // 吃掉 =
		p.nextToken() // 移到表达式起始
		stmt.Value = p.parseExpression(LOWEST)
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseReturnStmt 解析 return [expr];
func (p *Parser) parseReturnStmt() *ast.ReturnStmt {
	stmt := &ast.ReturnStmt{Token: p.curToken}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
		return stmt
	}

	p.nextToken()
	stmt.ReturnValue = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseIfStmt 解析 if 语句
func (p *Parser) parseIfStmt() *ast.IfStmt {
	stmt := &ast.IfStmt{Token: p.curToken}

	p.nextToken() // 移到条件表达式
	stmt.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	stmt.Consequence = p.parseBlockStmt()

	if p.peekTokenIs(token.ELSE) {
		p.nextToken() // 吃掉 else
		if p.peekTokenIs(token.IF) {
			p.nextToken() // 移到 if
			stmt.Alternative = p.parseIfStmt()
		} else if p.peekTokenIs(token.LBRACE) {
			p.nextToken() // 移到 {
			stmt.Alternative = p.parseBlockStmt()
		}
	}

	return stmt
}

// parseWhileStmt 解析 while 循环
func (p *Parser) parseWhileStmt() *ast.WhileStmt {
	stmt := &ast.WhileStmt{Token: p.curToken}

	p.nextToken() // 移到条件
	stmt.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	stmt.Body = p.parseBlockStmt()

	return stmt
}

// parseBlockStmt 解析代码块 { ... }
func (p *Parser) parseBlockStmt() *ast.BlockStmt {
	block := &ast.BlockStmt{
		Token:      p.curToken,
		Statements: make([]ast.Stmt, 0),
	}

	p.nextToken()

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	return block
}

// parseExprOrAssignStmt 解析赋值或表达式语句
func (p *Parser) parseExprOrAssignStmt() ast.Stmt {
	startTok := p.curToken
	expr := p.parseExpression(LOWEST)
	if expr == nil {
		return nil
	}

	// 赋值语句: left = right 或 left += right
	if p.peekTokenIs(token.ASSIGN) || p.peekTokenIs(token.ADD_ASSIGN) ||
		p.peekTokenIs(token.SUB_ASSIGN) || p.peekTokenIs(token.MUL_ASSIGN) ||
		p.peekTokenIs(token.QUO_ASSIGN) {
		p.nextToken()
		op := p.curToken.Literal
		p.nextToken()
		right := p.parseExpression(LOWEST)
		stmt := &ast.AssignStmt{
			Left:     expr,
			Operator: op,
			Right:    right,
		}
		if p.peekTokenIs(token.SEMICOLON) {
			p.nextToken()
		}
		return stmt
	}

	stmt := &ast.ExprStmt{Token: startTok, Expression: expr}
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

// parseExpression 普拉特算法解析表达式
func (p *Parser) parseExpression(precedence int) ast.Expr {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExp := prefix()

	for !p.peekTokenIs(token.SEMICOLON) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()
		leftExp = infix(leftExp)
	}

	return leftExp
}

func (p *Parser) parseIdentifier() ast.Expr {
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseIntegerLiteral() ast.Expr {
	lit := &ast.IntegerLiteral{Token: p.curToken}
	val, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		p.errors = append(p.errors, fmt.Sprintf("%s: 无法将 %q 解析为整型", p.curToken.Pos, p.curToken.Literal))
		return nil
	}
	lit.Value = val
	return lit
}

func (p *Parser) parseFloatLiteral() ast.Expr {
	lit := &ast.FloatLiteral{Token: p.curToken}
	val, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		p.errors = append(p.errors, fmt.Sprintf("%s: 无法将 %q 解析为浮点型", p.curToken.Pos, p.curToken.Literal))
		return nil
	}
	lit.Value = val
	return lit
}

func (p *Parser) parseStringLiteral() ast.Expr {
	return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseBooleanLiteral() ast.Expr {
	return &ast.BooleanLiteral{Token: p.curToken, Value: p.curToken.Literal == "true"}
}

func (p *Parser) parsePrefixExpr() ast.Expr {
	expr := &ast.UnaryExpr{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}

	p.nextToken()
	expr.Right = p.parseExpression(PREFIX)
	return expr
}

func (p *Parser) parseGroupedExpr() ast.Expr {
	tok := p.curToken
	p.nextToken()

	exp := p.parseExpression(LOWEST)
	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return &ast.GroupedExpr{Token: tok, Expression: exp}
}

func (p *Parser) parseInfixExpr(left ast.Expr) ast.Expr {
	expr := &ast.BinaryExpr{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expr.Right = p.parseExpression(precedence)

	return expr
}

func (p *Parser) parseCallExpr(function ast.Expr) ast.Expr {
	expr := &ast.CallExpr{Token: p.curToken, Function: function}
	expr.Arguments = p.parseCallArguments()
	return expr
}

func (p *Parser) parseCallArguments() []ast.Expr {
	args := make([]ast.Expr, 0)

	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return args
	}

	p.nextToken()
	args = append(args, p.parseExpression(LOWEST))

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		args = append(args, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return args
}

func (p *Parser) noPrefixParseFnError(t token.TokenType) {
	msg := fmt.Sprintf("%s: 遇到未预料的标记 %s (字面量: %q)", p.curToken.Pos, t, p.curToken.Literal)
	p.errors = append(p.errors, msg)
}
