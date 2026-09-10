package eval

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"strings"

	"lcode/pkg/ast"
)

// Value 表示运行时的动态值
type Value interface {
	String() string
	Type() string
}

type IntValue struct{ Value int64 }
func (v IntValue) String() string { return fmt.Sprintf("%d", v.Value) }
func (v IntValue) Type() string   { return "int" }

type FloatValue struct{ Value float64 }
func (v FloatValue) String() string { return fmt.Sprintf("%f", v.Value) }
func (v FloatValue) Type() string   { return "float" }

type BoolValue struct{ Value bool }
func (v BoolValue) String() string { return fmt.Sprintf("%t", v.Value) }
func (v BoolValue) Type() string   { return "bool" }

type StringValue struct{ Value string }
func (v StringValue) String() string { return v.Value }
func (v StringValue) Type() string   { return "string" }

type VoidValue struct{}
func (v VoidValue) String() string { return "void" }
func (v VoidValue) Type() string   { return "void" }

// ArrayValue 动态数组值
type ArrayValue struct {
	Elements []Value
}
func (v *ArrayValue) String() string {
	parts := make([]string, 0, len(v.Elements))
	for _, el := range v.Elements {
		parts = append(parts, el.String())
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
func (v *ArrayValue) Type() string { return "array" }

// StructInstanceValue 自定义结构体实例
type StructInstanceValue struct {
	StructName string
	Fields     map[string]Value
}
func (v *StructInstanceValue) String() string {
	parts := make([]string, 0, len(v.Fields))
	for k, val := range v.Fields {
		parts = append(parts, fmt.Sprintf("%s: %s", k, val.String()))
	}
	return fmt.Sprintf("%s { %s }", v.StructName, strings.Join(parts, ", "))
}
func (v *StructInstanceValue) Type() string { return v.StructName }

// BreakSignal 与 ContinueSignal 控制流跳转
type BreakSignal struct{}
func (b BreakSignal) String() string { return "break" }
func (b BreakSignal) Type() string   { return "signal" }

type ContinueSignal struct{}
func (c ContinueSignal) String() string { return "continue" }
func (c ContinueSignal) Type() string   { return "signal" }

// ReturnValue 用于控制流返回包装
type ReturnValue struct{ Value Value }
func (v ReturnValue) String() string { return v.Value.String() }
func (v ReturnValue) Type() string   { return v.Value.Type() }

// FunctionValue 函数闭包
type FunctionValue struct {
	Decl *ast.FuncDecl
	Env  *Environment
}
func (v FunctionValue) String() string { return fmt.Sprintf("<func %s>", v.Decl.Name.Value) }
func (v FunctionValue) Type() string   { return "function" }

// Environment 运行时环境
type Environment struct {
	parent       *Environment
	store        map[string]Value
	stdout       io.Writer
	ownedHandles []int64 // Rust-style RAII: 当前作用域持有的资源句柄
}

func NewEnvironment(parent *Environment, stdout io.Writer) *Environment {
	if stdout == nil && parent != nil {
		stdout = parent.stdout
	}
	return &Environment{
		parent:       parent,
		store:        make(map[string]Value),
		stdout:       stdout,
		ownedHandles: make([]int64, 0),
	}
}

func (e *Environment) Get(name string) (Value, bool) {
	if val, ok := e.store[name]; ok {
		return val, true
	}
	if e.parent != nil {
		return e.parent.Get(name)
	}
	return nil, false
}

func (e *Environment) Set(name string, val Value) {
	e.store[name] = val
}

func (e *Environment) Assign(name string, val Value) bool {
	if _, ok := e.store[name]; ok {
		e.store[name] = val
		return true
	}
	if e.parent != nil {
		return e.parent.Assign(name, val)
	}
	return false
}

// Evaluator 解释执行器
type Evaluator struct {
	stdout     *bytes.Buffer
	CallStack  *CallStack
	MemManager *MemoryManager
}

func New() *Evaluator {
	return &Evaluator{
		stdout:     new(bytes.Buffer),
		CallStack:  NewCallStack(1000),
		MemManager: NewMemoryManager(),
	}
}

// RunProgram 执行整个 AST 程序并返回控制台输出
func (ev *Evaluator) RunProgram(program *ast.Program) (string, error) {
	ev.stdout.Reset()
	ev.CallStack = NewCallStack(1000)
	ev.MemManager = NewMemoryManager()
	globalEnv := NewEnvironment(nil, ev.stdout)

	// 1. 预先注册所有顶层函数
	for _, decl := range program.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			globalEnv.Set(fn.Name.Value, FunctionValue{Decl: fn, Env: globalEnv})
		}
	}

	// 2. 执行顶层语句
	for _, stmt := range program.Stmts {
		res, err := ev.Eval(stmt, globalEnv)
		if err != nil {
			return ev.stdout.String(), err
		}
		if _, ok := res.(ReturnValue); ok {
			break
		}
	}

	// 3. 如果存在 main 函数，则自动调用 main()
	if mainFunc, ok := globalEnv.Get("main"); ok {
		if fn, ok := mainFunc.(FunctionValue); ok {
			fnEnv := NewEnvironment(fn.Env, ev.stdout)
			mainFrame := &CallFrame{
				FuncName: "main",
				Filename: fn.Decl.Token.Pos.Filename,
				Line:     fn.Decl.Token.Pos.Line,
				Args:     []string{},
				Locals:   make(map[string]string),
			}
			_ = ev.CallStack.Push(mainFrame)
			defer ev.CallStack.Pop()

			_, err := ev.evalBlockStmt(fn.Decl.Body, fnEnv)
			if err != nil {
				return ev.stdout.String(), err
			}
		}
	}

	return ev.stdout.String(), nil
}

