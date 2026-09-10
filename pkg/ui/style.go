package ui

import (
	"fmt"
	"strings"
)

// ANSI 转义码与 24 位 TrueColor 样式封装
const (
	Reset     = "\033[0m"
	Bold      = "\033[1m"
	Dim       = "\033[2m"
	Italic    = "\033[3m"
	Underline = "\033[4m"
	Inverse   = "\033[7m"

	// 经典前景色彩
	FgBlack   = "\033[30m"
	FgRed     = "\033[31m"
	FgGreen   = "\033[32m"
	FgYellow  = "\033[33m"
	FgBlue    = "\033[34m"
	FgMagenta = "\033[35m"
	FgCyan    = "\033[36m"
	FgWhite   = "\033[37m"

	// 高亮明亮色彩
	FgHiRed     = "\033[91m"
	FgHiGreen   = "\033[92m"
	FgHiYellow  = "\033[93m"
	FgHiBlue    = "\033[94m"
	FgHiMagenta = "\033[95m"
	FgHiCyan    = "\033[96m"
	FgHiWhite   = "\033[97m"

	// 现代暗黑背景色彩
	BgDarkGray = "\033[48;5;235m"
	BgCard     = "\033[48;5;237m"
	BgPrimary  = "\033[48;5;61m"
	BgSuccess  = "\033[48;5;29m"
	BgError    = "\033[48;5;52m"
)

// RGB 返回 24 位真彩色前景色转义码
func RGB(r, g, b int) string {
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
}

// RGBBg 返回 24 位真彩色背景色转义码
func RGBBg(r, g, b int) string {
	return fmt.Sprintf("\033[48;2;%d;%d;%dm", r, g, b)
}

// 现代调色板常量
var (
	ColorCyan    = RGB(6, 182, 212)    // #06b6d4 科技青
	ColorIndigo  = RGB(99, 102, 241)   // #6366f1 标志紫
	ColorViolet  = RGB(168, 85, 247)   // #a855f7 浅紫
	ColorEmerald = RGB(16, 185, 129)   // #10b981 翡翠绿
	ColorAmber   = RGB(245, 158, 11)   // #f59e0b 琥珀橙
	ColorRose    = RGB(244, 63, 94)    // #f43f5e 玫红
	ColorMuted   = RGB(100, 116, 139)  // #64748b 灰度
	ColorWhite   = RGB(248, 250, 252)  // #f8fafc 高亮白
	ColorDarkBg  = RGBBg(15, 23, 42)   // #0f172a 底色
)

// Paint 给文本上色
func Paint(color string, text string) string {
	return color + text + Reset
}

// Badge 渲染彩色药丸标签
func Badge(text, fg, bg string) string {
	return bg + fg + " " + text + " " + Reset
}

// BoxWithTitle 用 Unicode 圆角细线边框包裹内容
func BoxWithTitle(title string, content string, color string) string {
	lines := strings.Split(content, "\n")
	maxLen := len([]rune(title)) + 4
	for _, l := range lines {
		runeLen := len([]rune(stripAnsi(l)))
		if runeLen > maxLen {
			maxLen = runeLen
		}
	}
	if maxLen < 50 {
		maxLen = 50
	}

	var sb strings.Builder
	// 顶边：╭─ Title ──────╮
	sb.WriteString(color + "╭─ " + Reset + Bold + title + Reset + color + " " + strings.Repeat("─", maxLen-len([]rune(title))-3) + "╮" + Reset + "\n")

	// 内容行
	for _, l := range lines {
		runeLen := len([]rune(stripAnsi(l)))
		padding := maxLen - runeLen
		if padding < 0 {
			padding = 0
		}
		sb.WriteString(color + "│ " + Reset + l + strings.Repeat(" ", padding) + color + " │" + Reset + "\n")
	}

	// 底边：╰──────────────╯
	sb.WriteString(color + "╰" + strings.Repeat("─", maxLen+2) + "╯" + Reset)
	return sb.String()
}

// stripAnsi 移除 ANSI 控制码以准确计算可视字符长度
func stripAnsi(str string) string {
	var sb strings.Builder
	inSeq := false
	for _, r := range str {
		if r == '\033' {
			inSeq = true
			continue
		}
		if inSeq {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inSeq = false
			}
			continue
		}
		sb.WriteRune(r)
	}
	return sb.String()
}
