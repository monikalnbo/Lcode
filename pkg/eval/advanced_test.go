package eval

import (
	"strings"
	"testing"

	"lcode/pkg/lexer"
	"lcode/pkg/parser"
)

func TestTimeAndMemoryEval(t *testing.T) {
	input := `
fn main() -> void {
    let t0 = time_now_ms();
    let h = mem_alloc(3);
    mem_write(h, 0, 42);
    mem_write(h, 1, 99);
    let v0 = mem_read(h, 0);
    let v1 = mem_read(h, 1);
    mem_stats();
    mem_free(h);
    let leaks = mem_check_leaks();
    let elapsed = time_elapsed_ms(t0);
    println("v0:", v0);
    println("v1:", v1);
    println("leaks:", leaks);
    println("elapsed_ok:", elapsed >= 0);
}
`
	l := lexer.New("test_mem.lc", input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}

	ev := New()
	out, err := ev.RunProgram(prog)
	if err != nil {
		t.Fatalf("RunProgram error: %v", err)
	}

	if !strings.Contains(out, "v0: 42") {
		t.Errorf("Expected v0: 42, got %s", out)
	}
	if !strings.Contains(out, "v1: 99") {
		t.Errorf("Expected v1: 99, got %s", out)
	}
	if !strings.Contains(out, "leaks: 0") {
		t.Errorf("Expected leaks: 0, got %s", out)
	}
	if !strings.Contains(out, "elapsed_ok: true") {
		t.Errorf("Expected elapsed_ok: true, got %s", out)
	}
}

func TestMemoryDoubleFreeDetection(t *testing.T) {
	input := `
fn main() -> void {
    let h = mem_alloc(2);
    mem_free(h);
    mem_free(h); // Double free
}
`
	l := lexer.New("test_df.lc", input)
	p := parser.New(l)
	prog := p.ParseProgram()

	ev := New()
	_, err := ev.RunProgram(prog)
	if err == nil {
		t.Fatalf("Expected double-free error, got nil")
	}
	if !strings.Contains(err.Error(), "Double Free") {
		t.Errorf("Expected 'Double Free' error message, got %v", err)
	}
}

func TestMemoryOutOfBoundsDetection(t *testing.T) {
	input := `
fn main() -> void {
    let h = mem_alloc(2);
    mem_write(h, 5, 100); // Out of bounds
}
`
	l := lexer.New("test_oob.lc", input)
	p := parser.New(l)
	prog := p.ParseProgram()

	ev := New()
	_, err := ev.RunProgram(prog)
	if err == nil {
		t.Fatalf("Expected out-of-bounds error, got nil")
	}
	if !strings.Contains(err.Error(), "Out of Bounds") {
		t.Errorf("Expected 'Out of Bounds' error message, got %v", err)
	}
}

func TestAssertAndCallStack(t *testing.T) {
	input := `
fn foo(x: int) -> void {
    assert(x > 0, "x 必须大于零");
}

fn bar() -> void {
    foo(-1);
}

fn main() -> void {
    bar();
}
`
	l := lexer.New("test_assert.lc", input)
	p := parser.New(l)
	prog := p.ParseProgram()

	ev := New()
	_, err := ev.RunProgram(prog)
	if err == nil {
		t.Fatalf("Expected assertion failure error, got nil")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "Assertion Failed") {
		t.Errorf("Expected Assertion Failed, got: %s", errStr)
	}
	if !strings.Contains(errStr, "foo") || !strings.Contains(errStr, "bar") {
		t.Errorf("Expected call stack trace containing foo and bar, got: %s", errStr)
	}
}

func TestStackOverflowDetection(t *testing.T) {
	input := `
fn recurse() -> void {
    recurse();
}

fn main() -> void {
    recurse();
}
`
	l := lexer.New("test_overflow.lc", input)
	p := parser.New(l)
	prog := p.ParseProgram()

	ev := New()
	ev.CallStack = NewCallStack(20) // 设置测试用的小深度上限 20 层
	_, err := ev.RunProgram(prog)
	if err == nil {
		t.Fatalf("Expected Stack Overflow error, got nil")
	}
	if !strings.Contains(err.Error(), "Stack Overflow") {
		t.Errorf("Expected Stack Overflow error, got %v", err)
	}
}
