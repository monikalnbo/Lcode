package ui

import (
	"fmt"
	"strings"
)

// Diagnostic 诊断项
type Diagnostic struct {
	Level    string // "error", "warning", "info"
	Filename string
	Line     int
	Column   int
	Message  string
	Hint     string
}

// ParseDiagnostic 从错误字符串中精准提取文件名、行号、列号与详细提示
func ParseDiagnostic(rawMsg string, defaultFile string) Diagnostic {
	diag := Diagnostic{
		Level:    "error",
		Filename: defaultFile,
		Line:     1,
		Column:   1,
		Message:  rawMsg,
	}

	parts := strings.SplitN(rawMsg, ": ", 2)
	if len(parts) == 2 {
		posParts := strings.Split(parts[0], ":")
		if len(posParts) >= 3 {
			diag.Filename = posParts[0]
			_, _ = fmt.Sscanf(posParts[1], "%d", &diag.Line)
			_, _ = fmt.Sscanf(posParts[2], "%d", &diag.Column)
			diag.Message = parts[1]
		} else if len(posParts) == 2 {
			_, _ = fmt.Sscanf(posParts[0], "%d", &diag.Line)
			_, _ = fmt.Sscanf(posParts[1], "%d", &diag.Column)
			diag.Message = parts[1]
		}
	}

	if strings.Contains(diag.Message, "[E0382]") {
		diag.Hint = "借用已移动的值 (borrow of moved value)"
	}
	return diag
}

// RenderDiagnosticCard 渲染类似 Rust/Clang 的现代编译器高颜值代码错误定位框
func RenderDiagnosticCard(diag Diagnostic, sourceCode string) string {
	var sb strings.Builder

	// 错误头：error[E01]: message
	levelBadge := Badge("错误", FgHiWhite, RGBBg(239, 68, 68))
	if diag.Level == "warning" {
		levelBadge = Badge("警告", FgHiWhite, RGBBg(245, 158, 11))
	} else if strings.Contains(diag.Message, "E0382") {
		levelBadge = Badge("E0382 所有权冲突", FgHiWhite, RGBBg(225, 29, 72))
	}

	sb.WriteString(fmt.Sprintf("%s %s%s%s\n", levelBadge, Bold, diag.Message, Reset))
	sb.WriteString(fmt.Sprintf("  %s-->%s %s%s:%d:%d%s\n", ColorCyan, Reset, ColorMuted, diag.Filename, diag.Line, diag.Column, Reset))

	// 获取源码上下文行
	lines := strings.Split(sourceCode, "\n")
	lineIdx := diag.Line - 1

	borderCol := ColorCyan
	sb.WriteString(fmt.Sprintf("   %s│%s\n", borderCol, Reset))

	// 若为 E0382 所有权移动错误，尝试解析先前发生移动的行号并高亮展示多点轨迹
	var movedLine int
	if strings.Contains(diag.Message, "所有权已于第 ") {
		idx := strings.Index(diag.Message, "所有权已于第 ")
		after := diag.Message[idx+len("所有权已于第 "):]
		var l int
		if _, err := fmt.Sscanf(after, "%d", &l); err == nil && l > 0 && l-1 < len(lines) && l != diag.Line {
			movedLine = l
		}
	}

	// 渲染先前移动点 (Rust multi-span trace)
	if movedLine > 0 {
		mIdx := movedLine - 1
		mLine := lines[mIdx]
		sb.WriteString(fmt.Sprintf(" %s%2d │%s %s%s%s\n", ColorAmber, movedLine, borderCol, Bold+FgHiWhite, mLine, Reset))
		sb.WriteString(fmt.Sprintf("   %s│%s   %s- 所有权在此处移走 (value moved here)%s\n", borderCol, Reset, ColorAmber+Bold, Reset))
		if diag.Line-movedLine > 1 {
			sb.WriteString(fmt.Sprintf(" %s.. │%s\n", ColorMuted, borderCol))
		}
	} else if lineIdx-1 >= 0 && lineIdx-1 < len(lines) {
		// 上一行常规上下文
		sb.WriteString(fmt.Sprintf(" %s%2d │%s %s%s%s\n", ColorMuted, diag.Line-1, borderCol, ColorMuted, lines[lineIdx-1], Reset))
	}

	// 出错行
	if lineIdx >= 0 && lineIdx < len(lines) {
		errLine := lines[lineIdx]
		sb.WriteString(fmt.Sprintf(" %s%2d │%s %s\n", Bold+FgHiWhite, diag.Line, borderCol, errLine))

		// 下划指示线:   │          ^^^^^^^^^^
		colPad := diag.Column - 1
		if colPad < 0 {
			colPad = 0
		}
		if colPad > len(errLine) {
			colPad = len(errLine)
		}

		indicatorLen := 3
		if len(errLine) > colPad && len(errLine)-colPad < indicatorLen {
			indicatorLen = len(errLine) - colPad
		}
		if indicatorLen <= 0 {
			indicatorLen = 1
		}

		squiggle := ColorRose + Bold + strings.Repeat("^", indicatorLen) + Reset
		sb.WriteString(fmt.Sprintf("   %s│%s %s%s", borderCol, Reset, strings.Repeat(" ", colPad), squiggle))
		if diag.Hint != "" {
			sb.WriteString(" " + ColorRose + diag.Hint + Reset)
		}
		sb.WriteString("\n")
	}

	// 下一行上下文 (若存在)
	if lineIdx+1 < len(lines) {
		sb.WriteString(fmt.Sprintf(" %s%2d │%s %s%s%s\n", ColorMuted, diag.Line+1, borderCol, ColorMuted, lines[lineIdx+1], Reset))
	}

	sb.WriteString(fmt.Sprintf("   %s│%s\n", borderCol, Reset))
	return sb.String()
}
