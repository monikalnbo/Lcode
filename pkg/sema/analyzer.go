package sema

import (
	"fmt"

	"lcode/pkg/ast"
	"lcode/pkg/token"
)

// SemanticAnalyzer 语义与类型分析器
type SemanticAnalyzer struct {
	globalScope   *Scope
	currentScope  *Scope
	currentFunc   *ast.FuncDecl
	BorrowChecker *BorrowChecker
	Errors        []string
}

// NewAnalyzer 创建语义分析器并初始化内置符号
func NewAnalyzer() *SemanticAnalyzer {
	global := NewScope(nil)

	defBuiltin := func(name string, params []Type, ret Type) {
		_ = global.Define(&Symbol{
			Name: name,
			Kind: SymFunc,
			Type: &FuncType{
				ParamTypes: params,
				ReturnType: ret,
			},
			Pos: token.Position{Filename: "<builtin>"},
		})
	}

	// 基础 IO
	defBuiltin("print", []Type{TypeString}, TypeVoid)
	defBuiltin("println", []Type{TypeString}, TypeVoid)

	// 错误与断言
	defBuiltin("panic", []Type{TypeString}, TypeVoid)
	defBuiltin("assert", []Type{TypeBool, TypeString}, TypeVoid)

	// 栈技术
	defBuiltin("stack_dump", []Type{}, TypeVoid)
	defBuiltin("stack_depth", []Type{}, TypeInt)

	// 时间管理
	defBuiltin("time_now_ms", []Type{}, TypeInt)
	defBuiltin("time_now_secs", []Type{}, TypeInt)
	defBuiltin("time_now_micros", []Type{}, TypeInt)
	defBuiltin("time_sleep_ms", []Type{TypeInt}, TypeVoid)
	defBuiltin("time_elapsed_ms", []Type{TypeInt}, TypeInt)
	defBuiltin("time_format_now", []Type{}, TypeString)

	// 内存管理与 Rust 风格所有权/RAII
	defBuiltin("mem_alloc", []Type{TypeInt}, TypeInt)
	defBuiltin("alloc", []Type{TypeInt}, TypeInt)
	defBuiltin("mem_free", []Type{TypeInt}, TypeInt)
	defBuiltin("free", []Type{TypeInt}, TypeInt)
	defBuiltin("drop", []Type{TypeInt}, TypeInt)
	defBuiltin("mem_write", []Type{TypeInt, TypeInt, TypeInt}, TypeVoid)
	defBuiltin("write", []Type{TypeInt, TypeInt, TypeInt}, TypeVoid)
	defBuiltin("mem_read", []Type{TypeInt, TypeInt}, TypeInt)
	defBuiltin("read", []Type{TypeInt, TypeInt}, TypeInt)
	defBuiltin("mem_stats", []Type{}, TypeVoid)
	defBuiltin("stats", []Type{}, TypeVoid)
	defBuiltin("mem_check_leaks", []Type{}, TypeInt)
	defBuiltin("check_leaks", []Type{}, TypeInt)

	return &SemanticAnalyzer{
		globalScope:   global,
		currentScope:  global,
		BorrowChecker: NewBorrowChecker(),
		Errors:        make([]string, 0),
	}
}

func (sa *SemanticAnalyzer) addError(pos token.Position, msg string) {
	sa.Errors = append(sa.Errors, fmt.Sprintf("%s: 语义错误: %s", pos, msg))
}

func (sa *SemanticAnalyzer) enterScope() {
	sa.currentScope = NewScope(sa.currentScope)
}

func (sa *SemanticAnalyzer) exitScope() {
	if sa.currentScope.parent != nil {
		sa.currentScope = sa.currentScope.parent
	}
}

// Analyze 执行完整程序的语义分析
func (sa *SemanticAnalyzer) Analyze(program *ast.Program) {
	if program == nil {
		return
	}

	// 第一阶段：收集所有函数声明签名（支持互相递归与前向调用）
	for _, decl := range program.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			sa.declareFunc(fn)
		}
	}

	// 第二阶段：分析顶层语句
	for _, stmt := range program.Stmts {
		sa.analyzeStmt(stmt)
	}

	// 第三阶段：分析各个函数体
	for _, decl := range program.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			sa.analyzeFuncBody(fn)
		}
	}
}

