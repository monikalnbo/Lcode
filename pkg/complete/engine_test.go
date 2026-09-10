package complete

import (
	"testing"
)

func TestCompletionEngine(t *testing.T) {
	engine := NewEngine()

	// 1. 测试关键字前缀补全 (键入 "wh" 应匹配 "while")
	items := engine.Complete("", "wh")
	if len(items) == 0 || items[0].Label != "while" {
		t.Fatalf("期望前缀 'wh' 补全 'while', 实际=%v", items)
	}

	// 2. 测试内置函数前缀补全 (键入 "pr" 应匹配 "print", "println")
	items = engine.Complete("", "pr")
	foundPrintln := false
	for _, it := range items {
		if it.Label == "println" {
			foundPrintln = true
			break
		}
	}
	if !foundPrintln {
		t.Fatalf("期望前缀 'pr' 匹配到 'println'")
	}

	// 3. 测试源码自定义符号上下文补全
	code := `
fn calculateTax(income: float) -> float {
    let rate = 0.2;
    return income * rate;
}
`
	items = engine.Complete(code, "calc")
	if len(items) == 0 || items[0].Label != "calculateTax" {
		t.Fatalf("期望补全自定义函数 'calculateTax', 实际=%v", items)
	}

	// 4. 测试位置补全 CompleteAt
	items, prefix := engine.CompleteAt("let num = 10;\nfn add() {\n    pri\n}", 3, 7)
	if prefix != "pri" {
		t.Fatalf("期望提取前缀 'pri', 实际=%s", prefix)
	}
	if len(items) == 0 || items[0].Label != "print" && items[0].Label != "println" {
		t.Fatalf("期望光标处补全 print/println, 实际=%v", items)
	}
}
