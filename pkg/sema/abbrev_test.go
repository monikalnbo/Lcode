package sema

import (
	"testing"

	"lcode/pkg/lexer"
	"lcode/pkg/parser"
)

func TestAbbreviationResolution(t *testing.T) {
	input := `
fn compute_special_sum(a: int, b: int) -> int {
    ret a + b;
}

fn main() -> void {
    prin("Testing abbreviation: ");
    p("Hello concise world!");
    let res = compute_special_sum(10, 20);
    p(res);
}
`
	l := lexer.New("test_abbrev.lc", input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}

	sa := NewAnalyzer()
	sa.Analyze(prog)
	if len(sa.Errors) > 0 {
		t.Fatalf("Semantic analysis failed on abbreviations: %v", sa.Errors)
	}
}

func TestUniquePrefixFunctionCall(t *testing.T) {
	input := `
fn super_unique_calculation_algorithm(x: int) -> int {
    ret x * 3;
}

fn main() -> void {
    // 采用唯一前缀 super_unique_calc 缩写直接调用
    let val = super_unique_calc(5);
    p(val);
}
`
	l := lexer.New("test_prefix.lc", input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}

	sa := NewAnalyzer()
	sa.Analyze(prog)
	if len(sa.Errors) > 0 {
		t.Fatalf("Semantic analysis failed on unique prefix matching: %v", sa.Errors)
	}
}