// declareFunc 登记函数符号
func (sa *SemanticAnalyzer) declareFunc(fn *ast.FuncDecl) {
	var retType Type = TypeVoid
	if fn.ReturnType != nil {
		resolved := LookupBasicType(fn.ReturnType.Name)
		if resolved == nil {
			sa.addError(fn.ReturnType.Pos(), fmt.Sprintf("未知返回类型 %q", fn.ReturnType.Name))
		} else {
			retType = resolved
		}
	}

	paramTypes := make([]Type, 0, len(fn.Params))
	for _, p := range fn.Params {
		var pt Type = TypeUnknown
		if p.Type != nil {
			resolved := LookupBasicType(p.Type.Name)
			if resolved == nil {
				sa.addError(p.Type.Pos(), fmt.Sprintf("形参 %q 类型未知: %q", p.Name.Value, p.Type.Name))
			} else {
				pt = resolved
			}
		}
		paramTypes = append(paramTypes, pt)
	}

	sym := &Symbol{
		Name: fn.Name.Value,
		Kind: SymFunc,
		Type: &FuncType{
			ParamTypes: paramTypes,
			ReturnType: retType,
		},
		Pos: fn.Pos(),
	}

	if err := sa.globalScope.Define(sym); err != nil {
		sa.addError(fn.Pos(), err.Error())
	}
}

// analyzeFuncBody 分析函数体实现
func (sa *SemanticAnalyzer) analyzeFuncBody(fn *ast.FuncDecl) {
	oldFunc := sa.currentFunc
	sa.currentFunc = fn
	sa.enterScope()

	// 注册形参符号
	for _, p := range fn.Params {
		var pt Type = TypeUnknown
		if p.Type != nil {
			if t := LookupBasicType(p.Type.Name); t != nil {
				pt = t
			}
		}
		sym := &Symbol{
			Name: p.Name.Value,
			Kind: SymParam,
			Type: pt,
			Pos:  p.Pos(),
		}
		if err := sa.currentScope.Define(sym); err != nil {
			sa.addError(p.Pos(), err.Error())
		}
	}

	// 分析函数体内部语句
	if fn.Body != nil {
		for _, stmt := range fn.Body.Statements {
			sa.analyzeStmt(stmt)
		}
	}

	sa.exitScope()
	sa.currentFunc = oldFunc
}

func (sa *SemanticAnalyzer) analyzeStmt(stmt ast.Stmt) {
	if stmt == nil {
		return
	}

	switch s := stmt.(type) {
	case *ast.VarDeclStmt:
		sa.analyzeVarDecl(s)

	case *ast.AssignStmt:
		sa.analyzeAssignStmt(s)

	case *ast.BlockStmt:
		sa.enterScope()
		for _, sub := range s.Statements {
			sa.analyzeStmt(sub)
		}
		sa.exitScope()

	case *ast.ExprStmt:
		if s.Expression != nil {
			sa.inferExprType(s.Expression)
		}

	case *ast.IfStmt:
		condType := sa.inferExprType(s.Condition)
		if condType != nil && !condType.Equals(TypeBool) {
			sa.addError(s.Condition.Pos(), fmt.Sprintf("if 条件表达式类型必须为 bool, 实际为 %s", condType.Name()))
		}
		sa.analyzeStmt(s.Consequence)
		if s.Alternative != nil {
			sa.analyzeStmt(s.Alternative)
		}

	case *ast.WhileStmt:
		condType := sa.inferExprType(s.Condition)
		if condType != nil && !condType.Equals(TypeBool) {
			sa.addError(s.Condition.Pos(), fmt.Sprintf("while 条件表达式类型必须为 bool, 实际为 %s", condType.Name()))
		}
		sa.analyzeStmt(s.Body)

	case *ast.ReturnStmt:
		sa.analyzeReturnStmt(s)
	}
}

