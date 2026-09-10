package token

import (
	"fmt"
)

// Position 表示源代码中的位置（文件名、行号、列号）
type Position struct {
	Filename string
	Line     int
	Column   int
}

func (p Position) String() string {
	if p.Filename != "" {
		return fmt.Sprintf("%s:%d:%d", p.Filename, p.Line, p.Column)
	}
	return fmt.Sprintf("%d:%d", p.Line, p.Column)
}

// TokenType 代表词法单元类型
type TokenType string

const (
	// 特殊标记
	ILLEGAL TokenType = "ILLEGAL"
	EOF     TokenType = "EOF"

	// 标识符与字面量
	IDENT  TokenType = "IDENT"
	INT    TokenType = "INT"
	FLOAT  TokenType = "FLOAT"
	STRING TokenType = "STRING"
	BOOL   TokenType = "BOOL"

	// 算术运算符
	ADD TokenType = "+"
	SUB TokenType = "-"
	MUL TokenType = "*"
	QUO TokenType = "/"
	REM TokenType = "%"

	// 比较运算符
	EQL TokenType = "=="
	NEQ TokenType = "!="
	LSS TokenType = "<"
	LEQ TokenType = "<="
	GTR TokenType = ">"
	GEQ TokenType = ">="

	// 逻辑与借用运算符
	LAND TokenType = "&&"
	LOR  TokenType = "||"
	NOT  TokenType = "!"
	AMP  TokenType = "&" // Rust-style 借用/引用 (&x)

	// 赋值与声明
	ASSIGN     TokenType = "="
	DEFINE     TokenType = ":="
	ADD_ASSIGN TokenType = "+="
	SUB_ASSIGN TokenType = "-="
	MUL_ASSIGN TokenType = "*="
	QUO_ASSIGN TokenType = "/="

	// 分隔符与标点
	LPAREN    TokenType = "("
	RPAREN    TokenType = ")"
	LBRACE    TokenType = "{"
	RBRACE    TokenType = "}"
	LBRACKET  TokenType = "["
	RBRACKET  TokenType = "]"
	COMMA     TokenType = ","
	SEMICOLON TokenType = ";"
	COLON     TokenType = ":"
	DOT       TokenType = "."
	ARROW     TokenType = "->"

	// 核心关键字 (Lcode 自研语法)
	FN       TokenType = "fn"
	FUNC     TokenType = "func"
	LET      TokenType = "let"
	MUT      TokenType = "mut"
	VAR      TokenType = "var"
	CONST    TokenType = "const"
	IF       TokenType = "if"
	ELSE     TokenType = "else"
	WHILE    TokenType = "while"
	FOR      TokenType = "for"
	RETURN   TokenType = "return"
	BREAK    TokenType = "break"
	CONTINUE TokenType = "continue"
	TRUE     TokenType = "true"
	FALSE    TokenType = "false"
	NIL      TokenType = "nil"
	IMPORT   TokenType = "import"
	AS       TokenType = "as"
	IN       TokenType = "in"
	TRY      TokenType = "try"
	CATCH    TokenType = "catch"
)

// Token 结构体表示一个具体的词法单元
type Token struct {
	Type    TokenType
	Literal string
	Pos     Position
}

func (t Token) String() string {
	return fmt.Sprintf("[%s] '%s' at %s", t.Type, t.Literal, t.Pos)
}

// keywords 映射关键字字面量到其 TokenType
var keywords = map[string]TokenType{
	"fn":       FN,
	"fun":      FN,
	"func":     FUNC,
	"let":      LET,
	"mut":      MUT,
	"var":      VAR,
	"const":    CONST,
	"if":       IF,
	"else":     ELSE,
	"while":    WHILE,
	"for":      FOR,
	"return":   RETURN,
	"ret":      RETURN,
	"break":    BREAK,
	"continue": CONTINUE,
	"true":     TRUE,
	"false":    FALSE,
	"nil":      NIL,
	"import":   IMPORT,
	"imp":      IMPORT,
	"as":       AS,
	"in":       IN,
	"try":      TRY,
	"catch":    CATCH,
}

// LookupIdent 检查标识符是否为关键字
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
