# ⚡ Lcode - 纯 Go 现代语言编译器与高颜值终端 Studio

> **完全基于纯 Go 语言打造** • **无需任何 HTML / Web 前端** • **自研专属现代编程语言语法** • **高颜值极客终端界面 (TUI)**

---

## 目录索引

- [1. 核心架构与设计哲学](#1-核心架构与设计哲学)
- [2. Lcode 自研语法速览](#2-lcode-自研语法速览)
- [3. 纯 Go 终端美观界面 (TUI)](#3-纯-go-终端美观界面-tui)
- [4. 项目完整目录结构](#4-项目完整目录结构)
- [5. 编译运行与命令指南](#5-编译运行与命令指南)
- [6. 核心模块与实现原理](#6-核心模块与实现原理)

---

## 1. 核心架构与设计哲学

传统编译器项目要么只有枯燥简陋的控制台输出，要么引入沉重且脱节的 Web 界面（HTML/CSS/JS）。
**Lcode** 贯彻纯粹的软件极客美学：
1. **全栈纯 Go (100% Go)**：从词法扫描、语法树构建、符号表/类型推导、解释执行，到终端高颜值可视化渲染，全部由 Go 标准库驱动，**零外部 CGO 或 Node.js 依赖**。
2. **专属自研语法 (Lcode Syntax)**：精心设计的现代强类型语言，融合不可变默认性 (`let` vs `mut`)、简洁函数语法 (`fn`)、优雅的无括号控制流。
3. **极客级终端界面 (Studio)**：采用 ANSI 24-bit TrueColor 与 Unicode 现代圆角线条（`╭─╮`、`│`、`╰─╯`），提供直观的多栏仪表盘、彩色树形 AST 拓扑图、Token 矩阵、符号表卡片以及媲美 Rust 的源码精准指针诊断。

---

## 2. Lcode 自研语法速览

详细规范请参阅 [docs/syntax.md](docs/syntax.md)。

```lcode
// 1. 函数声明 (支持类型标注与箭头返回值)
fn fibonacci(n: int) -> int {
    if n <= 1 {
        return n;
    }
    return fibonacci(n - 1) + fibonacci(n - 2);
}

// 2. 主函数入口
fn main() {
    // 不可变变量 (绑定后不可修改，安全可靠)
    let title: string = "=== Lcode 编译器 ===";
    println(title);

    // 可变变量声明 (使用 mut 关键字)
    mut counter = 0;
    mut total = 0;

    while counter < 10 {
        counter += 1;
        total += counter;
    }

    println("1 到 10 累加结果为: ", total);

    // 递归函数调用
    let fibVal = fibonacci(8);
    println("fibonacci(8) =", fibVal);
}
```

---

## 3. 纯 Go 终端美观界面 (TUI)

Lcode 包含内建的终端 UI 渲染引擎 (`pkg/ui`)：

### ① 编译器终端工作台 (`studio`)
```bash
go run . studio examples/hello.lc
```
一键输出包含：
- 📜 **带行号与高亮的源代码画卷**
- 🏷️ **彩色词法单元 (Token) 流表格**
- 🌲 **层次化彩色树状抽象语法树 (AST)**
- 🛡️ **作用域符号表与类型推导卡片**
- 💻 **程序运行标准输出与耗时统计**

### ② 精准现代代码错误诊断 (Diagnostics)
当代码出现语法或类型错误时，提供 Rust/Clang 级别的代码框架指引：
```text
╭─ ⚠️  代码诊断报告 (Compiler Diagnostics) ────────────────────────╮
│                                                                  │
│ [错误] 无法给不可变变量 "pi" 重新赋值 (提示: 可将变量声明为 'mut pi')│
│   --> examples/error_demo.lc:4:5                                 │
│    │                                                             │
│  3 │     let pi = 3.14;                                          │
│  4 │     pi = 3.0;                                               │
│    │     ^^^^ 变量被声明为不可变 'let'                            │
│  5 │ }                                                           │
│    │                                                             │
╰──────────────────────────────────────────────────────────────────╯
```

---

## 4. 项目完整目录结构

```text
Lcode/
├── go.mod                     # Go 模块文件 (lcode)
├── main.go                    # 项目根入口
├── cmd/
│   └── lcode/
│       └── main.go            # 编译器 CLI 独立可执行程序入口
├── docs/
│   └── syntax.md              # Lcode 自研语言详细语法规范与 EBNF
├── pkg/
│   ├── token/                 # 词法单元、关键字与源码位置 (Position)
│   │   └── token.go
│   ├── lexer/                 # 词法分析器与转义/注释处理
│   │   ├── lexer.go
│   │   └── lexer_test.go
│   ├── ast/                   # 抽象语法树节点结构与 Visitor 接口
│   │   ├── ast.go
│   │   └── visitor.go
│   ├── parser/                # 递归下降 + Pratt 运算符优先级解析器
│   │   ├── parser.go
│   │   └── parser_test.go
│   ├── sema/                  # 语义分析、符号表、作用域与不可变性检查
│   │   ├── symbol.go
│   │   ├── type.go
│   │   ├── analyzer.go
│   │   └── analyzer_test.go
│   ├── eval/                  # 语法树解释执行器与输出流捕获
│   │   └── eval.go
│   └── ui/                    # 纯 Go 高颜值终端 UI 引擎 (无 HTML)
│       ├── style.go           # TrueColor 转义码、调色板与 Unicode 边框
│       ├── diagnostic.go      # Rust 风格代码报错框与指针下划线
│       ├── token_view.go      # 词法单元表格视图
│       ├── ast_view.go        # 彩色树枝 AST 拓扑图
│       ├── symbol_view.go     # 符号表与类型分析对齐卡片
│       ├── tui.go             # 全流程 Terminal Studio 控制台
│       └── repl.go            # 彩色交互式 REPL 命令行
└── examples/
    ├── hello.lc               # 综合语言特性示例
    ├── fibonacci.lc           # 递归与控制流示例
    └── error_demo.lc          # 错误诊断与下划线提示测试用例
```

---

## 5. 编译运行与命令指南

### 1. 启动终端 Studio 可视化面板
```bash
go run . studio examples/hello.lc
```

### 2. 交互式 REPL 命令行
直接在终端键入代码即时验证与求值：
```bash
go run . repl
```

### 3. 彩色抽象语法树 (AST) 打印
```bash
go run . ast examples/fibonacci.lc
```

### 4. 词法单元列表展示
```bash
go run . tokenize examples/hello.lc
```

### 5. 编译并直接运行代码
```bash
go run . run examples/fibonacci.lc
```

### 6. 执行全流程静态检查与诊断
```bash
go run . check examples/error_demo.lc
```

### 7. 运行项目单元测试
```bash
go test ./... -v
```
