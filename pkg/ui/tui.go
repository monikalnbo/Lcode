package ui

import (
	"fmt"
	"strings"
	"time"

	"lcode/pkg/eval"
	"lcode/pkg/lexer"
	"lcode/pkg/mod"
	"lcode/pkg/parser"
	"lcode/pkg/sema"
	"lcode/pkg/token"
)

// RunStudio 展示纯 Go 终端全套编译可视化 Studio
func RunStudio(filename string, sourceCode string) {
	fmt.Println()
	renderHeaderBanner(filename)

	// 1. 源码视图
	renderSourceBox(filename, sourceCode)

	// 2. 词法分析 (Lexer)
	startLex := time.Now()
	l := lexer.New(filename, sourceCode)
	tokens := make([]token.Token, 0)
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == token.EOF {
			break
		}
	}
	lexDuration := time.Since(startLex)

	// 3. 语法分析 (Parser)
	startParse := time.Now()
	l2 := lexer.New(filename, sourceCode)
	p := parser.New(l2)
	prog := p.ParseProgram()
	parseDuration := time.Since(startParse)

	// 3.1 模块与依赖解析 (Module Resolution)
	if len(prog.Imports) > 0 && len(p.Errors()) == 0 {
		resolver := mod.NewResolver(".", "std")
		if merged, err := resolver.ResolveAll(filename, prog); err == nil {
			prog = merged
		} else {
			p.AppendError(err.Error())
		}
	}

	// 4. 语义分析 (Sema)
	startSema := time.Now()
	analyzer := sema.NewAnalyzer()
	analyzer.Analyze(prog)
	semaDuration := time.Since(startSema)

	// 5. 状态指标条
	hasLexErr := len(l.Errors) > 0
	hasParseErr := len(p.Errors()) > 0
	hasSemaErr := len(analyzer.Errors) > 0
	hasErrors := hasLexErr || hasParseErr || hasSemaErr

	renderMetricsBar(len(tokens), len(prog.Decls)+len(prog.Stmts), lexDuration+parseDuration+semaDuration, hasErrors)

	// 如果有错误，输出高颜值代码诊断框
	if hasErrors {
		renderDiagnosticsBox(filename, sourceCode, l.Errors, p.Errors(), analyzer.Errors)
		return
	}

	// 6. 词法单元摘要与列表
	fmt.Println()
	tokenContent := RenderTokenTable(tokens)
	fmt.Println(BoxWithTitle("🏷️  词法单元流 (Token Stream)", tokenContent, ColorCyan))

	// 7. 抽象语法树 (AST)
	fmt.Println()
	astContent := RenderColorAST(prog)
	fmt.Println(BoxWithTitle("🌲 抽象语法树结构 (Abstract Syntax Tree)", astContent, ColorViolet))

	// 8. 符号表与类型检查结果
	fmt.Println()
	symbolContent := RenderSymbolTable(analyzer.GlobalScope())
	fmt.Println(BoxWithTitle("🛡️  符号表与作用域 (Symbol Table & Types)", symbolContent, ColorEmerald))

	// 9. 代码执行输出 (Terminal Output)
	fmt.Println()
	startExec := time.Now()
	evaluator := eval.New()
	stdout, err := evaluator.RunProgram(prog)
	execDuration := time.Since(startExec)

	execStatus := Paint(ColorEmerald+Bold, fmt.Sprintf("执行完成 (%v)", execDuration))
	if err != nil {
		execStatus = Paint(ColorRose+Bold, fmt.Sprintf("运行时错误: %v", err))
	}

	var termContent strings.Builder
	termContent.WriteString(ColorMuted + "状态: " + Reset + execStatus + "\n\n")
	if stdout != "" {
		termContent.WriteString(Bold + FgHiWhite + stdout + Reset)
	} else {
		termContent.WriteString(ColorMuted + "(程序执行完成，无控制台标准输出)\n" + Reset)
	}

	fmt.Println(BoxWithTitle("💻 控制台输出 (Program Terminal Output)", termContent.String(), ColorAmber))

	// 10. 运行时堆内存追踪与监控卡片 (如果存在内存分配)
	if evaluator.MemManager != nil && evaluator.MemManager.Stats().AllocCount > 0 {
		fmt.Println()
		memBox := RenderMemoryStatsBox(evaluator.MemManager.Stats(), evaluator.MemManager.CheckLeaks())
		fmt.Println(memBox)
	}

	// 11. 如果发生异常或 Panic，输出完整的调用栈回溯卡片
	if err != nil && evaluator.CallStack != nil && evaluator.CallStack.Depth() > 0 {
		fmt.Println()
		panicCard := RenderPanicCard("运行时异常中断", err.Error(), evaluator.CallStack.Frames())
		fmt.Println(panicCard)
	}

	fmt.Println()
}

func renderHeaderBanner(filename string) {
	title := "⚡ Lcode Compiler Studio ⚡"
	sub := "纯 Go 极客终端编译器前端 • 自研语言语法与可视化引擎"
	content := fmt.Sprintf("%s%s%s%s\n%s%s%s", Bold+ColorCyan, title, Reset, "", ColorMuted, sub, Reset)
	fmt.Println(BoxWithTitle("Lcode 编译器终端实验室", content, ColorViolet))
}

func renderSourceBox(filename string, code string) {
	lines := strings.Split(code, "\n")
	var sb strings.Builder
	for i, l := range lines {
		sb.WriteString(fmt.Sprintf("%s%3d │%s %s\n", ColorMuted, i+1, Reset, l))
	}
	fmt.Println(BoxWithTitle(fmt.Sprintf("📝 源代码: %s (%d 行)", filename, len(lines)), strings.TrimSuffix(sb.String(), "\n"), ColorCyan))
}

func renderMetricsBar(tokenCount, astCount int, duration time.Duration, hasErrors bool) {
	status := Badge("✓ 前端检查全部通过", FgHiWhite, RGBBg(16, 185, 129))
	if hasErrors {
		status = Badge("✗ 发现编译错误", FgHiWhite, RGBBg(239, 68, 68))
	}

	bar := fmt.Sprintf("%s  %sTokens: %d%s  %sAST顶层节点: %d%s  %s分析耗时: %v%s",
		status,
		Bold+ColorCyan, tokenCount, Reset,
		Bold+ColorViolet, astCount, Reset,
		ColorMuted, duration, Reset,
	)
	fmt.Println("\n" + bar)
}

func renderDiagnosticsBox(filename, sourceCode string, lexErrs, parseErrs, semaErrs []string) {
	fmt.Println()
	var sb strings.Builder

	for _, msg := range lexErrs {
		diag := ParseDiagnostic(msg, filename)
		diag.Hint = "请检查词法字符"
		sb.WriteString(RenderDiagnosticCard(diag, sourceCode) + "\n")
	}

	for _, msg := range parseErrs {
		diag := ParseDiagnostic(msg, filename)
		diag.Hint = "语法不符合 Lcode 语法规范"
		sb.WriteString(RenderDiagnosticCard(diag, sourceCode) + "\n")
	}

	for _, msg := range semaErrs {
		diag := ParseDiagnostic(msg, filename)
		if diag.Hint == "" {
			diag.Hint = "语义或类型约束不满足"
		}
		sb.WriteString(RenderDiagnosticCard(diag, sourceCode) + "\n")
	}

	fmt.Println(BoxWithTitle("⚠️  代码诊断报告 (Compiler Diagnostics)", strings.TrimSuffix(sb.String(), "\n"), ColorRose))
}
