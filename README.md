# ⚡ Lcode - 纯 Go 现代语言编译器与高颜值终端 Studio

> **完全基于纯 Go 语言打造** • **无需任何 HTML / Web 前端** • **自研极简现代语法** • **高颜值极客终端界面 (TUI)** • **内置 C++ 核心库与自动补全**

---

## 目录索引

- [1. 核心架构与设计哲学](#1-核心架构与设计哲学)
- [2. 极简语法与智能缩写速览](#2-极简语法与智能缩写速览)
- [3. 运行时核心四大技术矩阵](#3-运行时核心四大技术矩阵)
  - [3.1 错误判断与异常恐慌控制 (Error & Panic)](#31-错误判断与异常恐慌控制-error--panic)
  - [3.2 高精度时间管理 (Time Management)](#32-高精度时间管理-time-management)
  - [3.3 虚拟堆内存管理与泄漏检测 (Memory Management)](#33-虚拟堆内存管理与泄漏检测-memory-management)
  - [3.4 运行时调用栈与栈帧追踪技术 (Call Stack Technology)](#34-运行时调用栈与栈帧追踪技术-call-stack-technology)
- [4. 内置 C++ 核心库与 C++17 转译编译](#4-内置-c-核心库与-c17-转译编译)
- [5. 智能代码自动补全引擎](#5-智能代码自动补全引擎)
- [6. 纯 Go 终端高颜值 Studio 界面](#6-纯-go-终端高颜值-studio-界面)
- [7. 命令行工具与使用指南](#7-命令行工具与使用指南)
- [8. 项目完整目录结构](#8-项目完整目录结构)

---

## 1. 核心架构与设计哲学

传统编译器项目要么只有枯燥简陋的控制台输出，要么引入沉重且脱节的 Web 界面（HTML/CSS/JS）。
**Lcode** 贯彻纯粹的现代软件极客美学：
1. **全栈纯 Go (100% Go)**：从词法扫描、语法树构建、符号表/类型推导、模块解析、到终端高颜值可视化渲染，全部由 Go 标准库打造，**零 Web 依赖**。
2. **专属自研语法 (Lcode Syntax)**：精心设计的现代强类型语言，融合不可变默认性 (`let` vs `mut`)、函数语法 (`fn` / `fun`)、以及极致简便的**缩写与最短唯一前缀推导机制**。
3. **极客级终端界面 (Studio)**：采用 ANSI 24-bit TrueColor 与 Unicode 现代圆角线条（`╭─╮`、`│`、`╰─╯`），提供直观的多栏仪表盘、彩色树形 AST 拓扑图、Token 矩阵、符号表卡片以及媲美 Rust 的源码精准指针诊断。
4. **内置 C++17 核心库**：兼备 Go 解释运行与转译输出现代 C++17 源码的双通道引擎。

---

## 2. 极简语法与智能缩写速览

详细规范请参阅 [docs/syntax.md](docs/syntax.md)。

Lcode 专门优化了手写体感，支持常用关键词与函数的超简缩写：
- `prin(...)`：自动替代 `print(...)`（无换行输出）。
- `p(...)` 或 `pln(...)`：自动替代 `println(...)`（自动换行输出）。
- `ret`：自动替代 `return`。
- `fun`：自动替代 `fn`。
- **最短唯一前缀推导**：例如函数 `calculate_trajectory_hypotenuse`，在当前作用域唯一时可直接写 `calculate_trajectory(...)` 调用！

```lcode
import "std/error";
import "std/time";
import "std/mem";

fun compute_sample(x: int) -> int {
    ret x * 10;
}

fun main() -> void {
    prin("计算中: ");
    let res = compute_sample(5);
    p(res); // 输出: 计算中: 50
}
```

---

## 3. 运行时核心四大技术矩阵

### 3.1 错误判断与异常恐慌控制 (Error & Panic)
- 内置 `assert(condition, message)` 断言判定函数。
- 内置 `panic(message)` 运行时恐慌函数，崩溃时自动携带 Rust 风格的完整调用栈帧追踪。
- 标准库 `std/error.lc` 提供 `assert_true`、`assert_eq`、`is_ok`、`is_err` 状态码辅助。

### 3.2 高精度时间管理 (Time Management)
- 内置毫秒级 `time_now_ms()`、微秒级 `time_now_micros()` 时间戳获取。
- 线程休眠控制 `time_sleep_ms(ms)` 与耗时基准统计 `time_elapsed_ms(start)`。
- 标准库 `std/time.lc` 提供开箱即用的高阶时间与格式化接口。

### 3.3 虚拟堆内存管理与泄漏检测 (Memory Management)
- 虚拟堆内存管理器：`mem_alloc(size)` 分配连续字长内存块，返回唯一安全句柄。
- 安全读写：`mem_write(h, off, val)` 与 `mem_read(h, off)` 自带越界拦截与释放后使用 (UAF) 检查。
- 生命周期控制：`mem_free(h)` 自带双重释放 (Double Free) 检测。
- 零开销泄漏诊断：`mem_stats()` 打印堆指标，`mem_check_leaks()` 自动探查未释放块并告警。

### 3.4 运行时调用栈与栈帧追踪技术 (Call Stack Technology)
- 结构化 `CallStack` 与 `CallFrame` 深度管理。
- 内置 `stack_dump()` 与 `stack_depth()`，随时以树形框输出函数调用链。
- 递归栈溢出深度保护（默认 1000 层），防止递归死循环耗尽宿主资源。

---

## 4. 内置 C++ 核心库与 C++17 转译编译

内置纯头文件现代化 C++17 标准库 `cpp/include/lcode_cpp.hpp`：
- 高性能数学开方、快速幂、绝对值运算 (`lcode::math`)；
- 经典算法求最大公约数、素数检验、快速斐波那契 (`lcode::algo`)；
- 内存分配池与高精度计时工具 (`lcode::mem`, `lcode::time_mgr`)。

通过 `lcode to-cpp` 可将任意 Lcode 源码直接转译为自包含的现代 C++ 工程：
```bash
lcode to-cpp examples/concise_demo.lc -o build/concise_demo.cpp
g++ -std=c++17 -Icpp/include build/concise_demo.cpp -o build/concise_demo
./build/concise_demo
```

---

## 5. 智能代码自动补全引擎

Lcode 配备了静态语法与上下文感知的自动补全引擎：
1. **CLI 补全查询**：
   ```bash
   lcode complete prin
   lcode complete cpp
   ```
   以极高颜值的彩色徽章展示候选词、类别（Keyword/Type/Function/Module）、签名与文档说明。
2. **REPL 交互补全**：
   在交互式命令行中输入 `:c <前缀>`（例如 `:c pr`），即刻弹出智能建议卡片。

---

## 6. 纯 Go 终端高颜值 Studio 界面

运行全流程可视化终端 Studio：
```bash
lcode studio examples/error_demo.lc
```
包含模块：
- 📝 **带行号与高亮的源代码画卷**
- 🏷️ **彩色词法单元 (Token) 流表格**
- 🌲 **层次化彩色树状抽象语法树 (AST)**
- 🛡️ **作用域符号表与类型推导卡片**
- 💻 **控制台标准输出面板**
- 🧠 **运行时堆内存与泄漏探针监控卡片**
- 🥞 **崩溃调用栈与回溯卡片**

---

## 7. 命令行工具与使用指南

| 命令 | 说明 |
| :--- | :--- |
| `lcode studio [file]` | 启动全套终端编译器 Studio (默认 `examples/hello.lc`) |
| `lcode repl` | 启动交互式命令行 REPL (支持 `:c` 补全) |
| `lcode run <file>` | 编译并执行源程序 (自动解析模块导入与缩写) |
| `lcode to-cpp <file> [-o out]` | 将源文件转译为现代 C++17 源码并调用内置 C++ 库 |
| `lcode complete <prefix>` | 智能代码自动补全建议查询 (支持 `--json` 格式) |
| `lcode init` | 初始化当前目录为 Lcode 项目 (生成 `lcode.toml`) |
| `lcode ast <file>` | 以彩色树形分支结构输出抽象语法树 (AST) |
| `lcode tokenize <file>` | 以美化表格输出词法单元 (Token) 流 |
| `lcode check <file>` | 执行编译前端全流程检查与精准代码诊断 |
| `lcode version` | 显示版本信息 |

---

## 8. 项目完整目录结构

```text
Lcode/
├── go.mod                     # Go 模块文件 (lcode)
├── lcode.toml                 # Lcode 包管理器项目描述文件
├── main.go                    # 项目根入口 (支持 go run .)
├── cmd/
│   └── lcode/
│       └── main.go            # 编译器 CLI 独立可执行程序入口
├── cpp/
│   └── include/
│       └── lcode_cpp.hpp      # 内置高性能现代 C++17 纯头文件核心库
├── std/                       # Lcode 标准库体系
│   ├── math.lc                # 常用数学函数库
│   ├── algo.lc                # 经典算法库
│   ├── io.lc                  # 格式化与控制台工具库
│   ├── time.lc                # 高精度时间与耗时库
│   ├── mem.lc                 # 虚拟堆内存与泄漏检测库
│   ├── error.lc               # 错误处理与断言库
│   └── cpp/
│       └── core.lc            # 内置 C++ 核心库 Lcode 绑定接口
├── docs/
│   └── syntax.md              # 自研语言详细语法规范与缩写机制
├── pkg/
│   ├── token/                 # 词法单元、关键字与源码位置
│   ├── lexer/                 # 词法分析器与转义/注释处理
│   ├── ast/                   # 抽象语法树节点结构与 Visitor 接口
│   ├── parser/                # 递归下降 + Pratt 运算符优先级解析器
│   ├── mod/                   # 模块加载器、多文件导入与拓扑循环检测
│   ├── sema/                  # 语义分析、符号表、不可变性与缩写解析
│   ├── complete/              # 智能代码自动补全引擎
│   ├── codegen/
│   │   └── cpp/               # 现代 C++17 代码生成转译后端
│   ├── eval/                  # 解释执行器、调用栈追踪与虚拟内存管理器
│   └── ui/                    # 纯 Go 高颜值终端 UI 引擎 (无 HTML)
└── examples/
    ├── hello.lc               # 综合语言特性示例
    ├── concise_demo.lc        # 极简语法与智能缩写演示
    ├── error_demo.lc          # 错误处理、时间、内存与调用栈全景演示
    └── cpp_demo.lc            # 内置 C++ 核心库混合加速演示
```
