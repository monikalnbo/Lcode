package eval

import (
	"strings"
	"testing"

	"lcode/pkg/lexer"
	"lcode/pkg/parser"
)

func TestEvalCppBuiltins(t *testing.T) {
	input := `
fn main() -> void {
    let s = cpp_fast_sqrt(16.0);
    let p = cpp_pow(2.0, 3.0);
    let a = cpp_abs(-42);
    let g = cpp_gcd(48, 18);
    let f = cpp_fibonacci(10);
    let prime = cpp_is_prime(17);
    println("sqrt:", s);
    println("pow:", p);
    println("abs:", a);
    println("gcd:", g);
    println("fib:", f);
    println("prime:", prime);
}
`
	l := lexer.New("test.lc", input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	ev := New()
	out, err := ev.RunProgram(prog)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	if !strings.Contains(out, "sqrt: 4.000000") {
		t.Errorf("Expected sqrt: 4.000000, got %s", out)
	}
	if !strings.Contains(out, "pow: 8.000000") {
		t.Errorf("Expected pow: 8.000000, got %s", out)
	}
	if !strings.Contains(out, "abs: 42") {
		t.Errorf("Expected abs: 42, got %s", out)
	}
	if !strings.Contains(out, "gcd: 6") {
		t.Errorf("Expected gcd: 6, got %s", out)
	}
	if !strings.Contains(out, "fib: 55") {
		t.Errorf("Expected fib: 55, got %s", out)
	}
	if !strings.Contains(out, "prime: true") {
		t.Errorf("Expected prime: true, got %s", out)
	}
}

func TestEvalAbbreviations(t *testing.T) {
	input := `
fn double_val(x: int) -> int {
    ret x * 2;
}

fn main() -> void {
    prin("prefix:");
    p("hello world");
    let ans = double_val(21);
    p("ans:", ans);
}
`
	l := lexer.New("test_abbr.lc", input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	ev := New()
	out, err := ev.RunProgram(prog)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	if !strings.Contains(out, "prefix:hello world") {
		t.Errorf("Expected 'prefix:hello world', got %s", out)
	}
	if !strings.Contains(out, "ans: 42") {
		t.Errorf("Expected 'ans: 42', got %s", out)
	}
}

func TestRAIIScopeDrop(t *testing.T) {
	input := `
fn main() -> void {
    {
        // 局部块内部申请堆内存
        let temp = alloc(8);
        write(temp, 0, 777);
        let val = read(temp, 0);
        p("val:", val);
        // 不执行手动 free(temp)，依靠 Rust 风格的 RAII 作用域自动 Drop 释放！
    }

    // 离开块后，检查内存泄漏
    let leaks = check_leaks();
    p("leaks_after_scope:", leaks);
}
`
	l := lexer.New("test_raii.lc", input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	ev := New()
	out, err := ev.RunProgram(prog)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	if !strings.Contains(out, "val: 777") {
		t.Errorf("Expected 'val: 777', got %s", out)
	}
	if !strings.Contains(out, "leaks_after_scope: 0") {
		t.Errorf("Expected 'leaks_after_scope: 0' (automatic RAII drop), got %s", out)
	}
}


