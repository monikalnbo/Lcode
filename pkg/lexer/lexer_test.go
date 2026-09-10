package lexer

import (
	"testing"

	"lcode/pkg/token"
)

func TestNextToken(t *testing.T) {
	input := `
// 单行注释
/* 多行
   注释 */
let five = 5;
mut ten = 10.5;

fn add(x: int, y: int) -> int {
    return x + y;
}

mut result = add(five, ten);
if (result != 15) {
    print("hello \"world\"\n");
} else {
    while (five > 0) {
        result -= 1;
    }
}
`

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.LET, "let"},
		{token.IDENT, "five"},
		{token.ASSIGN, "="},
		{token.INT, "5"},
		{token.SEMICOLON, ";"},

		{token.MUT, "mut"},
		{token.IDENT, "ten"},
		{token.ASSIGN, "="},
		{token.FLOAT, "10.5"},
		{token.SEMICOLON, ";"},

		{token.FN, "fn"},
		{token.IDENT, "add"},
		{token.LPAREN, "("},
		{token.IDENT, "x"},
		{token.COLON, ":"},
		{token.IDENT, "int"},
		{token.COMMA, ","},
		{token.IDENT, "y"},
		{token.COLON, ":"},
		{token.IDENT, "int"},
		{token.RPAREN, ")"},
		{token.ARROW, "->"},
		{token.IDENT, "int"},
		{token.LBRACE, "{"},

		{token.RETURN, "return"},
		{token.IDENT, "x"},
		{token.ADD, "+"},
		{token.IDENT, "y"},
		{token.SEMICOLON, ";"},
		{token.RBRACE, "}"},

		{token.MUT, "mut"},
		{token.IDENT, "result"},
		{token.ASSIGN, "="},
		{token.IDENT, "add"},
		{token.LPAREN, "("},
		{token.IDENT, "five"},
		{token.COMMA, ","},
		{token.IDENT, "ten"},
		{token.RPAREN, ")"},
		{token.SEMICOLON, ";"},

		{token.IF, "if"},
		{token.LPAREN, "("},
		{token.IDENT, "result"},
		{token.NEQ, "!="},
		{token.INT, "15"},
		{token.RPAREN, ")"},
		{token.LBRACE, "{"},

		{token.IDENT, "print"},
		{token.LPAREN, "("},
		{token.STRING, "hello \"world\"\n"},
		{token.RPAREN, ")"},
		{token.SEMICOLON, ";"},
		{token.RBRACE, "}"},

		{token.ELSE, "else"},
		{token.LBRACE, "{"},

		{token.WHILE, "while"},
		{token.LPAREN, "("},
		{token.IDENT, "five"},
		{token.GTR, ">"},
		{token.INT, "0"},
		{token.RPAREN, ")"},
		{token.LBRACE, "{"},

		{token.IDENT, "result"},
		{token.SUB_ASSIGN, "-="},
		{token.INT, "1"},
		{token.SEMICOLON, ";"},
		{token.RBRACE, "}"},
		{token.RBRACE, "}"},

		{token.EOF, ""},
	}

	l := New("test.lc", input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - 词法单元类型不匹配. 期望=%q, 实际=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - 词法单元字面量不匹配. 期望=%q, 实际=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}

	if len(l.Errors) > 0 {
		t.Fatalf("词法分析产生错误: %v", l.Errors)
	}
}
