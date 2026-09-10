package eval

import (
	"bytes"
	"fmt"
	"io"
	"math"
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
	parent *Environment
	store  map[string]Value
	stdout io.Writer
}

func NewEnvironment(parent *Environment, stdout io.Writer) *Environment {
	if stdout == nil && parent != nil {
		stdout = parent.stdout
	}
	return &Environment{
		parent: parent,
		store:  make(map[string]Value),
		stdout: stdout,
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
		ident, ok := n.Left.(*ast.Identifier)
		if !ok {
			return nil, fmt.Errorf("赋值目标必须为标识符")
		}
		rightVal, err := ev.Eval(n.Right, env)
		if err != nil {
			return nil, err
		}

		if n.Operator != "=" {
			currVal, exists := env.Get(ident.Value)
			if !exists {
				return nil, fmt.Errorf("未定义的变量 %s", ident.Value)
			}
			rawOp := strings.TrimSuffix(n.Operator, "=")
			newVal, err := ev.evalBinary(rawOp, currVal, rightVal)
			if err != nil {
				return nil, err
			}
			rightVal = newVal
		}

		if !env.Assign(ident.Value, rightVal) {
			env.Set(ident.Value, rightVal)
		}
		return rightVal, nil

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
			if ret, ok := res.(ReturnValue); ok {
				return ret, nil
			}
		}
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
	subEnv := NewEnvironment(env, env.stdout)
	for _, stmt := range block.Statements {
		res, err := ev.Eval(stmt, subEnv)
		if err != nil {
			return nil, err
		}
		if ret, ok := res.(ReturnValue); ok {
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

	case "mem_alloc":
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
		return IntValue{Value: h}, true, nil

	case "mem_free":
		if len(call.Arguments) < 1 {
			return nil, true, fmt.Errorf("mem_free 需要指定内存句柄")
		}
		hVal, err := ev.Eval(call.Arguments[0], env)
		if err != nil {
			return nil, true, err
		}
		iVal, ok := hVal.(IntValue)
		if !ok {
			return nil, true, fmt.Errorf("mem_free 句柄必须为整型")
		}
		err = ev.MemManager.Free(iVal.Value)
		if err != nil {
			return nil, true, err
		}
		return IntValue{Value: 0}, true, nil

	case "mem_write":
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

	case "mem_read":
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

	case "mem_stats":
		st := ev.MemManager.Stats()
		out := fmt.Sprintf("【内存统计】: 累计分配 %d 字节, 当前活跃 %d 字节, 峰值 %d 字节, 分配 %d 次, 释放 %d 次\n",
			st.TotalAllocatedBytes, st.ActiveBytes, st.PeakBytes, st.AllocCount, st.FreeCount)
		_, _ = env.stdout.Write([]byte(out))
		return VoidValue{}, true, nil

	case "mem_check_leaks":
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
