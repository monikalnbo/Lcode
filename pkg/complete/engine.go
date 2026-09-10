package complete

import (
	"fmt"
	"sort"
	"strings"

	"lcode/pkg/ast"
	"lcode/pkg/lexer"
	"lcode/pkg/parser"
)

// CompletionItem 自动补全建议项
type CompletionItem struct {
	Label         string `json:"label"`         // 候选项文本
	Kind          string `json:"kind"`          // Keyword, Type, Function, Variable, Module
	Detail        string `json:"detail"`        // 详细信息（如类型签名）
	Documentation string `json:"documentation"` // 用法说明
	InsertText    string `json:"insert_text"`   // 插入内容
}

// Builtins 常驻全局补全库
var builtins = []CompletionItem{
	// 关键字
	{Label: "fn", Kind: "Keyword", Detail: "函数声明", Documentation: "定义一个命名的 Lcode 函数", InsertText: "fn ${1:name}(${2:params}) -> ${3:void} {\n    $0\n}"},
	{Label: "let", Kind: "Keyword", Detail: "不可变绑定", Documentation: "声明一个不可变常量绑定", InsertText: "let ${1:name} = ${2:value};"},
	{Label: "mut", Kind: "Keyword", Detail: "可变变量", Documentation: "声明一个可重新赋值的可变变量", InsertText: "mut ${1:name} = ${2:value};"},
	{Label: "if", Kind: "Keyword", Detail: "条件分支", Documentation: "条件判断语句", InsertText: "if ${1:condition} {\n    $0\n}"},
	{Label: "else", Kind: "Keyword", Detail: "条件否则分支", Documentation: "与 if 配合的否则分支", InsertText: "else {\n    $0\n}"},
	{Label: "while", Kind: "Keyword", Detail: "循环语句", Documentation: "当条件满足时持续循环", InsertText: "while ${1:condition} {\n    $0\n}"},
	{Label: "return", Kind: "Keyword", Detail: "函数返回", Documentation: "从当前函数返回指定值", InsertText: "return "},
	{Label: "import", Kind: "Keyword", Detail: "模块导入", Documentation: "导入标准库或项目模块", InsertText: "import \"${1:std/math}\";"},
	{Label: "true", Kind: "Keyword", Detail: "bool", Documentation: "布尔真值", InsertText: "true"},
	{Label: "false", Kind: "Keyword", Detail: "bool", Documentation: "布尔假值", InsertText: "false"},

	// 基本类型
	{Label: "int", Kind: "Type", Detail: "64位有符号整型", Documentation: "内置整型类型", InsertText: "int"},
	{Label: "float", Kind: "Type", Detail: "64位双精度浮点型", Documentation: "内置浮点类型", InsertText: "float"},
	{Label: "string", Kind: "Type", Detail: "UTF-8文本字符串", Documentation: "内置字符串类型", InsertText: "string"},
	{Label: "bool", Kind: "Type", Detail: "布尔类型", Documentation: "内置布尔类型", InsertText: "bool"},
	{Label: "void", Kind: "Type", Detail: "无返回值", Documentation: "表示函数无返回类型", InsertText: "void"},

	// 内置函数
	{Label: "println", Kind: "Function", Detail: "println(...args)", Documentation: "格式化输出至控制台并自动换行", InsertText: "println($1);"},
	{Label: "print", Kind: "Function", Detail: "print(...args)", Documentation: "格式化输出至控制台（不换行）", InsertText: "print($1);"},
	{Label: "panic", Kind: "Function", Detail: "panic(msg: string)", Documentation: "触发运行时恐慌中断并输出调用栈回溯", InsertText: "panic(\"$1\");"},
	{Label: "assert", Kind: "Function", Detail: "assert(cond: bool, msg: string)", Documentation: "断言条件必须满足，否则中断报错", InsertText: "assert($1, \"$2\");"},
	{Label: "stack_dump", Kind: "Function", Detail: "stack_dump() -> void", Documentation: "打印当前调用栈完整帧结构", InsertText: "stack_dump();"},
	{Label: "stack_depth", Kind: "Function", Detail: "stack_depth() -> int", Documentation: "获取当前函数调用栈层数", InsertText: "stack_depth()"},

	// 标准库与模块
	{Label: "std/math", Kind: "Module", Detail: "标准数学库", Documentation: "包含 abs, max, min, pow, clamp", InsertText: "\"std/math\""},
	{Label: "std/algo", Kind: "Module", Detail: "标准算法库", Documentation: "包含 gcd, is_prime 等常用算法", InsertText: "\"std/algo\""},
	{Label: "std/io", Kind: "Module", Detail: "标准IO输出库", Documentation: "包含格式化面板与打印工具", InsertText: "\"std/io\""},
	{Label: "std/time", Kind: "Module", Detail: "标准时间管理库", Documentation: "包含 now_ms, sleep_ms, elapsed_ms 等时间控制", InsertText: "\"std/time\""},
	{Label: "std/mem", Kind: "Module", Detail: "标准内存管理库", Documentation: "包含 alloc, free, read, write, check_leaks", InsertText: "\"std/mem\""},
	{Label: "std/error", Kind: "Module", Detail: "标准错误处理库", Documentation: "包含 assert_true, assert_eq, is_ok, is_err", InsertText: "\"std/error\""},
	{Label: "std/cpp/core", Kind: "Module", Detail: "内置 C++ 核心库", Documentation: "高性能现代 C++ 快速数学与算法库", InsertText: "\"std/cpp/core\""},

	// 时间管理函数提示
	{Label: "time_now_ms", Kind: "Function", Detail: "time_now_ms() -> int", Documentation: "获取当前 Unix 毫秒时间戳", InsertText: "time_now_ms()"},
	{Label: "time_sleep_ms", Kind: "Function", Detail: "time_sleep_ms(ms: int) -> void", Documentation: "休眠暂停执行指定毫秒数", InsertText: "time_sleep_ms($1);"},
	{Label: "time_elapsed_ms", Kind: "Function", Detail: "time_elapsed_ms(start: int) -> int", Documentation: "计算自 start_ms 起至今经历的毫秒数", InsertText: "time_elapsed_ms($1)"},
	{Label: "time_format_now", Kind: "Function", Detail: "time_format_now() -> string", Documentation: "获取当前系统时间格式化字符串", InsertText: "time_format_now()"},

	// 内存管理函数提示
	{Label: "mem_alloc", Kind: "Function", Detail: "mem_alloc(size: int) -> int", Documentation: "申请连续堆内存块并返回唯一句柄", InsertText: "mem_alloc($1)"},
	{Label: "mem_free", Kind: "Function", Detail: "mem_free(handle: int) -> int", Documentation: "释放指定句柄的内存块并检测双重释放", InsertText: "mem_free($1)"},
	{Label: "mem_write", Kind: "Function", Detail: "mem_write(h: int, off: int, val: int)", Documentation: "向内存指定偏移写入数值", InsertText: "mem_write($1, $2, $3);"},
	{Label: "mem_read", Kind: "Function", Detail: "mem_read(h: int, off: int) -> int", Documentation: "从内存指定偏移读取数值", InsertText: "mem_read($1, $2)"},
	{Label: "mem_stats", Kind: "Function", Detail: "mem_stats() -> void", Documentation: "打印当前堆内存分配与活跃字节数", InsertText: "mem_stats();"},
	{Label: "mem_check_leaks", Kind: "Function", Detail: "mem_check_leaks() -> int", Documentation: "检查未回收内存块，返回泄漏总数", InsertText: "mem_check_leaks()"},

	// 标准库函数提示
	{Label: "abs", Kind: "Function", Detail: "fn abs(x: int) -> int", Documentation: "计算绝对值 (来自 std/math)", InsertText: "abs($1)"},
	{Label: "max", Kind: "Function", Detail: "fn max(a: int, b: int) -> int", Documentation: "返回两者中较大值 (来自 std/math)", InsertText: "max($1, $2)"},
	{Label: "min", Kind: "Function", Detail: "fn min(a: int, b: int) -> int", Documentation: "返回两者中较小值 (来自 std/math)", InsertText: "min($1, $2)"},
	{Label: "pow", Kind: "Function", Detail: "fn pow(base: int, exp: int) -> int", Documentation: "计算整数乘方 (来自 std/math)", InsertText: "pow($1, $2)"},
	{Label: "gcd", Kind: "Function", Detail: "fn gcd(a: int, b: int) -> int", Documentation: "计算最大公约数 (来自 std/algo)", InsertText: "gcd($1, $2)"},
	{Label: "is_prime", Kind: "Function", Detail: "fn is_prime(n: int) -> bool", Documentation: "素数判断算法 (来自 std/algo)", InsertText: "is_prime($1)"},

	// 内置 C++ 高性能算法函数提示
	{Label: "cpp_fast_sqrt", Kind: "Function", Detail: "fn cpp_fast_sqrt(x: float) -> float", Documentation: "C++ 内置底层极速开方算法", InsertText: "cpp_fast_sqrt($1)"},
	{Label: "cpp_pow", Kind: "Function", Detail: "fn cpp_pow(base: float, exp: float) -> float", Documentation: "C++ 高精度快速浮点乘方", InsertText: "cpp_pow($1, $2)"},
	{Label: "cpp_abs", Kind: "Function", Detail: "fn cpp_abs(x: int) -> int", Documentation: "C++ 高性能绝对值运算", InsertText: "cpp_abs($1)"},
}

