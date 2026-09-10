# Lcode 语言设计与语法规范 (Language Specification)

Lcode 是一门静态类型、简洁且富有表现力的自研编程语言。其语法吸收了现代语言（Go、Rust、TypeScript）的核心优点，旨在提供安全、清晰且高效的编译体验。

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

## 3. 函数声明 (Functions)

函数使用 `fn` 关键字定义，形参需显式指定类型，返回值类型使用箭头 `->` 指向：

```lcode
fn add(a: int, b: int) -> int {
    return a + b;
}

// 无返回值函数 (默认为 void)
fn greet(name: string) {
    println("Hello, ", name, "!");
}
```

---

## 4. 控制流 (Control Flow)

### 条件判断 (`if / else`)
条件表达式无需括号（亦可添加括号）：

```lcode
if score >= 90 {
    println("优秀");
} else if score >= 60 {
    println("及格");
} else {
    println("不及格");
}
```

### 循环 (`while`)
```lcode
mut i = 1;
mut sum = 0;

while i <= 100 {
    sum += i;
    i += 1;
}
```

---

## 5. 运算符体系 (Operators)

| 类别 | 运算符 | 说明 |
| :--- | :--- | :--- |
| **算术运算符** | `+`, `-`, `*`, `/`, `%` | 纯数值运算，`+` 支持字符串拼接 |
| **复合赋值** | `=`, `+=`, `-=`, `*=`, `/=` | 仅允许作用于 `mut` 变量 |
| **比较运算符** | `==`, `!=`, `<`, `<=`, `>`, `>=` | 返回 `bool` 类型 |
| **逻辑运算符** | `&&`, `||`, `!` | 严格布尔短路求值 |

---

## 6. 内置系统函数 (Built-in Functions)

- `println(...)`：格式化打印至控制台并换行，支持多参数。
- `print(...)`：格式化打印至控制台不换行。
