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

func TestBackendFeaturesEval(t *testing.T) {
	input := `
struct User {
    name: string,
    score: int,
}

fn main() -> void {
    // 1. Struct 测试
    mut u: User = User { name: "Antigravity", score: 100 };
    p("user_name:", u.name);
    u.score = 250;
    p("user_score:", u.score);

    // 2. 动态数组与索引测试
    mut nums = [10, 20, 30];
    nums[1] = 99;
    push(nums, 40);
    let popped = pop(nums);
    p("popped:", popped);
    p("nums_len:", len(nums));
    p("nums_1:", nums[1]);

    // 3. for ... in 循环与 break / continue
    mut sum = 0;
    for n in nums {
        if n > 50 {
            continue;
        }
        sum += n;
    }
    p("filtered_sum:", sum);

    // 4. range 循环
    mut rsum = 0;
    for i in range(1, 5) {
        if i == 4 {
            break;
        }
        rsum += i;
    }
    p("range_sum:", rsum);

    // 5. try ... catch 容错测试
    try {
        let bad = 10 / 0;
    } catch (err) {
        p("caught_error_successfully");
    }

    // 6. 字符串处理
    let upper = str_upper("hello");
    let has = str_contains(upper, "LL");
    p("upper:", upper);
    p("has_ll:", has);

    // 7. 随机数
    let r = random_int(5, 10);
    let in_bounds = r >= 5 && r <= 10;
    p("random_valid:", in_bounds);
}
`
	l := lexer.New("test_backend.lc", input)
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

	expectedSubstrings := []string{
		"user_name: Antigravity",
		"user_score: 250",
		"popped: 40",
		"nums_len: 3",
		"nums_1: 99",
		"filtered_sum: 40", // 10 + 30 (99 was skipped via continue)
		"range_sum: 6",     // 1 + 2 + 3 (break at 4)
		"caught_error_successfully",
		"upper: HELLO",
		"has_ll: true",
		"random_valid: true",
	}

	for _, exp := range expectedSubstrings {
		if !strings.Contains(out, exp) {
			t.Errorf("Expected output to contain %q, but got:\n%s", exp, out)
		}
	}
}



