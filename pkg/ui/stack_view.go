package ui

import (
	"fmt"
	"strings"

	"lcode/pkg/eval"
)

// RenderCallStackBox 渲染美观的调用栈帧监控卡片
func RenderCallStackBox(frames []*eval.CallFrame) string {
	if len(frames) == 0 {
		return BoxWithTitle("🥞 运行时调用栈 (Call Stack)", Paint(ColorMuted, "(调用栈为空，当前处于顶层全局作用域)"), ColorCyan)
	}

	var sb strings.Builder
	for i := len(frames) - 1; i >= 0; i-- {
		f := frames[i]
		frameTag := Badge(fmt.Sprintf("#%d Frame", i), FgHiWhite, RGBBg(99, 102, 241))
		funcName := Paint(Bold+FgHiWhite, f.FuncName+"()")
		loc := Paint(ColorCyan, fmt.Sprintf("%s:%d", f.Filename, f.Line))

		sb.WriteString(fmt.Sprintf("%s  %s  %s\n", frameTag, funcName, loc))

		if len(f.Args) > 0 {
			argText := Paint(ColorMuted, "    实参: ") + Paint(ColorAmber, strings.Join(f.Args, ", "))
			sb.WriteString(argText + "\n")
		}
		if i > 0 {
			sb.WriteString(Paint(ColorMuted, "    ↓\n"))
		}
	}

	title := fmt.Sprintf("🥞 运行时调用栈 (当前深度: %d 层)", len(frames))
	return BoxWithTitle(title, strings.TrimSuffix(sb.String(), "\n"), ColorIndigo)
}

// RenderMemoryStatsBox 渲染内存使用与泄漏诊断面板
func RenderMemoryStatsBox(stats eval.MemoryStats, leaks []*eval.MemoryBlock) string {
	var sb strings.Builder

	// 核心统计指标卡片
	sb.WriteString(fmt.Sprintf("%s 累计分配: %s (%d 次)\n",
		Paint(ColorEmerald, "●"),
		Paint(Bold+FgHiWhite, fmt.Sprintf("%d 字节", stats.TotalAllocatedBytes)),
		stats.AllocCount,
	))
	sb.WriteString(fmt.Sprintf("%s 当前活跃: %s (已释放 %d 次)\n",
		Paint(ColorCyan, "●"),
		Paint(Bold+FgHiWhite, fmt.Sprintf("%d 字节", stats.ActiveBytes)),
		stats.FreeCount,
	))
	sb.WriteString(fmt.Sprintf("%s 峰值内存: %s\n",
		Paint(ColorAmber, "●"),
		Paint(Bold+FgHiWhite, fmt.Sprintf("%d 字节", stats.PeakBytes)),
	))

	// 内存泄漏检测
	sb.WriteString("\n")
	if len(leaks) == 0 {
		sb.WriteString(Badge("✓ 内存安全", FgHiWhite, RGBBg(16, 185, 129)) + " " + Paint(ColorEmerald, "无内存泄漏，所有已分配堆块均已安全回收！"))
	} else {
		sb.WriteString(Badge("⚠ 内存泄漏警告", FgHiWhite, RGBBg(239, 68, 68)) + " " + Paint(ColorRose, fmt.Sprintf("检测到 %d 个未释放的内存块:\n", len(leaks))))
		for _, lk := range leaks {
			sb.WriteString(fmt.Sprintf("  • 句柄 [0x%X] 大小 %d 字长 (在 %s:%d 处由 %s 分配)\n",
				lk.Handle, lk.Size, lk.CallerFn, lk.Line, lk.CallerFn))
		}
	}

	return BoxWithTitle("🧠 运行时堆内存管理器 (Heap & Memory Inspector)", sb.String(), ColorEmerald)
}

// RenderPanicCard 渲染发生运行时恐慌或断言失败时的 Rust 风格错误追踪卡片
func RenderPanicCard(title string, message string, frames []*eval.CallFrame) string {
	var sb strings.Builder

	sb.WriteString(Badge("💥 "+title, FgHiWhite, RGBBg(239, 68, 68)) + " " + Paint(Bold+FgHiWhite, message) + "\n\n")
	sb.WriteString(Paint(ColorViolet+Bold, "📜 崩溃时完整调用栈追踪 (Stack Trace Backtrace):") + "\n")

	if len(frames) == 0 {
		sb.WriteString(Paint(ColorMuted, "  (栈帧信息不可用)\n"))
	} else {
		for i := len(frames) - 1; i >= 0; i-- {
			f := frames[i]
			sb.WriteString(fmt.Sprintf("  %s %s(%s) at %s:%d\n",
				Paint(ColorRose, fmt.Sprintf("[%d]", i)),
				Paint(Bold+FgHiWhite, f.FuncName),
				strings.Join(f.Args, ", "),
				f.Filename, f.Line,
			))
		}
	}

	sb.WriteString("\n" + Paint(ColorMuted, "提示: 请检查条件断言或边界检查逻辑。"))
	return BoxWithTitle("🚨 运行时异常终止 (Runtime Panic)", sb.String(), ColorRose)
}
