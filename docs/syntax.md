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

## 7. Rust 风格内存管理与所有权模型 (Rust-style Ownership & Memory Management)

Lcode 深度对齐了 Rust 的核心内存安全与所有权模型，杜绝垃圾回收器 (GC) 的停顿抖动，同时实现绝对的内存安全与零泄漏保证：

### 1. 单一所有权与移动语义 (Single Ownership & Move Semantics)
每个堆分配资源有且仅有一个所有者。当资源被赋值给新变量或通过实参按值传入函数时，发生所有权转移 (Move)。原变量立即失效，若尝试二次访问，将在语义编译期被直接拦截并抛出标准 Rust 错误码 `[E0382]`：

```lcode
let a = alloc(16);
let b = a;          // 所有权转移: a -> b
let val = read(a, 0); // ❌ 编译期拦截: [E0382] 使用已移动的所有权变量 (borrow of moved value)
```

### 2. 借用与引用运算符 (`&`) (Borrowing)
若需读取或借用变量内容而不剥夺所有权，可使用 `&` 借用运算符：

```lcode
fn inspect(buf: int) -> void {
    p("Buffer value at 0:", read(buf, 0));
}

let a = alloc(16);
inspect(&a);        // 借用传参：不转移所有权
let val = read(a, 0); // ✓ 正常访问：a 依然保持完整所有权
```

### 3. 显式销毁 (`drop(x)`)
支持 Rust 风格的 `drop(x)` 显式消耗并释放资源所有权。被 `drop` 的变量后续禁止再次使用：

```lcode
let res = alloc(32);
drop(res);          // 立即销毁并回收底层内存
```

### 4. RAII 作用域自动析构 (Zero-leak RAII Scope Drop)
当代码块 `{ ... }` 或函数退出时，当前作用域内所有未被移走的资源句柄将被系统**自动析构释放 (RAII Drop)**，无需手动编写 `free()`，保障天然的零内存泄漏：

```lcode
{
    let temp = alloc(64);
    write(temp, 0, 999);
    // temp 离开花括号代码块，自动触发析构 Drop，无任何泄漏！
}
assert(check_leaks() == 0, "离开作用域后堆内存自动回收清零");
```

### 5. 返回值所有权逃逸 (Ownership Escape)
若函数在作用域结束时将资源通过 `ret` 返回，该资源的所有权将安全逃逸至调用方父级作用域，不会被提早释放：

```lcode
fn make_buffer() -> int {
    let buf = alloc(128);
    ret buf; // 所有权安全逃逸至上层
}
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
