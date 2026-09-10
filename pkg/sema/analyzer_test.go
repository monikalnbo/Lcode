package sema

import (
	"testing"

	"lcode/pkg/lexer"
	"lcode/pkg/parser"
)

func TestSemanticAnalysisSuccess(t *testing.T) {
	code := `
fn multiply(a: int, b: int) -> int {
    return a * b;
}

fn main() {
    let x: int = 10;
    mut y = 20;
    y += 5; // 合法修改
    let z = multiply(x, y);

    if z > 100 {
        println("large result");
    }
}
`
	l := lexer.New("test.lc", code)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("语法解析错误: %v", p.Errors())
	}

	analyzer := NewAnalyzer()
	analyzer.Analyze(program)

	if len(analyzer.Errors) > 0 {
		t.Fatalf("语义分析期望成功，但遇到错误: %v", analyzer.Errors)
	}
}

func TestSemanticAnalysisErrors(t *testing.T) {
	code := `
fn bad(x: int) -> int {
    let a: int = "hello"; // 1. 类型不匹配
    let b = undefined_var + 1; // 2. 未定义变量
    let c = 10;
    c = 20; // 3. 尝试修改不可变变量
    return "string"; // 4. 返回值类型不匹配
}
`
	l := lexer.New("test.lc", code)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("语法解析错误: %v", p.Errors())
	}

	analyzer := NewAnalyzer()
	analyzer.Analyze(program)

	if len(analyzer.Errors) < 4 {
		t.Fatalf("期望捕获至少 4 个语义错误，实际捕获 %d 个: %v", len(analyzer.Errors), analyzer.Errors)
	}
}

func TestBackendSemanticAnalysis(t *testing.T) {
	code := `
struct User {
    name: string,
    age: int,
}

fn process() {
    let u: User = User { name: "Alice", age: 30 };
    let n = u.name;

    mut list = [10, 20, 30];
    let first = list[0];
    list[1] = 99;
    push(list, 100);
    let l = len(list);

    for item in list {
        if item > 50 {
            break;
        } else {
            continue;
        }
    }

    for i in range(0, 5) {
        let val = i * 2;
    }

    try {
        let bad = 10 / 0;
    } catch (err) {
        println(err);
    }
}
`
	l := lexer.New("backend_test.lc", code)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("语法解析错误: %v", p.Errors())
	}

	analyzer := NewAnalyzer()
	analyzer.Analyze(program)

	if len(analyzer.Errors) > 0 {
		t.Fatalf("后端语义分析期望成功，但遇到错误: %v", analyzer.Errors)
	}
}