// Eval 递归对 AST 节点求值
func (ev *Evaluator) Eval(node ast.Node, env *Environment) (Value, error) {
	if node == nil {
		return VoidValue{}, nil
	}

	switch n := node.(type) {
	case *ast.IntegerLiteral:
		return IntValue{Value: n.Value}, nil

	case *ast.FloatLiteral:
		return FloatValue{Value: n.Value}, nil

	case *ast.StringLiteral:
		return StringValue{Value: n.Value}, nil

	case *ast.BooleanLiteral:
		return BoolValue{Value: n.Value}, nil

	case *ast.ArrayLiteral:
		elems := make([]Value, 0, len(n.Elements))
		for _, el := range n.Elements {
			v, err := ev.Eval(el, env)
			if err != nil {
				return nil, err
			}
			elems = append(elems, v)
		}
		return &ArrayValue{Elements: elems}, nil

	case *ast.IndexExpr:
		target, err := ev.Eval(n.Left, env)
		if err != nil {
			return nil, err
		}
		idxVal, err := ev.Eval(n.Index, env)
		if err != nil {
			return nil, err
		}
		idxInt, ok := idxVal.(IntValue)
		if !ok {
			return nil, fmt.Errorf("索引必须是整型")
		}
		idx := int(idxInt.Value)
		if arr, ok := target.(*ArrayValue); ok {
			if idx < 0 || idx >= len(arr.Elements) {
				return nil, fmt.Errorf("数组索引越界: index %d, len %d", idx, len(arr.Elements))
			}
			return arr.Elements[idx], nil
		}
		if str, ok := target.(StringValue); ok {
			runes := []rune(str.Value)
			if idx < 0 || idx >= len(runes) {
				return nil, fmt.Errorf("字符串索引越界: index %d, len %d", idx, len(runes))
			}
			return StringValue{Value: string(runes[idx])}, nil
		}
		return nil, fmt.Errorf("目标类型 %s 不支持索引访问", target.Type())

	case *ast.MemberExpr:
		target, err := ev.Eval(n.Object, env)
		if err != nil {
			return nil, err
		}
		if st, ok := target.(*StructInstanceValue); ok {
			if val, ok := st.Fields[n.Property.Value]; ok {
				return val, nil
			}
			return nil, fmt.Errorf("结构体 %s 没有字段 %q", st.StructName, n.Property.Value)
		}
		return nil, fmt.Errorf("无法在非结构体类型 %s 上访问属性 %q", target.Type(), n.Property.Value)

	case *ast.StructLiteral:
		fields := make(map[string]Value)
		for fname, fexpr := range n.Fields {
			fval, err := ev.Eval(fexpr, env)
			if err != nil {
				return nil, err
			}
			fields[fname] = fval
		}
		return &StructInstanceValue{
			StructName: n.Name.Value,
			Fields:     fields,
		}, nil

	case *ast.Identifier:
		if val, ok := env.Get(n.Value); ok {
			return val, nil
		}
		return nil, fmt.Errorf("%s: 运行时未定义变量 %q", n.Pos(), n.Value)

	case *ast.GroupedExpr:
		return ev.Eval(n.Expression, env)

	case *ast.UnaryExpr:
		right, err := ev.Eval(n.Right, env)
		if err != nil {
			return nil, err
		}
		return ev.evalUnary(n.Operator, right)

	case *ast.BinaryExpr:
		left, err := ev.Eval(n.Left, env)
		if err != nil {
			return nil, err
		}
		right, err := ev.Eval(n.Right, env)
		if err != nil {
			return nil, err
		}
		return ev.evalBinary(n.Operator, left, right)

	case *ast.VarDeclStmt:
		var initVal Value = VoidValue{}
		if n.Value != nil {
			val, err := ev.Eval(n.Value, env)
			if err != nil {
				return nil, err
			}
			initVal = val
		}
		env.Set(n.Name.Value, initVal)
		return initVal, nil

	case *ast.AssignStmt:
		rightVal, err := ev.Eval(n.Right, env)
		if err != nil {
			return nil, err
		}

		switch lhs := n.Left.(type) {
		case *ast.Identifier:
			if n.Operator != "=" {
				currVal, exists := env.Get(lhs.Value)
				if !exists {
					return nil, fmt.Errorf("未定义的变量 %s", lhs.Value)
				}
				rawOp := strings.TrimSuffix(n.Operator, "=")
				newVal, err := ev.evalBinary(rawOp, currVal, rightVal)
				if err != nil {
					return nil, err
				}
				rightVal = newVal
			}
			if !env.Assign(lhs.Value, rightVal) {
				env.Set(lhs.Value, rightVal)
			}
			return rightVal, nil

		case *ast.IndexExpr:
			target, err := ev.Eval(lhs.Left, env)
			if err != nil {
				return nil, err
			}
			arr, ok := target.(*ArrayValue)
			if !ok {
				return nil, fmt.Errorf("只能对数组执行索引赋值")
			}
			idxVal, err := ev.Eval(lhs.Index, env)
			if err != nil {
				return nil, err
			}
			idxInt, ok := idxVal.(IntValue)
			if !ok {
				return nil, fmt.Errorf("数组索引必须为整型")
			}
			idx := int(idxInt.Value)
			if idx < 0 || idx >= len(arr.Elements) {
				return nil, fmt.Errorf("数组索引越界: index %d, len %d", idx, len(arr.Elements))
			}
			if n.Operator != "=" {
				rawOp := strings.TrimSuffix(n.Operator, "=")
				newVal, err := ev.evalBinary(rawOp, arr.Elements[idx], rightVal)
				if err != nil {
					return nil, err
				}
				rightVal = newVal
			}
			arr.Elements[idx] = rightVal
			return rightVal, nil

		case *ast.MemberExpr:
			target, err := ev.Eval(lhs.Object, env)
			if err != nil {
				return nil, err
			}
			st, ok := target.(*StructInstanceValue)
			if !ok {
				return nil, fmt.Errorf("只能对结构体实例执行属性赋值")
			}
			if n.Operator != "=" {
				currVal, ok := st.Fields[lhs.Property.Value]
				if !ok {
					return nil, fmt.Errorf("结构体 %s 没有字段 %q", st.StructName, lhs.Property.Value)
				}
				rawOp := strings.TrimSuffix(n.Operator, "=")
				newVal, err := ev.evalBinary(rawOp, currVal, rightVal)
				if err != nil {
					return nil, err
				}
				rightVal = newVal
			}
			st.Fields[lhs.Property.Value] = rightVal
			return rightVal, nil

		default:
			return nil, fmt.Errorf("非法赋值左值目标")
		}

	case *ast.BlockStmt:
		return ev.evalBlockStmt(n, env)

	case *ast.ExprStmt:
		return ev.Eval(n.Expression, env)

	case *ast.IfStmt:
		cond, err := ev.Eval(n.Condition, env)
		if err != nil {
			return nil, err
		}
		bCond, ok := cond.(BoolValue)
		if !ok {
			return nil, fmt.Errorf("if 条件必须为 bool")
		}

		if bCond.Value {
			return ev.Eval(n.Consequence, env)
		} else if n.Alternative != nil {
			return ev.Eval(n.Alternative, env)
		}
		return VoidValue{}, nil

	case *ast.WhileStmt:
		for {
			cond, err := ev.Eval(n.Condition, env)
			if err != nil {
				return nil, err
			}
			bCond, ok := cond.(BoolValue)
			if !ok {
				return nil, fmt.Errorf("while 条件必须为 bool")
			}
			if !bCond.Value {
				break
			}

			res, err := ev.Eval(n.Body, env)
			if err != nil {
				return nil, err
			}
			if _, ok := res.(BreakSignal); ok {
				break
			}
			if _, ok := res.(ContinueSignal); ok {
				continue
			}
			if ret, ok := res.(ReturnValue); ok {
				return ret, nil
			}
		}
		return VoidValue{}, nil

	case *ast.ForInStmt:
		iterVal, err := ev.Eval(n.Iterable, env)
		if err != nil {
			return nil, err
		}
		arr, ok := iterVal.(*ArrayValue)
		if !ok {
			return nil, fmt.Errorf("for ... in 只能遍历 array, 遇到 %s", iterVal.Type())
		}
		for _, item := range arr.Elements {
			loopEnv := NewEnvironment(env, env.stdout)
			loopEnv.Set(n.VarName.Value, item)
			res, err := ev.evalBlockStmt(n.Body, loopEnv)
			if err != nil {
				return nil, err
			}
			if _, ok := res.(BreakSignal); ok {
				break
			}
			if _, ok := res.(ContinueSignal); ok {
				continue
			}
			if ret, ok := res.(ReturnValue); ok {
				return ret, nil
			}
		}
		return VoidValue{}, nil

	case *ast.BreakStmt:
		return BreakSignal{}, nil

	case *ast.ContinueStmt:
		return ContinueSignal{}, nil

	case *ast.TryCatchStmt:
		tryEnv := NewEnvironment(env, env.stdout)
		res, err := ev.evalBlockStmt(n.TryBlock, tryEnv)
		if err != nil {
			// 捕获异常，容错恢复
			catchEnv := NewEnvironment(env, env.stdout)
			if n.ErrVar != nil {
				catchEnv.Set(n.ErrVar.Value, StringValue{Value: err.Error()})
			}
			return ev.evalBlockStmt(n.CatchBlock, catchEnv)
		}
		return res, nil

	case *ast.StructDecl:
		return VoidValue{}, nil

	case *ast.ReturnStmt:
		if n.ReturnValue != nil {
			val, err := ev.Eval(n.ReturnValue, env)
			if err != nil {
				return nil, err
			}
			return ReturnValue{Value: val}, nil
		}
		return ReturnValue{Value: VoidValue{}}, nil

	case *ast.CallExpr:
		return ev.evalCall(n, env)
	}

	return VoidValue{}, nil
}