// Engine 自动补全分析引擎
type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

// Complete 根据前缀字符串搜索候选项
func (e *Engine) Complete(sourceCode string, prefix string) []CompletionItem {
	candidates := make([]CompletionItem, 0)
	seen := make(map[string]bool)

	// 1. 扫描源码中已定义的函数与变量标识符
	codeSymbols := extractSymbols(sourceCode)
	for _, sym := range codeSymbols {
		if !seen[sym.Label] {
			seen[sym.Label] = true
			candidates = append(candidates, sym)
		}
	}

	// 2. 加入内置全局候选项
	for _, b := range builtins {
		if !seen[b.Label] {
			seen[b.Label] = true
			candidates = append(candidates, b)
		}
	}

	// 3. 前缀过滤与评分排序
	if prefix == "" {
		return candidates
	}

	prefixLower := strings.ToLower(prefix)
	matched := make([]CompletionItem, 0)

	for _, c := range candidates {
		labelLower := strings.ToLower(c.Label)
		if strings.HasPrefix(labelLower, prefixLower) {
			matched = append(matched, c)
		}
	}

	// 按前缀完全匹配度与类别排序
	sort.Slice(matched, func(i, j int) bool {
		// 完全匹配前缀排在前面
		if strings.HasPrefix(matched[i].Label, prefix) && !strings.HasPrefix(matched[j].Label, prefix) {
			return true
		}
		if !strings.HasPrefix(matched[i].Label, prefix) && strings.HasPrefix(matched[j].Label, prefix) {
			return false
		}
		return matched[i].Label < matched[j].Label
	})

	return matched
}

