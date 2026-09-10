# Lcode 语言设计与语法规范 (Language Specification)

Lcode 是一门静态类型、极简高效且富有表现力的自研编程语言。其语法吸收了现代语言（Go、Rust、TypeScript）的核心优点，旨在提供安全、清晰且高效的编译与运行体验。

---

## 1. 变量绑定与可变性 (Variables & Mutability)

在 Lcode 中，默认变量均为**不可变 (Immutable)**。只有使用 `mut` 关键字声明的变量才允许被重新赋值：

```lcode
// 1. 不可变变量 (带类型推导)
let pi = 3.1415926;
let title: string = "Lcode 编译器";

// 2. 可变变量 (Mutable)
mut counter = 0;
counter += 1; // 合法

// 以下将触发语义错误：
// pi = 3.0; // 错误：无法给不可变绑定 'pi' 赋值，请使用 'mut pi'
```

---

## 2. 数据类型 (Data Types)

- `int`：64 位有符号整型
- `float`：64 位双精度浮点型
- `bool`：布尔类型 (`true` / `false`)
- `string`：UTF-8 文本字符串（支持 `\n`, `\t`, `\"` 转义）
- `void`：无返回值类型

---

## 3. 极简语法与智能缩写机制 (Concise Syntax & Abbreviations)

为了提高编写效率，Lcode 支持极致简写的关键字与**最短唯一前缀推导机制**：

### 关键字与函数缩写：
- `prin(...)`：自动作为 `print(...)` 缩写（不换行输出）。
- `p(...)` 或 `pln(...)`：自动作为 `println(...)` 缩写（换行输出）。
- `ret`：等价于 `return`。
- `fun`：等价于 `fn`。

### 最短唯一前缀推导 (Unique Prefix Matching)：
如果一个函数名较长，只要调用时提供的缩写在该作用域下具有唯一候选者，编译器即自动推导绑定：
```lcode
fun calculate_trajectory_hypotenuse(x: int, y: int) -> int {
    ret x * x + y * y;
}

fun main() -> void {
    // 只要前缀唯一，即可使用简写调用
    let res = calculate_trajectory(3, 4);
    p("计算结果:", res);
}
```

---

## 4. 模块与库管理系统 (Modules & Imports)

支持标准库 (`std/`) 与本地模块的多文件拓扑依赖解析及环依赖检测：

```lcode
import "std/math";      // 导入标准数学库
import "std/time";      // 导入高精度时间库
import "std/mem";       // 导入内存管理库
import "std/error";     // 导入断言与错误库
import "std/cpp/core";  // 导入内置 C++ 高性能核心库
```

---

## 5. 错误处理与断言机制 (Error Handling & Assertions)

Lcode 提供内置运行时恐慌 (`panic`) 与断言判断 (`assert`)，崩溃时自动携带 Rust 风格的完整调用栈帧追踪：

```lcode
import "std/error";

// 1. 断言检查
assert(denominator != 0, "分母不可为零");

// 2. 运行时恐慌
if critical_error {
    panic("致命硬件或系统故障，程序安全中断！");
}

// 3. 状态码判断
let status = do_work();
if is_err(status) {
    p("捕获到非零错误状态码:", status);
}
```

---

## 6. 高精度时间管理 (Time Management)

提供系统毫秒/微秒级时间戳获取、睡眠暂停与耗时统计：

```lcode
import "std/time";

let t0 = now_ms();
sleep_ms(50); // 睡眠 50 毫秒
let elapsed = elapsed_ms(t0);
p("代码段耗时 (ms):", elapsed);
p("当前格式化时间:", format_now());
```

---

## 7. 虚拟堆内存管理与泄漏追踪 (Memory Management)

通过安全句柄管理虚拟堆内存分配、读写与双重释放检测，并具备零开销内存泄漏探针：

```lcode
import "std/mem";

// 申请 4 个 64 位整数的连续内存块
let ptr = alloc(4);

// 带有边界越界检查与释放后使用检查的读写
write(ptr, 0, 1024);
let v = read(ptr, 0);

// 查看堆分配指标
stats();

// 安全回收释放
free(ptr);

// 零泄漏检查
let leakCount = check_leaks();
```

---

## 8. 运行时调用栈技术 (Call Stack & Frame Inspection)

- `stack_dump()`：以结构化树形框打印当前调用栈帧、调用源文件及行号。
- `stack_depth()`：返回当前调用栈深度。
- 具备栈深度上限保护（默认 1000 层），防止递归无限死循环导致宿主崩溃。

---

## 9. 内置 C++ 核心库与转译后端 (Built-in C++ Library)

Lcode 不仅在 Go 解释执行器中直接模拟加速 C++ 内置函数，还提供 `to-cpp` 编译器后端，可一键输出自包含的现代化 C++17 代码并调用 `cpp/include/lcode_cpp.hpp`：

```bash
lcode to-cpp main.lc -o build/main.cpp
g++ -std=c++17 -Icpp/include build/main.cpp -o build/main
```