func (ev *Evaluator) evalBlockStmt(block *ast.BlockStmt, env *Environment) (Value, error) {
	if block == nil {
		return VoidValue{}, nil
	}
	subEnv := NewEnvironment(env, env.stdout)
	defer func() {
		// Rust 风格 RAII: 当前代码块作用域退出，自动 Drop 析构本块内持有的未释放堆资源
		for _, handle := range subEnv.ownedHandles {
			if b, ok := ev.MemManager.blocks[handle]; ok && !b.Freed {
				_ = ev.MemManager.Free(handle)
			}
		}
	}()

	for _, stmt := range block.Statements {
		res, err := ev.Eval(stmt, subEnv)
		if err != nil {
			return nil, err
		}
		if _, ok := res.(BreakSignal); ok {
			return res, nil
		}
		if _, ok := res.(ContinueSignal); ok {
			return res, nil
		}
		if ret, ok := res.(ReturnValue); ok {
			// 如果返回的是句柄值，发生所有权逃逸转移给父级环境，当前作用域不提前 drop！
			if iv, ok := ret.Value.(IntValue); ok {
				for i, h := range subEnv.ownedHandles {
					if h == iv.Value {
						if env != nil {
							env.ownedHandles = append(env.ownedHandles, h)
						}
						subEnv.ownedHandles = append(subEnv.ownedHandles[:i], subEnv.ownedHandles[i+1:]...)
						break
					}
				}
			}
			return ret, nil
		}
	}
	return VoidValue{}, nil
}

