package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"lcode/pkg/complete"
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
			fmt.Println("  :help             显示此帮助")
			fmt.Println("  :c <prefix>       查询指定前缀的智能补全候选项")
			fmt.Println("  :tokens <expr>    显示词法单元 Token 流")
			fmt.Println("  :ast <expr>       显示彩色语法树 AST")
			fmt.Println("  exit              退出 REPL")
			continue
		}

		// 自动补全命令支持: :c <prefix> 或 ? <prefix>
		if strings.HasPrefix(line, ":c ") || strings.HasPrefix(line, "? ") {
			prefix := strings.TrimSpace(line[3:])
			RenderCompletionList(prefix)
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

// RenderCompletionList 美化打印自动补全候选项
func RenderCompletionList(prefix string) {
	engine := complete.NewEngine()
	items := engine.Complete("", prefix)
	if len(items) == 0 {
		fmt.Printf("%s未找到匹配前缀 %q 的补全建议%s\n", ColorMuted, prefix, Reset)
		return
	}

	fmt.Printf("\n%s[智能自动补全建议] 前缀 %q 匹配到 %d 项:%s\n", ColorCyan+Bold, prefix, len(items), Reset)
	for _, it := range items {
		kindColor := ColorViolet
		if it.Kind == "Type" {
			kindColor = ColorEmerald
		} else if it.Kind == "Function" {
			kindColor = ColorAmber
		} else if it.Kind == "Module" {
			kindColor = ColorRose
		}
		badge := Paint(kindColor, fmt.Sprintf("[%-8s]", it.Kind))
		fmt.Printf("  • %s%-16s%s %s  %s%-28s%s %s%s%s\n",
			Bold+FgHiWhite, it.Label, Reset,
			badge,
			ColorCyan, it.Detail, Reset,
			ColorMuted, it.Documentation, Reset,
		)
	}
	fmt.Println()
}
