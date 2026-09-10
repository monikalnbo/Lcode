package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"lcode/pkg/eval"
	"lcode/pkg/lexer"
	"lcode/pkg/parser"
	"lcode/pkg/sema"
	"lcode/pkg/token"
	"lcode/pkg/ui"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		// 默认如果没有参数，运行示例的 studio 可视化面板
		samplePath := "examples/hello.lc"
		if _, err := os.Stat(samplePath); err == nil {
			content, _ := os.ReadFile(samplePath)
			ui.RunStudio(samplePath, string(content))
			return
		}
		printUsage()
		return
	}

	command := os.Args[1]

	switch command {
	case "studio", "ui":
		file := "examples/hello.lc"
		if len(os.Args) >= 3 {
			file = os.Args[2]
		}
		content := readFileOrExit(file)
		ui.RunStudio(file, content)

	case "repl":
		ui.RunREPL()

	case "run":
		if len(os.Args) < 3 {
			fmt.Println("用法: lcode run <filename>")
			os.Exit(1)
		}
		runExecute(os.Args[2])

	case "ast":
		if len(os.Args) < 3 {
			fmt.Println("用法: lcode ast <filename>")
			os.Exit(1)
		}
		runAST(os.Args[2])

	case "tokenize":
		if len(os.Args) < 3 {
			fmt.Println("用法: lcode tokenize <filename>")
			os.Exit(1)
		}
		runTokenize(os.Args[2])

	case "check":
		if len(os.Args) < 3 {
			fmt.Println("用法: lcode check <filename>")
			os.Exit(1)
		}
		runCheck(os.Args[2])

	case "version", "-v", "--version":
		fmt.Printf("Lcode 编译器系统 v%s (纯 Go 现代化终端架构)\n", version)

	case "help", "-h", "--help":
		printUsage()

	default:
		// 如果直接传入文件路径，启动 Studio 面板
		if _, err := os.Stat(command); err == nil {
			content := readFileOrExit(command)
			ui.RunStudio(command, content)
		} else {
			fmt.Printf("未知子命令: %s\n\n", command)
			printUsage()
			os.Exit(1)
		}
	}
}

func printUsage() {
	fmt.Printf("%s%sLcode 编译器前端系统 (v%s)%s\n", ui.Bold, ui.ColorCyan, version, ui.Reset)
	fmt.Println(ui.ColorMuted + "纯 Go 打造 • 专属自研现代语法 • 高颜值终端界面 (TUI)" + ui.Reset)
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  go run . studio [file]     启动全套高颜值终端编译器 Studio (默认 examples/hello.lc)")
	fmt.Println("  go run . repl              启动交互式命令行 REPL")
	fmt.Println("  go run . run <file>        编译并执行源程序")
	fmt.Println("  go run . ast <file>        以彩色树形分支结构输出抽象语法树 (AST)")
	fmt.Println("  go run . tokenize <file>   以美化表格输出词法单元 (Token) 流")
	fmt.Println("  go run . check <file>      执行编译前端全流程检查与精准代码诊断")
	fmt.Println("  go run . version           显示版本信息")
	fmt.Println("  go run . help              显示此帮助信息")
}

func readFileOrExit(filename string) string {
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("%s错误: 无法读取源文件 %s: %v%s\n", ui.ColorRose, filename, err, ui.Reset)
		os.Exit(1)
	}
	return string(content)
}

func runAST(filename string) {
	content := readFileOrExit(filename)
	l := lexer.New(filepath.Base(filename), content)
	p := parser.New(l)
	prog := p.ParseProgram()

	if len(p.Errors()) > 0 {
		fmt.Println(ui.ColorRose + "语法错误:" + ui.Reset)
		for _, err := range p.Errors() {
			fmt.Printf("  - %s\n", err)
		}
		os.Exit(1)
	}

	fmt.Println()
	astTree := ui.RenderColorAST(prog)
	fmt.Println(ui.BoxWithTitle("🌲 抽象语法树 (AST)", astTree, ui.ColorViolet))
}

func runTokenize(filename string) {
	content := readFileOrExit(filename)
	l := lexer.New(filepath.Base(filename), content)
	tokens := make([]token.Token, 0)
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == token.EOF {
			break
		}
	}

	fmt.Println()
	table := ui.RenderTokenTable(tokens)
	fmt.Println(ui.BoxWithTitle(fmt.Sprintf("🏷️  词法单元列表: %s (共 %d 个)", filename, len(tokens)), table, ui.ColorCyan))
}

func runExecute(filename string) {
	content := readFileOrExit(filename)
	l := lexer.New(filepath.Base(filename), content)
	p := parser.New(l)
	prog := p.ParseProgram()

	if len(l.Errors) > 0 || len(p.Errors()) > 0 {
		ui.RunStudio(filename, content)
		os.Exit(1)
	}

	analyzer := sema.NewAnalyzer()
	analyzer.Analyze(prog)
	if len(analyzer.Errors) > 0 {
		ui.RunStudio(filename, content)
		os.Exit(1)
	}

	start := time.Now()
	evaluator := eval.New()
	stdout, err := evaluator.RunProgram(prog)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("%s[运行时错误] %v%s\n", ui.ColorRose, err, ui.Reset)
		os.Exit(1)
	}

	fmt.Print(stdout)
	fmt.Printf("%s[执行成功, 耗时 %v]%s\n", ui.ColorMuted, duration, ui.Reset)
}

func runCheck(filename string) {
	content := readFileOrExit(filename)
	l := lexer.New(filepath.Base(filename), content)
	p := parser.New(l)
	prog := p.ParseProgram()

	analyzer := sema.NewAnalyzer()
	analyzer.Analyze(prog)

	hasErrors := len(l.Errors) > 0 || len(p.Errors()) > 0 || len(analyzer.Errors) > 0

	if hasErrors {
		fmt.Println()
		for _, msg := range l.Errors {
			fmt.Println(ui.RenderDiagnosticCard(ui.Diagnostic{Level: "error", Filename: filename, Line: 1, Column: 1, Message: msg}, content))
		}
		for _, msg := range p.Errors() {
			fmt.Println(ui.RenderDiagnosticCard(ui.Diagnostic{Level: "error", Filename: filename, Line: 1, Column: 1, Message: msg}, content))
		}
		for _, msg := range analyzer.Errors {
			fmt.Println(ui.RenderDiagnosticCard(ui.Diagnostic{Level: "error", Filename: filename, Line: 1, Column: 1, Message: msg}, content))
		}
		os.Exit(1)
	}

	fmt.Println(ui.Badge("✓ 检查通过", ui.FgHiWhite, ui.RGBBg(16, 185, 129)) + " " + ui.Paint(ui.ColorEmerald, "源文件词法、语法与类型检查全部正确！"))
}
