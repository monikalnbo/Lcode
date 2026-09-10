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

// RenderDiagnosticCard 渲染类似 Rust/Clang 的现代编译器高颜值代码错误定位框
func RenderDiagnosticCard(diag Diagnostic, sourceCode string) string {
	var sb strings.Builder

	// 错误头：error[E01]: message
	levelBadge := Badge("错误", FgHiWhite, RGBBg(239, 68, 68))
	if diag.Level == "warning" {
		levelBadge = Badge("警告", FgHiWhite, RGBBg(245, 158, 11))
	}

	sb.WriteString(fmt.Sprintf("%s %s%s%s\n", levelBadge, Bold, diag.Message, Reset))
	sb.WriteString(fmt.Sprintf("  %s-->%s %s%s:%d:%d%s\n", ColorCyan, Reset, ColorMuted, diag.Filename, diag.Line, diag.Column, Reset))

	// 获取源码上下文行
	lines := strings.Split(sourceCode, "\n")
	lineIdx := diag.Line - 1

	borderCol := ColorCyan
	sb.WriteString(fmt.Sprintf("   %s│%s\n", borderCol, Reset))

	// 上一行上下文 (若存在)
	if lineIdx-1 >= 0 && lineIdx-1 < len(lines) {
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
