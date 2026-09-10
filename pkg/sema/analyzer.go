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

	// 后端核心内建：数组与切片
	defBuiltin("range", []Type{TypeInt, TypeInt}, TypeArray)
	defBuiltin("len", []Type{TypeAny}, TypeInt)
	defBuiltin("append", []Type{TypeAny, TypeAny}, TypeArray)
	defBuiltin("push", []Type{TypeAny, TypeAny}, TypeArray)
	defBuiltin("pop", []Type{TypeAny}, TypeAny)

	// 后端核心内建：文件 IO (std/fs)
	defBuiltin("fs_read_text", []Type{TypeString}, TypeString)
	defBuiltin("fs_write_text", []Type{TypeString, TypeString}, TypeBool)
	defBuiltin("fs_exists", []Type{TypeString}, TypeBool)
	defBuiltin("fs_remove", []Type{TypeString}, TypeBool)

	// 后端核心内建：字符串处理 (std/str)
	defBuiltin("str_len", []Type{TypeString}, TypeInt)
	defBuiltin("str_contains", []Type{TypeString, TypeString}, TypeBool)
	defBuiltin("str_split", []Type{TypeString, TypeString}, TypeArray)
	defBuiltin("str_trim", []Type{TypeString}, TypeString)
	defBuiltin("str_upper", []Type{TypeString}, TypeString)
	defBuiltin("str_lower", []Type{TypeString}, TypeString)

	// 后端核心内建：随机数 (std/random)
	defBuiltin("random_int", []Type{TypeInt, TypeInt}, TypeInt)
	defBuiltin("random_float", []Type{}, TypeFloat)

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

// resolveType 根据类型名解析基本类型或用户自定义结构体类型
func (sa *SemanticAnalyzer) resolveType(name string) Type {
	if t := LookupBasicType(name); t != nil {
		return t
	}
	if sym, ok := sa.currentScope.Resolve(name); ok {
		if st, ok := sym.Type.(*StructType); ok {
			return st
		}
	}
	return nil
}

// declareStruct 登记结构体类型定义
func (sa *SemanticAnalyzer) declareStruct(st *ast.StructDecl) {
	fields := make(map[string]Type)
	for _, f := range st.Fields {
		var ft Type = TypeAny
		if f.Type != nil {
			if t := sa.resolveType(f.Type.Name); t != nil {
				ft = t
			} else {
				sa.addError(f.Type.Pos(), fmt.Sprintf("结构体 %s 的字段 %s 未知类型: %s", st.Name.Value, f.Name.Value, f.Type.Name))
			}
		}
		fields[f.Name.Value] = ft
	}
	structType := &StructType{
		StructName: st.Name.Value,
		Fields:     fields,
	}
	sym := &Symbol{
		Name: st.Name.Value,
		Kind: SymStruct,
		Type: structType,
		Pos:  st.Pos(),
	}
	if err := sa.globalScope.Define(sym); err != nil {
		sa.addError(st.Pos(), err.Error())
	}
}

// Analyze 执行完整程序的语义分析
func (sa *SemanticAnalyzer) Analyze(program *ast.Program) {
	if program == nil {
		return
	}

	// 第一阶段：收集所有结构体声明与函数声明签名（支持互相递归与前向调用）
	for _, decl := range program.Decls {
		if st, ok := decl.(*ast.StructDecl); ok {
			sa.declareStruct(st)
		}
	}
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
		resolved := sa.resolveType(fn.ReturnType.Name)
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
			resolved := sa.resolveType(p.Type.Name)
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
			if t := sa.resolveType(p.Type.Name); t != nil {
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

	case *ast.ForInStmt:
		iterType := sa.inferExprType(s.Iterable)
		var elemType Type = TypeAny
		if arrType, ok := iterType.(*ArrayType); ok {
			if arrType.ElementType != nil {
				elemType = arrType.ElementType
			}
		} else if iterType != nil && !iterType.Equals(TypeAny) {
			sa.addError(s.Iterable.Pos(), fmt.Sprintf("for ... in 只能遍历 array, 遇到类型 %s", iterType.Name()))
		}
		sa.enterScope()
		_ = sa.currentScope.Define(&Symbol{
			Name:  s.VarName.Value,
			Kind:  SymVar,
			Type:  elemType,
			Pos:   s.VarName.Pos(),
			IsMut: true,
		})
		if s.Body != nil {
			for _, sub := range s.Body.Statements {
				sa.analyzeStmt(sub)
			}
		}
		sa.exitScope()

	case *ast.BreakStmt:
		// 允许在循环内跳出

	case *ast.ContinueStmt:
		// 允许在循环内继续

	case *ast.TryCatchStmt:
		if s.TryBlock != nil {
			sa.enterScope()
			for _, sub := range s.TryBlock.Statements {
				sa.analyzeStmt(sub)
			}
			sa.exitScope()
		}
		if s.CatchBlock != nil {
			sa.enterScope()
			if s.ErrVar != nil {
				_ = sa.currentScope.Define(&Symbol{
					Name: s.ErrVar.Value,
					Kind: SymVar,
					Type: TypeString,
					Pos:  s.ErrVar.Pos(),
				})
			}
			for _, sub := range s.CatchBlock.Statements {
				sa.analyzeStmt(sub)
			}
			sa.exitScope()
		}

	case *ast.StructDecl:
		sa.declareStruct(s)

	case *ast.ReturnStmt:
		sa.analyzeReturnStmt(s)
	}
}

