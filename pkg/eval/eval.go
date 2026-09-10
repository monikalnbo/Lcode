package eval

import (
	"bytes"
	"fmt"
	"io"
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
	stdout *bytes.Buffer
}

func New() *Evaluator {
	return &Evaluator{
		stdout: new(bytes.Buffer),
	}
}

// RunProgram 执行整个 AST 程序并返回控制台输出
func (ev *Evaluator) RunProgram(program *ast.Program) (string, error) {
	ev.stdout.Reset()
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

	// 内置函数 println / print
	if ident.Value == "println" || ident.Value == "print" {
		args := make([]string, 0, len(call.Arguments))
		for _, arg := range call.Arguments {
			val, err := ev.Eval(arg, env)
			if err != nil {
				return nil, err
			}
			args = append(args, val.String())
		}
		outText := strings.Join(args, " ")
		if ident.Value == "println" {
			outText += "\n"
		}
		_, _ = env.stdout.Write([]byte(outText))
		return VoidValue{}, nil
	}

	// 用户自定义函数
	fnVal, ok := env.Get(ident.Value)
	if !ok {
		return nil, fmt.Errorf("未定义的函数 %q", ident.Value)
	}
	fn, ok := fnVal.(FunctionValue)
	if !ok {
		return nil, fmt.Errorf("%q 不是函数", ident.Value)
	}

	fnEnv := NewEnvironment(fn.Env, env.stdout)
	for i, param := range fn.Decl.Params {
		if i >= len(call.Arguments) {
			break
		}
		argVal, err := ev.Eval(call.Arguments[i], env)
		if err != nil {
			return nil, err
		}
		fnEnv.Set(param.Name.Value, argVal)
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
