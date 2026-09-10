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

func TestRustMemoryManagementAndRAII(t *testing.T) {
	input := `
fn create_and_escape() -> int {
    let buf = alloc(10);
    write(buf, 0, 777);
    ret buf; // 所有权逃逸至调用者，不在本作用域释放
}

fn main() -> void {
    // 1. 测试代码块作用域自动 Drop
    {
        let temp = alloc(20);
        write(temp, 0, 123);
        // temp 离开块作用域，自动由 RAII Drop 释放
    }
    let leaks1 = check_leaks();
    assert(leaks1 == 0, "RAII 作用域自动 Drop 应当确保零内存泄漏");

    // 2. 测试返回值所有权逃逸与显式 drop
    let my_buf = create_and_escape();
    let val = read(my_buf, 0);
    assert(val == 777, "逃逸句柄数据应当保持完整");
    drop(my_buf); // 显式释放

    let leaks2 = check_leaks();
    assert(leaks2 == 0, "显式 drop 后应当零内存泄漏");
    p("Rust 内存管理全部验证通过");
}
`
	l := lexer.New("test_rust_mem.lc", input)
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
	if !strings.Contains(out, "Rust 内存管理全部验证通过") {
		t.Errorf("Expected output not found, got: %s", out)
	}
}