func (sa *SemanticAnalyzer) analyzeVarDecl(v *ast.VarDeclStmt) {
	var declaredType Type
	if v.Type != nil {
		declaredType = sa.resolveType(v.Type.Name)
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
		if !declaredType.Equals(valType) && !declaredType.Equals(TypeAny) {
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
	switch lhs := as.Left.(type) {
	case *ast.Identifier:
		sym, found := sa.currentScope.Resolve(lhs.Value)
		if !found {
			sa.addError(lhs.Pos(), fmt.Sprintf("未定义的变量 %q", lhs.Value))
			return
		}

		if !sym.IsMut && sym.Kind != SymVar {
			sa.addError(lhs.Pos(), fmt.Sprintf("无法给不可变变量 %q 重新赋值 (提示: 可将变量声明为 'mut %s')", lhs.Value, lhs.Value))
			return
		}

		rightType := sa.inferExprType(as.Right)
		if rightType != nil && sym.Type != nil && !sym.Type.Equals(rightType) && !sym.Type.Equals(TypeAny) {
			sa.addError(as.Right.Pos(), fmt.Sprintf("赋值类型不匹配: 变量 %q 类型为 %s, 赋值表达式类型为 %s",
				lhs.Value, sym.Type.Name(), rightType.Name()))
		}

		// 借用检查器：赋值转移所有权 (Move) 或接收堆分配
		if srcIdent, ok := as.Right.(*ast.Identifier); ok {
			if sa.BorrowChecker.IsResource(srcIdent.Value) {
				_ = sa.BorrowChecker.Move(srcIdent.Value, lhs.Value, as.Pos())
			}
		} else if call, ok := as.Right.(*ast.CallExpr); ok {
			if fnIdent, ok := call.Function.(*ast.Identifier); ok {
				if fnIdent.Value == "alloc" || fnIdent.Value == "mem_alloc" {
					sa.BorrowChecker.RegisterResource(lhs.Value, as.Pos())
				}
			}
		}

	case *ast.IndexExpr:
		targetType := sa.inferExprType(lhs.Left)
		idxType := sa.inferExprType(lhs.Index)
		if idxType != nil && !idxType.Equals(TypeInt) {
			sa.addError(lhs.Index.Pos(), "数组索引必须为 int")
		}
		valType := sa.inferExprType(as.Right)
		if arrType, ok := targetType.(*ArrayType); ok && arrType.ElementType != nil && !arrType.ElementType.Equals(TypeAny) {
			if valType != nil && !valType.Equals(arrType.ElementType) {
				sa.addError(as.Right.Pos(), fmt.Sprintf("数组元素类型不匹配: 期望 %s, 传入 %s", arrType.ElementType.Name(), valType.Name()))
			}
		}

	case *ast.MemberExpr:
		objType := sa.inferExprType(lhs.Object)
		valType := sa.inferExprType(as.Right)
		if st, ok := objType.(*StructType); ok {
			if expected, ok := st.Fields[lhs.Property.Value]; ok {
				if valType != nil && expected != nil && !valType.Equals(expected) && !expected.Equals(TypeAny) {
					sa.addError(as.Right.Pos(), fmt.Sprintf("结构体字段 %s.%s 类型不匹配: 期望 %s, 传入 %s", st.StructName, lhs.Property.Value, expected.Name(), valType.Name()))
				}
			} else {
				sa.addError(lhs.Property.Pos(), fmt.Sprintf("结构体 %s 没有字段 %q", st.StructName, lhs.Property.Value))
			}
		}

	default:
		sa.addError(as.Left.Pos(), "赋值操作的左值必须是标识符、数组索引或结构体字段")
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

	case *ast.ArrayLiteral:
		var elemType Type = TypeAny
		if len(e.Elements) > 0 {
			elemType = sa.inferExprType(e.Elements[0])
			for i := 1; i < len(e.Elements); i++ {
				t := sa.inferExprType(e.Elements[i])
				if elemType != nil && t != nil && !elemType.Equals(t) {
					elemType = TypeAny
				}
			}
		}
		return &ArrayType{ElementType: elemType}

	case *ast.IndexExpr:
		leftType := sa.inferExprType(e.Left)
		idxType := sa.inferExprType(e.Index)
		if idxType != nil && !idxType.Equals(TypeInt) {
			sa.addError(e.Index.Pos(), fmt.Sprintf("索引表达式类型必须为 int, 实际为 %s", idxType.Name()))
		}
		if leftType == nil {
			return TypeUnknown
		}
		if arrType, ok := leftType.(*ArrayType); ok {
			if arrType.ElementType != nil {
				return arrType.ElementType
			}
			return TypeAny
		}
		if leftType.Equals(TypeString) {
			return TypeString
		}
		if leftType.Equals(TypeAny) {
			return TypeAny
		}
		sa.addError(e.Left.Pos(), fmt.Sprintf("类型 %s 不支持索引操作", leftType.Name()))
		return TypeUnknown

	case *ast.MemberExpr:
		objType := sa.inferExprType(e.Object)
		if objType == nil {
			return TypeUnknown
		}
		if st, ok := objType.(*StructType); ok {
			if ft, ok := st.Fields[e.Property.Value]; ok {
				return ft
			}
			sa.addError(e.Property.Pos(), fmt.Sprintf("结构体 %s 没有名为 %q 的字段", st.StructName, e.Property.Value))
			return TypeUnknown
		}
		if objType.Equals(TypeAny) {
			return TypeAny
		}
		sa.addError(e.Object.Pos(), fmt.Sprintf("无法在非结构体类型 %s 上访问字段 %q", objType.Name(), e.Property.Value))
		return TypeUnknown

	case *ast.StructLiteral:
		sym, found := sa.currentScope.Resolve(e.Name.Value)
		if !found {
			sa.addError(e.Name.Pos(), fmt.Sprintf("未定义的结构体类型 %q", e.Name.Value))
			return TypeUnknown
		}
		st, ok := sym.Type.(*StructType)
		if !ok {
			sa.addError(e.Name.Pos(), fmt.Sprintf("%q 不是结构体类型", e.Name.Value))
			return TypeUnknown
		}
		for fname, valExpr := range e.Fields {
			valType := sa.inferExprType(valExpr)
			expectedType, ok := st.Fields[fname]
			if !ok {
				sa.addError(valExpr.Pos(), fmt.Sprintf("结构体 %s 没有字段 %q", st.StructName, fname))
				continue
			}
			if valType != nil && expectedType != nil && !valType.Equals(expectedType) && !expectedType.Equals(TypeAny) {
				sa.addError(valExpr.Pos(), fmt.Sprintf("结构体 %s 字段 %q 类型不匹配: 期望 %s, 传入 %s",
					st.StructName, fname, expectedType.Name(), valType.Name()))
			}
		}
		return st
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

		// 内置 range 遍历函数
		if funcIdent.Value == "range" {
			if len(e.Arguments) < 1 || len(e.Arguments) > 2 {
				sa.addError(e.Pos(), "range 期望 1 或 2 个整型参数 (end 或 start, end)")
			}
			for _, arg := range e.Arguments {
				t := sa.inferExprType(arg)
				if t != nil && !t.Equals(TypeInt) {
					sa.addError(arg.Pos(), "range 参数必须是 int")
				}
			}
			return &ArrayType{ElementType: TypeInt}
		}

		// 内置容器长度函数 len
		if funcIdent.Value == "len" {
			if len(e.Arguments) != 1 {
				sa.addError(e.Pos(), "len 期望 1 个参数")
			} else {
				t := sa.inferExprType(e.Arguments[0])
				if t != nil && !t.Equals(TypeArray) && !t.Equals(TypeString) && !t.Equals(TypeAny) {
					sa.addError(e.Arguments[0].Pos(), fmt.Sprintf("len 期望 array 或 string, 遇到 %s", t.Name()))
				}
			}
			return TypeInt
		}

		// 内置追加元素函数 append / push
		if funcIdent.Value == "append" || funcIdent.Value == "push" {
			if len(e.Arguments) != 2 {
				sa.addError(e.Pos(), fmt.Sprintf("%s 期望 2 个参数 (arr, val)", funcIdent.Value))
				return TypeArray
			}
			arrType := sa.inferExprType(e.Arguments[0])
			elemType := sa.inferExprType(e.Arguments[1])
			if arrType != nil && !arrType.Equals(TypeArray) && !arrType.Equals(TypeAny) {
				sa.addError(e.Arguments[0].Pos(), fmt.Sprintf("%s 第 1 个参数必须是 array", funcIdent.Value))
			}
			if at, ok := arrType.(*ArrayType); ok && at.ElementType != nil && !at.ElementType.Equals(TypeAny) {
				return at
			}
			return &ArrayType{ElementType: elemType}
		}

		// 内置弹栈函数 pop
		if funcIdent.Value == "pop" {
			if len(e.Arguments) != 1 {
				sa.addError(e.Pos(), "pop 期望 1 个参数")
				return TypeAny
			}
			arrType := sa.inferExprType(e.Arguments[0])
			if at, ok := arrType.(*ArrayType); ok && at.ElementType != nil {
				return at.ElementType
			}
			return TypeAny
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
