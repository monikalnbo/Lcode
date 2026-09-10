package sema

import (
	"strings"
	"testing"

	"lcode/pkg/lexer"
	"lcode/pkg/parser"
)

func TestBorrowCheckerMoveDetection(t *testing.T) {
	input := `
fn main() -> void {
    let a = alloc(4);
    let b = a; // 所有权转移: a -> b
    let val = read(a, 0); // 错误：非法访问已转移变量 a
    p(val);
}
`
	l := lexer.New("test_move.lc", input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}

	sa := NewAnalyzer()
	sa.Analyze(prog)

	if len(sa.Errors) == 0 {
		t.Fatalf("Expected borrow checker error E0382 for accessing moved variable 'a', got none")
	}

	hasE0382 := false
	for _, err := range sa.Errors {
		if strings.Contains(err, "E0382") && strings.Contains(err, "borrow of moved value") {
			hasE0382 = true
			break
		}
	}

	if !hasE0382 {
		t.Errorf("Expected E0382 borrow of moved value error, got: %v", sa.Errors)
	}
}

func TestBorrowCheckerValidUsage(t *testing.T) {
	input := `
fn main() -> void {
    let a = alloc(4);
    write(a, 0, 100);
    let v = read(a, 0);
    p("v:", v);
    let b = a; // 转移给 b
    let v2 = read(b, 0); // 合法：访问持有者 b
    p("v2:", v2);
}
`
	l := lexer.New("test_valid.lc", input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}

	sa := NewAnalyzer()
	sa.Analyze(prog)

	if len(sa.Errors) > 0 {
		t.Fatalf("Expected valid usage to pass borrow checker, got errors: %v", sa.Errors)
	}
}

func TestBorrowCheckerPassByValueMove(t *testing.T) {
	input := `
fn consume(buf: int) -> void {
    p("Consumed buffer:", buf);
}

fn main() -> void {
    let a = alloc(8);
    consume(a); // 按值传递：所有权移动给 consume
    let v = read(a, 0); // 错误：已移动变量二次访问
    p(v);
}
`
	l := lexer.New("test_fn_move.lc", input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}

	sa := NewAnalyzer()
	sa.Analyze(prog)

	if len(sa.Errors) == 0 {
		t.Fatalf("Expected E0382 error for accessing moved variable passed by value, got none")
	}

	hasE0382 := false
	for _, err := range sa.Errors {
		if strings.Contains(err, "E0382") {
			hasE0382 = true
			break
		}
	}
	if !hasE0382 {
		t.Errorf("Expected E0382 error, got: %v", sa.Errors)
	}
}

func TestBorrowCheckerBorrowReference(t *testing.T) {
	input := `
fn inspect(buf: int) -> void {
    p("Inspecting buffer:", buf);
}

fn main() -> void {
    let a = alloc(8);
    inspect(&a); // 引用借用：不转移所有权
    let v = read(a, 0); // 合法：a 依然保持所有权
    p("v:", v);
}
`
	l := lexer.New("test_borrow.lc", input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}

	sa := NewAnalyzer()
	sa.Analyze(prog)

	if len(sa.Errors) > 0 {
		t.Fatalf("Expected borrow reference to preserve ownership, got errors: %v", sa.Errors)
	}
}

func TestBorrowCheckerExplicitDrop(t *testing.T) {
	input := `
fn main() -> void {
    let a = alloc(16);
    drop(a); // 显式消耗并释放所有权
    let v = read(a, 0); // 错误：已 drop 的变量禁止二次读取
    p(v);
}
`
	l := lexer.New("test_drop.lc", input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}

	sa := NewAnalyzer()
	sa.Analyze(prog)

	if len(sa.Errors) == 0 {
		t.Fatalf("Expected E0382 error for accessing dropped variable, got none")
	}
	hasE0382 := false
	for _, err := range sa.Errors {
		if strings.Contains(err, "E0382") {
			hasE0382 = true
			break
		}
	}
	if !hasE0382 {
		t.Errorf("Expected E0382 error, got: %v", sa.Errors)
	}
}