func (ev *Evaluator) evalCall(call *ast.CallExpr, env *Environment) (Value, error) {
	ident, ok := call.Function.(*ast.Identifier)
	if !ok {
		return nil, fmt.Errorf("只支持直接函数名调用")
	}

	// 1. 内置函数拦截 (IO, Error, Stack, Time, Memory, C++)
	if val, handled, err := ev.evalBuiltinCall(ident.Value, call, env); handled {
		return val, err
	}

	// 2. 用户自定义函数
	fnVal, ok := env.Get(ident.Value)
	if !ok {
		return nil, fmt.Errorf("未定义的函数 %q", ident.Value)
	}
	fn, ok := fnVal.(FunctionValue)
	if !ok {
		return nil, fmt.Errorf("%q 不是函数", ident.Value)
	}

	// 评估实参
	argVals := make([]Value, 0, len(call.Arguments))
	argStrs := make([]string, 0, len(call.Arguments))
	for _, arg := range call.Arguments {
		argVal, err := ev.Eval(arg, env)
		if err != nil {
			return nil, err
		}
		argVals = append(argVals, argVal)
		argStrs = append(argStrs, argVal.String())
	}

	// 压入调用栈帧 (带有栈深度限制保护)
	frame := &CallFrame{
		FuncName: fn.Decl.Name.Value,
		Filename: fn.Decl.Token.Pos.Filename,
		Line:     call.Token.Pos.Line,
		Args:     argStrs,
		Locals:   make(map[string]string),
	}
	if err := ev.CallStack.Push(frame); err != nil {
		return nil, fmt.Errorf("%s\n%s", err, ev.CallStack.FormatTrace())
	}
	defer ev.CallStack.Pop()

	fnEnv := NewEnvironment(fn.Env, env.stdout)
	for i, param := range fn.Decl.Params {
		if i < len(argVals) {
			fnEnv.Set(param.Name.Value, argVals[i])
		}
	}

	res, err := ev.evalBlockStmt(fn.Decl.Body, fnEnv)
	if err != nil {
		return nil, err
	}
	if ret, ok := res.(ReturnValue); ok {
		return ret.Value, nil
	}
	return VoidValue{}, nil
}

func (ev *Evaluator) evalUnary(op string, right Value) (Value, error) {
	switch op {
	case "&", "*":
		// Rust 风格借用/解引用运算符
		return right, nil
	case "!":
		if b, ok := right.(BoolValue); ok {
			return BoolValue{Value: !b.Value}, nil
		}
		return nil, fmt.Errorf("! 只能作用于 bool")
	case "-":
		if i, ok := right.(IntValue); ok {
			return IntValue{Value: -i.Value}, nil
		}
		if f, ok := right.(FloatValue); ok {
			return FloatValue{Value: -f.Value}, nil
		}
		return nil, fmt.Errorf("- 只能作用于数值")
	}
	return nil, fmt.Errorf("未知一元运算符: %s", op)
}

func (ev *Evaluator) evalBinary(op string, left, right Value) (Value, error) {
	// 字符串拼接
	if op == "+" {
		if l, ok := left.(StringValue); ok {
			return StringValue{Value: l.Value + right.String()}, nil
		}
		if r, ok := right.(StringValue); ok {
			return StringValue{Value: left.String() + r.Value}, nil
		}
	}

	// 整型运算
	if l, ok := left.(IntValue); ok {
		if r, ok := right.(IntValue); ok {
			switch op {
			case "+": return IntValue{Value: l.Value + r.Value}, nil
			case "-": return IntValue{Value: l.Value - r.Value}, nil
			case "*": return IntValue{Value: l.Value * r.Value}, nil
			case "/":
				if r.Value == 0 {
					return nil, fmt.Errorf("除以零错误")
				}
				return IntValue{Value: l.Value / r.Value}, nil
			case "%":
				if r.Value == 0 {
					return nil, fmt.Errorf("除以零错误")
				}
				return IntValue{Value: l.Value % r.Value}, nil
			case "==": return BoolValue{Value: l.Value == r.Value}, nil
			case "!=": return BoolValue{Value: l.Value != r.Value}, nil
			case "<":  return BoolValue{Value: l.Value < r.Value}, nil
			case "<=": return BoolValue{Value: l.Value <= r.Value}, nil
			case ">":  return BoolValue{Value: l.Value > r.Value}, nil
			case ">=": return BoolValue{Value: l.Value >= r.Value}, nil
			}
		}
	}

	// 浮点数运算
	if l, ok := left.(FloatValue); ok {
		if r, ok := right.(FloatValue); ok {
			switch op {
			case "+": return FloatValue{Value: l.Value + r.Value}, nil
			case "-": return FloatValue{Value: l.Value - r.Value}, nil
			case "*": return FloatValue{Value: l.Value * r.Value}, nil
			case "/": return FloatValue{Value: l.Value / r.Value}, nil
			case "==": return BoolValue{Value: l.Value == r.Value}, nil
			case "!=": return BoolValue{Value: l.Value != r.Value}, nil
			case "<":  return BoolValue{Value: l.Value < r.Value}, nil
			case "<=": return BoolValue{Value: l.Value <= r.Value}, nil
			case ">":  return BoolValue{Value: l.Value > r.Value}, nil
			case ">=": return BoolValue{Value: l.Value >= r.Value}, nil
			}
		}
	}

	// 布尔逻辑运算
	if l, ok := left.(BoolValue); ok {
		if r, ok := right.(BoolValue); ok {
			switch op {
			case "&&": return BoolValue{Value: l.Value && r.Value}, nil
			case "||": return BoolValue{Value: l.Value || r.Value}, nil
			case "==": return BoolValue{Value: l.Value == r.Value}, nil
			case "!=": return BoolValue{Value: l.Value != r.Value}, nil
			}
		}
	}

	return nil, fmt.Errorf("不支持在类型 %s 与 %s 之间应用运算符 %q", left.Type(), right.Type(), op)
}

