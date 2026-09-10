package sema

import (
	"strings"
)

// StandardAliases 常用内建函数缩写映射
var StandardAliases = map[string]string{
	"p":        "println",
	"pr":       "print",
	"prin":     "print",
	"pri":      "print",
	"pln":      "println",
	"prn":      "println",
	"mem_chk":  "mem_check_leaks",
	"stk_dump": "stack_dump",
	"stk":      "stack_dump",
	"now":      "time_now_ms",
	"sleep":    "time_sleep_ms",
}

// ResolveFunctionAbbrev 根据缩写或最短唯一前缀，推导并返回规范的函数名称
func (sa *SemanticAnalyzer) ResolveFunctionAbbrev(name string) (string, bool) {
	// 1. 优先查找内置固定缩写表
	if canonical, ok := StandardAliases[name]; ok {
		return canonical, true
	}

	// 2. 唯一前缀扫描 (Unique Prefix Matching)
	// 收集当前作用域可访问的所有函数符号
	candidates := make([]string, 0)

	curr := sa.currentScope
	for curr != nil {
		for symName, sym := range curr.symbols {
			if sym.Kind == SymFunc && strings.HasPrefix(symName, name) {
				if !containsStr(candidates, symName) {
					candidates = append(candidates, symName)
				}
			}
		}
		curr = curr.parent
	}

	// 如果仅有 print 与 println 符合前缀 (例如 "prin")，且前缀更偏向 "print"
	if len(candidates) == 2 {
		hasPrint := containsStr(candidates, "print")
		hasPrintln := containsStr(candidates, "println")
		if hasPrint && hasPrintln {
			if strings.HasPrefix("print", name) {
				return "print", true
			}
		}
	}

	// 如果唯一匹配 (恰好只有 1 个候选者符合前缀)
	if len(candidates) == 1 {
		return candidates[0], true
	}

	return "", false
}

func containsStr(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
