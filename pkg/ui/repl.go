package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"lcode/pkg/eval"
	"lcode/pkg/lexer"
	"lcode/pkg/parser"
	"lcode/pkg/sema"
)

// RunREPL 启动彩色交互式命令行
func RunREPL() {
	fmt.Println()
	fmt.Println(Bold + ColorCyan + "=== Lcode 交互式编译器 REPL (v0.1.0) ===" + Reset)
	fmt.Println(ColorMuted + "输入 Lcode 语句直接求值。输入 ':help' 查看命令，输入 'exit' 退出。" + Reset)
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	evaluator := eval.New()
	analyzer := sema.NewAnalyzer()

	for {
		fmt.Print(Bold + ColorViolet + "lcode> " + Reset)
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" || line == ":q" {
			fmt.Println(ColorEmerald + "再见！" + Reset)
			break
		}
		if line == ":help" {
			fmt.Println(ColorCyan + "REPL 命令列表:" + Reset)
			fmt.Println("  :help      显示此帮助")
			fmt.Println("  :tokens    输入代码并显示 Tokens 表")
			fmt.Println("  :ast       输入代码并显示彩色 AST")
			fmt.Println("  exit       退出 REPL")
			continue
		}

		// 解析
		l := lexer.New("<repl>", line)
		p := parser.New(l)
		prog := p.ParseProgram()

		if len(l.Errors) > 0 {
			fmt.Println(ColorRose + "词法错误: " + l.Errors[0] + Reset)
			continue
		}
		if len(p.Errors()) > 0 {
			fmt.Println(ColorRose + "语法错误: " + p.Errors()[0] + Reset)
			continue
		}

		analyzer.Analyze(prog)
		if len(analyzer.Errors) > 0 {
			fmt.Println(ColorRose + "语义错误: " + analyzer.Errors[0] + Reset)
			continue
		}

		// 执行
		out, err := evaluator.RunProgram(prog)
		if err != nil {
			fmt.Println(ColorRose + "运行错误: " + err.Error() + Reset)
		} else if out != "" {
			fmt.Print(FgHiWhite + out + Reset)
		}
	}
}