// CompleteAt 根据代码位置（行号与列号）执行上下文补全
func (e *Engine) CompleteAt(sourceCode string, line, col int) ([]CompletionItem, string) {
	lines := strings.Split(sourceCode, "\n")
	if line <= 0 || line > len(lines) {
		return e.Complete(sourceCode, ""), ""
	}

	curLine := lines[line-1]
	if col > len(curLine) {
		col = len(curLine)
	}

	// 提取光标前的词缀
	prefixStart := col
	for prefixStart > 0 {
		ch := curLine[prefixStart-1]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '/' {
			prefixStart--
		} else {
			break
		}
	}

	prefix := curLine[prefixStart:col]
	return e.Complete(sourceCode, prefix), prefix
}

// extractSymbols 从源码中快速提取用户自定义的符号声明
func extractSymbols(code string) []CompletionItem {
	items := make([]CompletionItem, 0)
	if strings.TrimSpace(code) == "" {
		return items
	}

	l := lexer.New("<complete>", code)
	p := parser.New(l)
	prog := p.ParseProgram()

	for _, decl := range prog.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			params := make([]string, 0, len(fn.Params))
			for _, p := range fn.Params {
				params = append(params, p.String())
			}
			ret := "void"
			if fn.ReturnType != nil {
				ret = fn.ReturnType.Name
			}
			sig := fmt.Sprintf("fn %s(%s) -> %s", fn.Name.Value, strings.Join(params, ", "), ret)
			items = append(items, CompletionItem{
				Label:         fn.Name.Value,
				Kind:          "Function",
				Detail:        sig,
				Documentation: "当前源文件定义的函数",
				InsertText:    fn.Name.Value + "($1)",
			})
		}
	}

	for _, stmt := range prog.Stmts {
		if v, ok := stmt.(*ast.VarDeclStmt); ok {
			kind := "Variable"
			if !v.IsMut {
				kind = "Constant"
			}
			typeStr := "<inferred>"
			if v.Type != nil {
				typeStr = v.Type.Name
			}
			items = append(items, CompletionItem{
				Label:         v.Name.Value,
				Kind:          kind,
				Detail:        typeStr,
				Documentation: "当前作用域内声明的变量",
				InsertText:    v.Name.Value,
			})
		}
	}

	return items
}