func (ev *Evaluator) evalBuiltinCall(name string, call *ast.CallExpr, env *Environment) (Value, bool, error) {
	switch name {
	case "println", "print", "prin", "pri", "pr", "p", "pln", "prn":
		args := make([]string, 0, len(call.Arguments))
		for _, arg := range call.Arguments {
			val, err := ev.Eval(arg, env)
			if err != nil {
				return nil, true, err
			}
			args = append(args, val.String())
		}
		outText := strings.Join(args, " ")
		if name == "println" || name == "p" || name == "pln" || name == "prn" {
			outText += "\n"
		}
		_, _ = env.stdout.Write([]byte(outText))
		return VoidValue{}, true, nil

	case "panic":
		msg := "运行时恐慌 (Panic)"
		if len(call.Arguments) > 0 {
			v, err := ev.Eval(call.Arguments[0], env)
			if err != nil {
				return nil, true, err
			}
			msg = v.String()
		}
		trace := ev.CallStack.FormatTrace()
		return nil, true, fmt.Errorf("【运行时恐慌 (Panic)】: %s\n%s", msg, trace)

	case "assert":
		if len(call.Arguments) < 1 {
			return nil, true, fmt.Errorf("assert 至少需要 1 个布尔表达式参数")
		}
		condVal, err := ev.Eval(call.Arguments[0], env)
		if err != nil {
			return nil, true, err
		}
		bCond, ok := condVal.(BoolValue)
		if !ok {
			return nil, true, fmt.Errorf("assert 第一个参数必须为 bool 类型")
		}
		if !bCond.Value {
			msg := "断言判定失败 (Assertion Failed)"
			if len(call.Arguments) >= 2 {
				v, _ := ev.Eval(call.Arguments[1], env)
				if v != nil {
					msg = v.String()
				}
			}
			trace := ev.CallStack.FormatTrace()
			return nil, true, fmt.Errorf("【断言失败 (Assertion Failed)】: %s\n%s", msg, trace)
		}
		return VoidValue{}, true, nil

	case "stack_dump":
		trace := ev.CallStack.FormatTrace()
		outText := fmt.Sprintf("\n🥞 当前运行时调用栈回溯 (深度 %d 层):\n%s\n", ev.CallStack.Depth(), trace)
		_, _ = env.stdout.Write([]byte(outText))
		return VoidValue{}, true, nil

	case "stack_depth":
		return IntValue{Value: int64(ev.CallStack.Depth())}, true, nil

	case "time_now_ms":
		return IntValue{Value: TimeNowMs()}, true, nil

	case "time_now_secs":
		return IntValue{Value: TimeNowSecs()}, true, nil

	case "time_now_micros":
		return IntValue{Value: TimeNowMicros()}, true, nil

	case "time_sleep_ms":
		if len(call.Arguments) > 0 {
			v, err := ev.Eval(call.Arguments[0], env)
			if err == nil {
				if iv, ok := v.(IntValue); ok {
					TimeSleepMs(iv.Value)
				}
			}
		}
		return VoidValue{}, true, nil

	case "time_elapsed_ms":
		if len(call.Arguments) > 0 {
			v, err := ev.Eval(call.Arguments[0], env)
			if err == nil {
				if iv, ok := v.(IntValue); ok {
					return IntValue{Value: TimeNowMs() - iv.Value}, true, nil
				}
			}
		}
		return IntValue{Value: 0}, true, nil

	case "time_format_now":
		return StringValue{Value: TimeFormatNow()}, true, nil

	case "mem_alloc", "alloc":
		if len(call.Arguments) < 1 {
			return nil, true, fmt.Errorf("mem_alloc 需要指定分配大小")
		}
		sVal, err := ev.Eval(call.Arguments[0], env)
		if err != nil {
			return nil, true, err
		}
		iVal, ok := sVal.(IntValue)
		if !ok {
			return nil, true, fmt.Errorf("mem_alloc 大小必须为整型")
		}
		curFn := "main"
		if ev.CallStack.Depth() > 0 {
			curFn = ev.CallStack.frames[ev.CallStack.Depth()-1].FuncName
		}
		h, err := ev.MemManager.Alloc(int(iVal.Value), curFn, call.Pos().Line)
		if err != nil {
			return nil, true, err
		}
		// 记录当前作用域持有该句柄，作用域结束时若未转移将自动 RAII Drop
		if env != nil {
			env.ownedHandles = append(env.ownedHandles, h)
		}
		return IntValue{Value: h}, true, nil

	case "mem_free", "free", "drop":
		if len(call.Arguments) < 1 {
			return nil, true, fmt.Errorf("mem_free/drop 需要指定内存句柄")
		}
		hVal, err := ev.Eval(call.Arguments[0], env)
		if err != nil {
			return nil, true, err
		}
		iVal, ok := hVal.(IntValue)
		if !ok {
			return nil, true, fmt.Errorf("mem_free/drop 句柄必须为整型")
		}
		err = ev.MemManager.Free(iVal.Value)
		if err != nil {
			return nil, true, err
		}
		// 显式释放后从当前作用域拥有列表中移除，避免作用域退出时重复 drop
		if env != nil {
			for i, h := range env.ownedHandles {
				if h == iVal.Value {
					env.ownedHandles = append(env.ownedHandles[:i], env.ownedHandles[i+1:]...)
					break
				}
			}
		}
		return IntValue{Value: 0}, true, nil

	case "mem_write", "write":
		if len(call.Arguments) < 3 {
			return nil, true, fmt.Errorf("mem_write 需要 (handle, offset, value)")
		}
		h, err := ev.Eval(call.Arguments[0], env)
		if err != nil {
			return nil, true, err
		}
		off, err := ev.Eval(call.Arguments[1], env)
		if err != nil {
			return nil, true, err
		}
		val, err := ev.Eval(call.Arguments[2], env)
		if err != nil {
			return nil, true, err
		}
		hVal, ok1 := h.(IntValue)
		offVal, ok2 := off.(IntValue)
		valInt, ok3 := val.(IntValue)
		if !ok1 || !ok2 || !ok3 {
			return nil, true, fmt.Errorf("mem_write 参数必须为整型")
		}
		err = ev.MemManager.Write(hVal.Value, int(offVal.Value), valInt.Value)
		if err != nil {
			return nil, true, err
		}
		return VoidValue{}, true, nil

	case "mem_read", "read":
		if len(call.Arguments) < 2 {
			return nil, true, fmt.Errorf("mem_read 需要 (handle, offset)")
		}
		h, err := ev.Eval(call.Arguments[0], env)
		if err != nil {
			return nil, true, err
		}
		off, err := ev.Eval(call.Arguments[1], env)
		if err != nil {
			return nil, true, err
		}
		hVal, ok1 := h.(IntValue)
		offVal, ok2 := off.(IntValue)
		if !ok1 || !ok2 {
			return nil, true, fmt.Errorf("mem_read 参数必须为整型")
		}
		v, err := ev.MemManager.Read(hVal.Value, int(offVal.Value))
		if err != nil {
			return nil, true, err
		}
		return IntValue{Value: v}, true, nil

	case "mem_stats", "stats":
		st := ev.MemManager.Stats()
		out := fmt.Sprintf("【内存统计】: 累计分配 %d 字节, 当前活跃 %d 字节, 峰值 %d 字节, 分配 %d 次, 释放 %d 次\n",
			st.TotalAllocatedBytes, st.ActiveBytes, st.PeakBytes, st.AllocCount, st.FreeCount)
		_, _ = env.stdout.Write([]byte(out))
		return VoidValue{}, true, nil

	case "mem_check_leaks", "check_leaks":
		leaks := ev.MemManager.CheckLeaks()
		if len(leaks) > 0 {
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("⚠️ 检测到 %d 个未释放的内存泄漏块:\n", len(leaks)))
			for _, lk := range leaks {
				sb.WriteString(fmt.Sprintf("  • 句柄 0x%X 大小 %d 字长 (在 %s:%d 处分配)\n", lk.Handle, lk.Size, lk.CallerFn, lk.Line))
			}
			_, _ = env.stdout.Write([]byte(sb.String()))
		} else {
			_, _ = env.stdout.Write([]byte("✓ 内存泄漏检查: 零泄漏，所有分配已安全释放\n"))
		}
		return IntValue{Value: int64(len(leaks))}, true, nil

	// 后端核心内建：数组与切片操作
	case "range":
		var start, end int64 = 0, 0
		if len(call.Arguments) == 1 {
			v, err := ev.Eval(call.Arguments[0], env)
			if err != nil {
				return nil, true, err
			}
			iv, ok := v.(IntValue)
			if !ok {
				return nil, true, fmt.Errorf("range 参数必须为 int")
			}
			end = iv.Value
		} else if len(call.Arguments) >= 2 {
			v1, err := ev.Eval(call.Arguments[0], env)
			if err != nil {
				return nil, true, err
			}
			v2, err := ev.Eval(call.Arguments[1], env)
			if err != nil {
				return nil, true, err
			}
			iv1, ok1 := v1.(IntValue)
			iv2, ok2 := v2.(IntValue)
			if !ok1 || !ok2 {
				return nil, true, fmt.Errorf("range 参数必须为 int")
			}
			start = iv1.Value
			end = iv2.Value
		}
		count := int(end - start)
		if count < 0 {
			count = 0
		}
		elems := make([]Value, 0, count)
		for i := start; i < end; i++ {
			elems = append(elems, IntValue{Value: i})
		}
		return &ArrayValue{Elements: elems}, true, nil

	case "len":
		if len(call.Arguments) < 1 {
			return nil, true, fmt.Errorf("len 期望 1 个参数")
		}
		v, err := ev.Eval(call.Arguments[0], env)
		if err != nil {
			return nil, true, err
		}
		if arr, ok := v.(*ArrayValue); ok {
			return IntValue{Value: int64(len(arr.Elements))}, true, nil
		}
		if str, ok := v.(StringValue); ok {
			return IntValue{Value: int64(len([]rune(str.Value)))}, true, nil
		}
		return nil, true, fmt.Errorf("len 期望 array 或 string, 遇到 %s", v.Type())

	case "append", "push":
		if len(call.Arguments) < 2 {
			return nil, true, fmt.Errorf("%s 期望 2 个参数 (arr, val)", name)
		}
		target, err := ev.Eval(call.Arguments[0], env)
		if err != nil {
			return nil, true, err
		}
		arr, ok := target.(*ArrayValue)
		if !ok {
			return nil, true, fmt.Errorf("%s 第 1 个参数必须是 array", name)
		}
		val, err := ev.Eval(call.Arguments[1], env)
		if err != nil {
			return nil, true, err
		}
		arr.Elements = append(arr.Elements, val)
		return arr, true, nil

	case "pop":
		if len(call.Arguments) < 1 {
			return nil, true, fmt.Errorf("pop 期望 1 个参数 (arr)")
		}
		target, err := ev.Eval(call.Arguments[0], env)
		if err != nil {
			return nil, true, err
		}
		arr, ok := target.(*ArrayValue)
		if !ok {
			return nil, true, fmt.Errorf("pop 参数必须是 array")
		}
		if len(arr.Elements) == 0 {
			return nil, true, fmt.Errorf("pop 错误: 数组为空")
		}
		last := arr.Elements[len(arr.Elements)-1]
		arr.Elements = arr.Elements[:len(arr.Elements)-1]
		return last, true, nil

	// 后端核心内建：文件 IO (std/fs)
	case "fs_read_text":
		if len(call.Arguments) < 1 {
			return nil, true, fmt.Errorf("fs_read_text 需要文件路径")
		}
		pVal, err := ev.Eval(call.Arguments[0], env)
		if err != nil {
			return nil, true, err
		}
		b, err := os.ReadFile(pVal.String())
		if err != nil {
			return nil, true, fmt.Errorf("读取文件失败 %q: %v", pVal.String(), err)
		}
		return StringValue{Value: string(b)}, true, nil

	case "fs_write_text":
		if len(call.Arguments) < 2 {
			return nil, true, fmt.Errorf("fs_write_text 需要 (path, text)")
		}
		pVal, err := ev.Eval(call.Arguments[0], env)
		if err != nil {
			return nil, true, err
		}
		tVal, err := ev.Eval(call.Arguments[1], env)
		if err != nil {
			return nil, true, err
		}
		err = os.WriteFile(pVal.String(), []byte(tVal.String()), 0644)
		if err != nil {
			return BoolValue{Value: false}, true, fmt.Errorf("写入文件失败 %q: %v", pVal.String(), err)
		}
		return BoolValue{Value: true}, true, nil

	case "fs_exists":
		if len(call.Arguments) < 1 {
			return nil, true, fmt.Errorf("fs_exists 需要文件路径")
		}
		pVal, err := ev.Eval(call.Arguments[0], env)
		if err != nil {
			return nil, true, err
		}
		_, err = os.Stat(pVal.String())
		return BoolValue{Value: err == nil}, true, nil

	case "fs_remove":
		if len(call.Arguments) < 1 {
			return nil, true, fmt.Errorf("fs_remove 需要文件路径")
		}
		pVal, err := ev.Eval(call.Arguments[0], env)
		if err != nil {
			return nil, true, err
		}
		err = os.Remove(pVal.String())
		return BoolValue{Value: err == nil}, true, nil

	// 后端核心内建：字符串处理 (std/str)
	case "str_len":
		if len(call.Arguments) < 1 {
			return nil, true, fmt.Errorf("str_len 需要 1 个参数")
		}
		v, err := ev.Eval(call.Arguments[0], env)
		if err != nil {
			return nil, true, err
		}
		return IntValue{Value: int64(len([]rune(v.String())))}, true, nil

	case "str_contains":
		if len(call.Arguments) < 2 {
			return nil, true, fmt.Errorf("str_contains 需要 2 个参数")
		}
		s1, err := ev.Eval(call.Arguments[0], env)
		if err != nil {
			return nil, true, err
		}
		s2, err := ev.Eval(call.Arguments[1], env)
		if err != nil {
			return nil, true, err
		}
		return BoolValue{Value: strings.Contains(s1.String(), s2.String())}, true, nil

	case "str_split":
		if len(call.Arguments) < 2 {
			return nil, true, fmt.Errorf("str_split 需要 (s, sep)")
		}
		s1, err := ev.Eval(call.Arguments[0], env)
		if err != nil {
			return nil, true, err
		}
		s2, err := ev.Eval(call.Arguments[1], env)
		if err != nil {
			return nil, true, err
		}
		splits := strings.Split(s1.String(), s2.String())
		elems := make([]Value, 0, len(splits))
		for _, sp := range splits {
			elems = append(elems, StringValue{Value: sp})
		}
		return &ArrayValue{Elements: elems}, true, nil

	case "str_trim":
		if len(call.Arguments) < 1 {
			return nil, true, fmt.Errorf("str_trim 需要 1 个参数")
		}
		v, err := ev.Eval(call.Arguments[0], env)
		if err != nil {
			return nil, true, err
		}
		return StringValue{Value: strings.TrimSpace(v.String())}, true, nil

	case "str_upper":
		if len(call.Arguments) < 1 {
			return nil, true, fmt.Errorf("str_upper 需要 1 个参数")
		}
		v, err := ev.Eval(call.Arguments[0], env)
		if err != nil {
			return nil, true, err
		}
		return StringValue{Value: strings.ToUpper(v.String())}, true, nil

	case "str_lower":
		if len(call.Arguments) < 1 {
			return nil, true, fmt.Errorf("str_lower 需要 1 个参数")
		}
		v, err := ev.Eval(call.Arguments[0], env)
		if err != nil {
			return nil, true, err
		}
		return StringValue{Value: strings.ToLower(v.String())}, true, nil

	// 后端核心内建：随机数 (std/random)
	case "random_int":
		var minVal, maxVal int64 = 0, 100
		if len(call.Arguments) >= 2 {
			v1, _ := ev.Eval(call.Arguments[0], env)
			v2, _ := ev.Eval(call.Arguments[1], env)
			if iv1, ok := v1.(IntValue); ok {
				minVal = iv1.Value
			}
			if iv2, ok := v2.(IntValue); ok {
				maxVal = iv2.Value
			}
		}
		if maxVal <= minVal {
			return IntValue{Value: minVal}, true, nil
		}
		n := rand.Int63n(maxVal-minVal) + minVal
		return IntValue{Value: n}, true, nil

	case "random_float":
		return FloatValue{Value: rand.Float64()}, true, nil
	}

	if strings.HasPrefix(name, "cpp_") {
		args := make([]Value, 0, len(call.Arguments))
		for _, arg := range call.Arguments {
			val, err := ev.Eval(arg, env)
			if err != nil {
				return nil, true, err
			}
			args = append(args, val)
		}
		res, handled, err := ev.evalCppBuiltin(name, args)
		return res, handled, err
	}

	return nil, false, nil
}