func (sa *SemanticAnalyzer) analyzeVarDecl(v *ast.VarDeclStmt) {
	var declaredType Type
	if v.Type != nil {
		declaredType = LookupBasicType(v.Type.Name)
		if declaredType == nil {
			sa.addError(v.Type.Pos(), fmt.Sprintf("未知类型标注 %q", v.Type.Name))
		}
	}

	var valType Type
	if v.Value != nil {
		valType = sa.inferExprType(v.Value)
	}

	var finalType Type = TypeUnknown
	if declaredType != nil && valType != nil {
		if !declaredType.Equals(valType) {
			sa.addError(v.Value.Pos(), fmt.Sprintf("类型不匹配: 变量 %q 声明类型为 %s, 但赋值类型为 %s",
				v.Name.Value, declaredType.Name(), valType.Name()))
		}
		finalType = declaredType
	} else if declaredType != nil {
		finalType = declaredType
	} else if valType != nil {
		finalType = valType
	}

	kind := SymConst
	if v.IsMut {
		kind = SymMutVar
	}

	sym := &Symbol{
		Name:  v.Name.Value,
		Kind:  kind,
		IsMut: v.IsMut,
		Type:  finalType,
		Pos:   v.Pos(),
	}

	if err := sa.currentScope.Define(sym); err != nil {
		sa.addError(v.Pos(), err.Error())
	}

	// 借用检查器：识别所有权托管资源与所有权移动 (Ownership & Move Tracking)
	if call, ok := v.Value.(*ast.CallExpr); ok {
		if fnIdent, ok := call.Function.(*ast.Identifier); ok {
			if fnIdent.Value == "alloc" || fnIdent.Value == "mem_alloc" {
				sa.BorrowChecker.RegisterResource(v.Name.Value, v.Pos())
			}
		}
	} else if srcIdent, ok := v.Value.(*ast.Identifier); ok {
		// 变量赋值转移: let b = a; 发生所有权转移 (Move)
		_ = sa.BorrowChecker.Move(srcIdent.Value, v.Name.Value, v.Pos())
	}
}

func (sa *SemanticAnalyzer) analyzeAssignStmt(as *ast.AssignStmt) {
	ident, ok := as.Left.(*ast.Identifier)
	if !ok {
		sa.addError(as.Left.Pos(), "赋值操作的左值必须是标识符")
		return
	}

	sym, found := sa.currentScope.Resolve(ident.Value)
	if !found {
		sa.addError(ident.Pos(), fmt.Sprintf("未定义的变量 %q", ident.Value))
		return
	}

	if !sym.IsMut && sym.Kind != SymVar {
		sa.addError(ident.Pos(), fmt.Sprintf("无法给不可变变量 %q 重新赋值 (提示: 可将变量声明为 'mut %s')", ident.Value, ident.Value))
		return
	}

	rightType := sa.inferExprType(as.Right)
	if rightType != nil && sym.Type != nil && !sym.Type.Equals(rightType) {
		sa.addError(as.Right.Pos(), fmt.Sprintf("赋值类型不匹配: 变量 %q 类型为 %s, 赋值表达式类型为 %s",
			ident.Value, sym.Type.Name(), rightType.Name()))
	}

	// 借用检查器：赋值转移所有权 (Move) 或接收堆分配
	if srcIdent, ok := as.Right.(*ast.Identifier); ok {
		if sa.BorrowChecker.IsResource(srcIdent.Value) {
			_ = sa.BorrowChecker.Move(srcIdent.Value, ident.Value, as.Pos())
		}
	} else if call, ok := as.Right.(*ast.CallExpr); ok {
		if fnIdent, ok := call.Function.(*ast.Identifier); ok {
			if fnIdent.Value == "alloc" || fnIdent.Value == "mem_alloc" {
				sa.BorrowChecker.RegisterResource(ident.Value, as.Pos())
			}
		}
	}
}

// GlobalScope 返回全局作用域
func (sa *SemanticAnalyzer) GlobalScope() *Scope {
	return sa.globalScope
}

func (sa *SemanticAnalyzer) analyzeReturnStmt(rs *ast.ReturnStmt) {
	if sa.currentFunc == nil {
		sa.addError(rs.Pos(), "return 语句只能出现在函数内部")
		return
	}

	var expectedRet Type = TypeVoid
	if sa.currentFunc.ReturnType != nil {
		if t := LookupBasicType(sa.currentFunc.ReturnType.Name); t != nil {
			expectedRet = t
		}
	}

	if rs.ReturnValue == nil {
		if !expectedRet.Equals(TypeVoid) {
			sa.addError(rs.Pos(), fmt.Sprintf("函数 %q 期望返回类型 %s, 但 return 为空",
				sa.currentFunc.Name.Value, expectedRet.Name()))
		}
		return
	}

	actualRet := sa.inferExprType(rs.ReturnValue)
	if actualRet != nil && !actualRet.Equals(expectedRet) {
		sa.addError(rs.ReturnValue.Pos(), fmt.Sprintf("函数 %q 返回类型不匹配: 期望 %s, 实际返回 %s",
			sa.currentFunc.Name.Value, expectedRet.Name(), actualRet.Name()))
	}
}

