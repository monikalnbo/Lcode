package parser

import (
	"testing"

	"lcode/pkg/ast"
	"lcode/pkg/lexer"
)

func TestVarDeclStatements(t *testing.T) {
	input := `
let x = 5;
mut y: int = 10;
let z = 20;
`
	l := lexer.New("test.lc", input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Stmts) != 3 {
		t.Fatalf("program.Stmts 不包含 3 条语句. 实际数量=%d", len(program.Stmts))
	}

	tests := []struct {
		expectedIdentifier string
		expectedType       string
		isMut              bool
	}{
		{"x", "", false},
		{"y", "int", true},
		{"z", "", false},
	}

	for i, tt := range tests {
		stmt := program.Stmts[i]
		varStmt, ok := stmt.(*ast.VarDeclStmt)
		if !ok {
			t.Fatalf("stmt 不是 *ast.VarDeclStmt. 实际=%T", stmt)
		}

		if varStmt.Name.Value != tt.expectedIdentifier {
			t.Errorf("varStmt.Name.Value 期望=%s, 实际=%s", tt.expectedIdentifier, varStmt.Name.Value)
		}

		if tt.expectedType != "" {
			if varStmt.Type == nil || varStmt.Type.Name != tt.expectedType {
				t.Errorf("varStmt.Type 期望=%s, 实际=%v", tt.expectedType, varStmt.Type)
			}
		}

		if varStmt.IsMut != tt.isMut {
			t.Errorf("varStmt.IsMut 期望=%v, 实际=%v", tt.isMut, varStmt.IsMut)
		}
	}
}

func TestOperatorPrecedenceParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"-a * b;",
			"((-a) * b);",
		},
		{
			"!-a;",
			"(!(-a));",
		},
		{
			"a + b + c;",
			"((a + b) + c);",
		},
		{
			"a + b * c;",
			"(a + (b * c));",
		},
		{
			"a + b / c;",
			"(a + (b / c));",
		},
		{
			"1 + 2 * 3 == 4 + 5;",
			"((1 + (2 * 3)) == (4 + 5));",
		},
		{
			"true == false || !flag;",
			"((true == false) || (!flag));",
		},
		{
			"add(a, b * c);",
			"add(a, (b * c));",
		},
	}

	for _, tt := range tests {
		l := lexer.New("test.lc", tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		if len(program.Stmts) != 1 {
			t.Fatalf("program.Stmts 长度不为 1. 实际=%d", len(program.Stmts))
		}

		actual := program.Stmts[0].String()
		if actual != tt.expected {
			t.Errorf("表达式解析期望=%q, 实际=%q", tt.expected, actual)
		}
	}
}

func TestFuncDecl(t *testing.T) {
	input := `
fn add(x: int, y: int) -> int {
    return x + y;
}
`
	l := lexer.New("test.lc", input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Decls) != 1 {
		t.Fatalf("program.Decls 期望 1 个声明, 实际=%d", len(program.Decls))
	}

	fn, ok := program.Decls[0].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("program.Decls[0] 不是 *ast.FuncDecl. 实际=%T", program.Decls[0])
	}

	if fn.Name.Value != "add" {
		t.Errorf("函数名期望 'add', 实际=%s", fn.Name.Value)
	}

	if len(fn.Params) != 2 {
		t.Fatalf("函数参数数量期望 2, 实际=%d", len(fn.Params))
	}

	if fn.ReturnType == nil || fn.ReturnType.Name != "int" {
		t.Errorf("返回值类型期望 'int', 实际=%v", fn.ReturnType)
	}

	if len(fn.Body.Statements) != 1 {
		t.Fatalf("函数体语句数量期望 1, 实际=%d", len(fn.Body.Statements))
	}
}

func checkParserErrors(t *testing.T, p *Parser) {
	errors := p.Errors()
	if len(errors) == 0 {
		return
	}

	t.Errorf("语法解析遇到 %d 个错误:", len(errors))
	for _, msg := range errors {
		t.Errorf("语法错误: %s", msg)
	}
	t.FailNow()
}
