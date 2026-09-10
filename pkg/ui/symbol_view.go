package ui

import (
	"fmt"
	"strings"

	"lcode/pkg/sema"
)

// RenderSymbolTable 将符号表与作用域信息渲染为终端美化表格
func RenderSymbolTable(scope *sema.Scope) string {
	if scope == nil {
		return "无符号表信息"
	}

	var sb strings.Builder
	header := fmt.Sprintf("%-16s │ %-8s │ %-24s │ %-12s", "符号标识符", "修饰/类型", "推导类型签名", "源码定义位置")
	divider := strings.Repeat("─", 17) + "┼" + strings.Repeat("─", 10) + "┼" + strings.Repeat("─", 26) + "┼" + strings.Repeat("─", 14)

	sb.WriteString(ColorCyan + header + Reset + "\n")
	sb.WriteString(ColorMuted + divider + Reset + "\n")

	symbols := scope.Symbols()
	if len(symbols) == 0 {
		sb.WriteString(ColorMuted + "（当前作用域内无注册符号）\n" + Reset)
		return sb.String()
	}

	for _, s := range symbols {
		kindColor := ColorCyan
		kindLabel := string(s.Kind)
		if s.Kind == sema.SymMutVar {
			kindColor = ColorRose
		} else if s.Kind == sema.SymConst {
			kindColor = ColorEmerald
		} else if s.Kind == sema.SymFunc {
			kindColor = ColorViolet
		}

		kindBadge := Paint(kindColor+Bold, fmt.Sprintf("%-8s", kindLabel))
		typeStr := "<unknown>"
		if s.Type != nil {
			typeStr = s.Type.Name()
		}

		line := fmt.Sprintf("%s%-16s%s │ %s │ %s%-24s%s │ %s%-12s%s",
			Bold+FgHiWhite, s.Name, Reset,
			kindBadge,
			ColorCyan, typeStr, Reset,
			ColorMuted, s.Pos.String(), Reset,
		)
		sb.WriteString(line + "\n")
	}

	return sb.String()
}