func (ev *Evaluator) evalCppBuiltin(name string, args []Value) (Value, bool, error) {
	switch name {
	case "cpp_fast_sqrt":
		if len(args) < 1 {
			return nil, true, fmt.Errorf("cpp_fast_sqrt 需要 1 个参数")
		}
		var x float64
		if f, ok := args[0].(FloatValue); ok {
			x = f.Value
		} else if i, ok := args[0].(IntValue); ok {
			x = float64(i.Value)
		} else {
			return nil, true, fmt.Errorf("cpp_fast_sqrt 参数必须为数值类型")
		}
		if x < 0 {
			return FloatValue{Value: 0.0}, true, nil
		}
		return FloatValue{Value: math.Sqrt(x)}, true, nil

	case "cpp_pow":
		if len(args) < 2 {
			return nil, true, fmt.Errorf("cpp_pow 需要 2 个参数")
		}
		var base, exp float64
		if f, ok := args[0].(FloatValue); ok {
			base = f.Value
		} else if i, ok := args[0].(IntValue); ok {
			base = float64(i.Value)
		}
		if f, ok := args[1].(FloatValue); ok {
			exp = f.Value
		} else if i, ok := args[1].(IntValue); ok {
			exp = float64(i.Value)
		}
		return FloatValue{Value: math.Pow(base, exp)}, true, nil

	case "cpp_abs":
		if len(args) < 1 {
			return nil, true, fmt.Errorf("cpp_abs 需要 1 个参数")
		}
		if i, ok := args[0].(IntValue); ok {
			if i.Value < 0 {
				return IntValue{Value: -i.Value}, true, nil
			}
			return i, true, nil
		}
		if f, ok := args[0].(FloatValue); ok {
			return FloatValue{Value: math.Abs(f.Value)}, true, nil
		}
		return nil, true, fmt.Errorf("cpp_abs 参数必须为数值类型")

	case "cpp_gcd":
		if len(args) < 2 {
			return nil, true, fmt.Errorf("cpp_gcd 需要 2 个参数")
		}
		a, ok1 := args[0].(IntValue)
		b, ok2 := args[1].(IntValue)
		if !ok1 || !ok2 {
			return nil, true, fmt.Errorf("cpp_gcd 参数必须为整型")
		}
		x, y := a.Value, b.Value
		for y != 0 {
			t := y
			y = x % y
			x = t
		}
		if x < 0 {
			x = -x
		}
		return IntValue{Value: x}, true, nil

	case "cpp_fibonacci":
		if len(args) < 1 {
			return nil, true, fmt.Errorf("cpp_fibonacci 需要 1 个参数")
		}
		nVal, ok := args[0].(IntValue)
		if !ok {
			return nil, true, fmt.Errorf("cpp_fibonacci 参数必须为整型")
		}
		n := nVal.Value
		if n <= 0 {
			return IntValue{Value: 0}, true, nil
		}
		if n == 1 {
			return IntValue{Value: 1}, true, nil
		}
		var a, b int64 = 0, 1
		for i := int64(2); i <= n; i++ {
			c := a + b
			a = b
			b = c
		}
		return IntValue{Value: b}, true, nil

	case "cpp_is_prime":
		if len(args) < 1 {
			return nil, true, fmt.Errorf("cpp_is_prime 需要 1 个参数")
		}
		nVal, ok := args[0].(IntValue)
		if !ok {
			return nil, true, fmt.Errorf("cpp_is_prime 参数必须为整型")
		}
		n := nVal.Value
		if n <= 1 {
			return BoolValue{Value: false}, true, nil
		}
		if n <= 3 {
			return BoolValue{Value: true}, true, nil
		}
		if n%2 == 0 || n%3 == 0 {
			return BoolValue{Value: false}, true, nil
		}
		for i := int64(5); i*i <= n; i += 6 {
			if n%i == 0 || n%(i+2) == 0 {
				return BoolValue{Value: false}, true, nil
			}
		}
		return BoolValue{Value: true}, true, nil
	}
	return nil, false, nil
}