// inferExprType 推导表达式类型
func (sa *SemanticAnalyzer) inferExprType(expr ast.Expr) Type {
	if expr == nil {
		return TypeUnknown
	}

	switch e := expr.(type) {
	case *ast.IntegerLiteral:
		return TypeInt
	case *ast.FloatLiteral:
		return TypeFloat
	case *ast.StringLiteral:
		return TypeString
	case *ast.BooleanLiteral:
		return TypeBool
	case *ast.Identifier:
		sym, found := sa.currentScope.Resolve(e.Value)
		if !found {
			sa.addError(e.Pos(), fmt.Sprintf("未定义的标识符 %q", e.Value))
			return TypeUnknown
		}
		// 借用检查器：审查变量是否可合法访问（未被移动）
		if err := sa.BorrowChecker.AccessRead(e.Value, e.Pos()); err != nil {
			sa.addError(e.Pos(), err.Error())
		}
		return sym.Type

	case *ast.GroupedExpr:
		return sa.inferExprType(e.Expression)

	case *ast.UnaryExpr:
		rightType := sa.inferExprType(e.Right)
		if e.Operator == "!" {
			if rightType != nil && !rightType.Equals(TypeBool) {
				sa.addError(e.Pos(), fmt.Sprintf("逻辑非运算符 '!' 操作数必须为 bool, 实际为 %s", rightType.Name()))
			}
			return TypeBool
		}
		if e.Operator == "-" {
			if rightType != nil && !rightType.Equals(TypeInt) && !rightType.Equals(TypeFloat) {
				sa.addError(e.Pos(), fmt.Sprintf("负号运算符 '-' 操作数必须为数值类型, 实际为 %s", rightType.Name()))
			}
			return rightType
		}
		if e.Operator == "&" {
			// Rust 风格借用操作符 (&x)
			if ident, ok := e.Right.(*ast.Identifier); ok {
				if err := sa.BorrowChecker.Borrow(ident.Value, e.Pos()); err != nil {
					sa.addError(e.Pos(), err.Error())
				}
			}
			return rightType
		}
		if e.Operator == "*" {
			// 解引用操作符 (*x)
			return rightType
		}
		return TypeUnknown

	case *ast.BinaryExpr:
		leftType := sa.inferExprType(e.Left)
		rightType := sa.inferExprType(e.Right)

		switch e.Operator {
		case "+":
			if leftType.Equals(TypeString) || rightType.Equals(TypeString) {
				return TypeString
			}
			if leftType.Equals(TypeInt) && rightType.Equals(TypeInt) {
				return TypeInt
			}
			if leftType.Equals(TypeFloat) && rightType.Equals(TypeFloat) {
				return TypeFloat
			}
			sa.addError(e.Pos(), fmt.Sprintf("运算符 '+' 操作数类型不兼容: %s 与 %s", leftType.Name(), rightType.Name()))
			return TypeUnknown

		case "-", "*", "/", "%":
			if leftType.Equals(TypeInt) && rightType.Equals(TypeInt) {
				return TypeInt
			}
			if leftType.Equals(TypeFloat) && rightType.Equals(TypeFloat) {
				return TypeFloat
			}
			sa.addError(e.Pos(), fmt.Sprintf("算术运算符 %q 要求数值类型且保持一致: 遇到 %s 与 %s",
				e.Operator, leftType.Name(), rightType.Name()))
			return TypeUnknown

		case "==", "!=":
			if !leftType.Equals(rightType) {
				sa.addError(e.Pos(), fmt.Sprintf("比较运算符 %q 两侧类型不匹配: %s 与 %s",
					e.Operator, leftType.Name(), rightType.Name()))
			}
			return TypeBool

		case "<", "<=", ">", ">=":
			if (!leftType.Equals(TypeInt) && !leftType.Equals(TypeFloat)) || !leftType.Equals(rightType) {
				sa.addError(e.Pos(), fmt.Sprintf("关系运算符 %q 必须作用于相同的数值类型: 遇到 %s 与 %s",
					e.Operator, leftType.Name(), rightType.Name()))
			}
			return TypeBool

		case "&&", "||":
			if !leftType.Equals(TypeBool) || !rightType.Equals(TypeBool) {
				sa.addError(e.Pos(), fmt.Sprintf("逻辑运算符 %q 操作数必须为 bool: 遇到 %s 与 %s",
					e.Operator, leftType.Name(), rightType.Name()))
			}
			return TypeBool
		}

	case *ast.CallExpr:
		funcIdent, ok := e.Function.(*ast.Identifier)
		if !ok {
			sa.addError(e.Pos(), "暂时仅支持对标识符函数的直接调用")
			return TypeUnknown
		}

		sym, found := sa.currentScope.Resolve(funcIdent.Value)
		if !found {
			// 智能缩写与最短唯一前缀自动解析 (如 prin -> print, p -> println)
			if canonical, ok := sa.ResolveFunctionAbbrev(funcIdent.Value); ok {
				funcIdent.Value = canonical
				sym, found = sa.currentScope.Resolve(canonical)
			}
		}
		if !found {
			sa.addError(funcIdent.Pos(), fmt.Sprintf("未定义的函数 %q", funcIdent.Value))
			return TypeUnknown
		}

		// 内置变长打印函数 print / println 支持任意类型与参数数量
		if funcIdent.Value == "print" || funcIdent.Value == "println" {
			for _, arg := range e.Arguments {
				sa.inferExprType(arg)
			}
			return TypeVoid
		}

		// 内置断言与恐慌函数
		if funcIdent.Value == "assert" {
			if len(e.Arguments) < 1 {
				sa.addError(e.Pos(), "assert 至少需要 1 个布尔参数")
			} else {
				t0 := sa.inferExprType(e.Arguments[0])
				if t0 != nil && !t0.Equals(TypeBool) {
					sa.addError(e.Arguments[0].Pos(), fmt.Sprintf("assert 第 1 个参数必须为 bool, 传入 %s", t0.Name()))
				}
				if len(e.Arguments) >= 2 {
					_ = sa.inferExprType(e.Arguments[1])
				}
			}
			return TypeVoid
		}

		if funcIdent.Value == "panic" {
			for _, a := range e.Arguments {
				_ = sa.inferExprType(a)
			}
			return TypeVoid
		}

		// 内存释放与 Rust 显式 drop 消费所有权
		if funcIdent.Value == "drop" || funcIdent.Value == "free" || funcIdent.Value == "mem_free" {
			if len(e.Arguments) >= 1 {
				_ = sa.inferExprType(e.Arguments[0])
				if ident, ok := e.Arguments[0].(*ast.Identifier); ok {
					if err := sa.BorrowChecker.Drop(ident.Value, ident.Pos()); err != nil {
						sa.addError(ident.Pos(), err.Error())
					}
				}
			}
			return TypeInt
		}

		fnType, ok := sym.Type.(*FuncType)
		if !ok {
			sa.addError(funcIdent.Pos(), fmt.Sprintf("%q 不是一个可调用的函数", funcIdent.Value))
			return TypeUnknown
		}

		if len(e.Arguments) != len(fnType.ParamTypes) {
			sa.addError(e.Pos(), fmt.Sprintf("函数 %q 调用参数数量不匹配: 期望 %d, 实际传入 %d",
				funcIdent.Value, len(fnType.ParamTypes), len(e.Arguments)))
			return fnType.ReturnType
		}

		isReadOrWrite := funcIdent.Value == "read" || funcIdent.Value == "mem_read" || funcIdent.Value == "write" || funcIdent.Value == "mem_write"
		for i, arg := range e.Arguments {
			argType := sa.inferExprType(arg)
			expectedType := fnType.ParamTypes[i]
			if argType != nil && expectedType != nil && !argType.Equals(expectedType) {
				sa.addError(arg.Pos(), fmt.Sprintf("函数 %q 第 %d 个实参类型不匹配: 期望 %s, 传入 %s",
					funcIdent.Value, i+1, expectedType.Name(), argType.Name()))
			}

			// Rust 风格按值传递移动所有权 (Move on by-value argument pass)
			if !isReadOrWrite {
				if ident, ok := arg.(*ast.Identifier); ok {
					if sa.BorrowChecker.IsResource(ident.Value) {
						targetParam := fmt.Sprintf("%s(形参_%d)", funcIdent.Value, i+1)
						_ = sa.BorrowChecker.Move(ident.Value, targetParam, arg.Pos())
					}
				}
			}
		}

		return fnType.ReturnType
	}

	return TypeUnknown
}
