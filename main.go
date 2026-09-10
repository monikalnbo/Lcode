package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lcode/pkg/codegen/cpp"
	"lcode/pkg/complete"
	"lcode/pkg/eval"
	"lcode/pkg/lexer"
	"lcode/pkg/mod"
	"lcode/pkg/parser"
	"lcode/pkg/sema"
	"lcode/pkg/token"
	"lcode/pkg/ui"
)

const version = "0.2.0"

func main() {
	if len(os.Args) < 2 {
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

	case "to-cpp":
		runToCpp(os.Args[2:])

	case "complete":
		runComplete(os.Args[2:])

	case "init":
		runInit()

	case "version", "-v", "--version":
		fmt.Printf("Lcode 编译器系统 v%s (纯 Go 现代化终端架构 + 内置 C++ 核心库)\n", version)

	case "help", "-h", "--help":
		printUsage()

	default:
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
	fmt.Println(ui.ColorMuted + "纯 Go 打造 • 自研现代语法 • 高颜值终端界面 • 内置 C++ 核心库与自动补全" + ui.Reset)
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  go run . studio [file]          启动全套高颜值终端编译器 Studio (默认 examples/hello.lc)")
	fmt.Println("  go run . repl                   启动交互式命令行 REPL (支持按 :c 补全)")
	fmt.Println("  go run . run <file>             编译并执行源程序 (自动解析模块导入)")
	fmt.Println("  go run . to-cpp <file> [-o out] 将源文件转译为现代 C++17 源码并调用内置 C++ 库")
	fmt.Println("  go run . complete <prefix>      智能代码自动补全建议查询 (支持 --json 格式)")
	fmt.Println("  go run . init                   初始化当前目录为 Lcode 项目 (生成 lcode.toml)")
	fmt.Println("  go run . ast <file>             以彩色树形分支结构输出抽象语法树 (AST)")
	fmt.Println("  go run . tokenize <file>        以美化表格输出词法单元 (Token) 流")
	fmt.Println("  go run . check <file>           执行编译前端全流程检查与精准代码诊断")
	fmt.Println("  go run . version                显示版本信息")
	fmt.Println("  go run . help                   显示此帮助信息")
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

	if len(prog.Imports) > 0 {
		resolver := mod.NewResolver(".", "std")
		if merged, err := resolver.ResolveAll(filename, prog); err == nil {
			prog = merged
		}
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

	if len(prog.Imports) > 0 {
		resolver := mod.NewResolver(".", "std")
		merged, err := resolver.ResolveAll(filename, prog)
		if err != nil {
			fmt.Printf("%s[模块解析错误] %v%s\n", ui.ColorRose, err, ui.Reset)
			os.Exit(1)
		}
		prog = merged
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

	var modErrors []string
	if len(prog.Imports) > 0 {
		resolver := mod.NewResolver(".", "std")
		merged, err := resolver.ResolveAll(filename, prog)
		if err != nil {
			modErrors = append(modErrors, err.Error())
		} else {
			prog = merged
		}
	}

	analyzer := sema.NewAnalyzer()
	analyzer.Analyze(prog)

	hasErrors := len(l.Errors) > 0 || len(p.Errors()) > 0 || len(analyzer.Errors) > 0 || len(modErrors) > 0

	if hasErrors {
		fmt.Println()
		for _, msg := range l.Errors {
			fmt.Println(ui.RenderDiagnosticCard(ui.ParseDiagnostic(msg, filename), content))
		}
		for _, msg := range p.Errors() {
			fmt.Println(ui.RenderDiagnosticCard(ui.ParseDiagnostic(msg, filename), content))
		}
		for _, msg := range modErrors {
			fmt.Println(ui.RenderDiagnosticCard(ui.ParseDiagnostic(msg, filename), content))
		}
		for _, msg := range analyzer.Errors {
			fmt.Println(ui.RenderDiagnosticCard(ui.ParseDiagnostic(msg, filename), content))
		}
		os.Exit(1)
	}

	fmt.Println(ui.Badge("✓ 检查通过", ui.FgHiWhite, ui.RGBBg(16, 185, 129)) + " " + ui.Paint(ui.ColorEmerald, "源文件与依赖模块词法、语法与类型检查全部正确！"))
}

func runToCpp(args []string) {
	if len(args) < 1 {
		fmt.Println("用法: lcode to-cpp <filename> [-o output.cpp]")
		os.Exit(1)
	}
	srcFile := args[0]
	outFile := strings.TrimSuffix(srcFile, filepath.Ext(srcFile)) + ".cpp"
	if len(args) >= 3 && args[1] == "-o" {
		outFile = args[2]
	}

	content := readFileOrExit(srcFile)
	l := lexer.New(filepath.Base(srcFile), content)
	p := parser.New(l)
	prog := p.ParseProgram()

	if len(l.Errors) > 0 || len(p.Errors()) > 0 {
		fmt.Println(ui.ColorRose + "编译前端错误，无法生成 C++ 源码:" + ui.Reset)
		for _, err := range append(l.Errors, p.Errors()...) {
			fmt.Printf("  - %s\n", err)
		}
		os.Exit(1)
	}

	if len(prog.Imports) > 0 {
		resolver := mod.NewResolver(".", "std")
		merged, err := resolver.ResolveAll(srcFile, prog)
		if err != nil {
			fmt.Printf("%s模块加载失败: %v%s\n", ui.ColorRose, err, ui.Reset)
			os.Exit(1)
		}
		prog = merged
	}

	analyzer := sema.NewAnalyzer()
	analyzer.Analyze(prog)
	if len(analyzer.Errors) > 0 {
		fmt.Println(ui.ColorRose + "类型语义错误，无法生成 C++ 源码:" + ui.Reset)
		for _, err := range analyzer.Errors {
			fmt.Printf("  - %s\n", err)
		}
		os.Exit(1)
	}

	gen := cpp.NewGenerator()
	cppCode, err := gen.Generate(prog)
	if err != nil {
		fmt.Printf("%s[C++ 代码生成错误] %v%s\n", ui.ColorRose, err, ui.Reset)
		os.Exit(1)
	}

	if dir := filepath.Dir(outFile); dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}

	if err := os.WriteFile(outFile, []byte(cppCode), 0644); err != nil {
		fmt.Printf("%s写入输出文件 %s 失败: %v%s\n", ui.ColorRose, outFile, err, ui.Reset)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println(ui.Badge("✓ 转译成功", ui.FgHiWhite, ui.RGBBg(16, 185, 129)) + " " + ui.Paint(ui.ColorCyan, fmt.Sprintf("现代 C++17 源码已输出至: %s", outFile)))
	fmt.Println(ui.Paint(ui.ColorMuted, "可直接调用 g++ 或 clang++ 编译: g++ -std=c++17 -Icpp/include "+outFile+" -o "+strings.TrimSuffix(outFile, ".cpp")))
}

func runComplete(args []string) {
	if len(args) < 1 {
		fmt.Println("用法: lcode complete <prefix> [file] [--json]")
		os.Exit(1)
	}
	prefix := args[0]
	file := ""
	isJSON := false
	for _, a := range args[1:] {
		if a == "--json" {
			isJSON = true
		} else if file == "" {
			file = a
		}
	}

	source := ""
	if file != "" {
		if data, err := os.ReadFile(file); err == nil {
			source = string(data)
		}
	}

	engine := complete.NewEngine()
	items := engine.Complete(source, prefix)

	if isJSON {
		jsonStr, _ := ui.RenderCompletionJSON(items)
		fmt.Println(jsonStr)
		return
	}

	fmt.Println()
	fmt.Println(ui.RenderCompletionCard(items, prefix))
}

func runInit() {
	tomlContent := `[package]
name = "my_lcode_app"
version = "0.1.0"
authors = ["Developer"]
edition = "2026"

[dependencies]
std = "builtin"
cpp = "builtin"

[build]
target = "native"
cpp_standard = "c++17"
`
	mainLcContent := `import "std/math";
import "std/cpp/core";

fn main() -> void {
    println("🎉 欢迎使用 Lcode 编程语言！");
    let r = cpp_fast_sqrt(25.0);
    println("极速开方结果:", r);
}
`
	_ = os.WriteFile("lcode.toml", []byte(tomlContent), 0644)
	_ = os.WriteFile("main.lc", []byte(mainLcContent), 0644)
	fmt.Println(ui.Badge("✓ 项目初始化成功", ui.FgHiWhite, ui.RGBBg(16, 185, 129)) + " " + ui.Paint(ui.ColorCyan, "已创建 lcode.toml 与 main.lc"))
	fmt.Println(ui.Paint(ui.ColorMuted, "运行项目: go run . run main.lc"))
}
