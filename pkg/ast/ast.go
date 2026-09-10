package ast

import (
	"bytes"
	"fmt"
	"strings"

	"lcode/pkg/token"
)

// Node 抽象语法树的所有节点的基础接口
type Node interface {
	Pos() token.Position
	String() string
}

// Expr 表示表达式节点
type Expr interface {
	Node
	exprNode()
}

// Stmt 表示语句节点
type Stmt interface {
	Node
	stmtNode()
}

// Decl 表示顶级声明节点
type Decl interface {
	Node
	declNode()
}

// ==========================================
// 根节点
// ==========================================

// Program 代表整个源文件的 AST 根
type Program struct {
	Imports []*ImportDecl
	Decls   []Decl
	Stmts   []Stmt
}

func (p *Program) Pos() token.Position {
	if len(p.Imports) > 0 {
		return p.Imports[0].Pos()
	}
	if len(p.Decls) > 0 {
		return p.Decls[0].Pos()
	}
	if len(p.Stmts) > 0 {
		return p.Stmts[0].Pos()
	}
	return token.Position{}
}

func (p *Program) String() string {
	var out bytes.Buffer
	for _, imp := range p.Imports {
		out.WriteString(imp.String() + "\n")
	}
	for _, d := range p.Decls {
		out.WriteString(d.String() + "\n")
	}
	for _, s := range p.Stmts {
		out.WriteString(s.String() + "\n")
	}
	return out.String()
}

// ImportDecl 模块导入声明 (import "std/math"; 或 import "std/math" as m;)
type ImportDecl struct {
	Token token.Token // 'import'
	Path  string      // 导入路径，如 "std/math" 或 "./utils.lc"
	Alias string      // 可选别名
}

func (id *ImportDecl) Pos() token.Position { return id.Token.Pos }
func (id *ImportDecl) declNode()           {}
func (id *ImportDecl) stmtNode()           {}
func (id *ImportDecl) String() string {
	if id.Alias != "" {
		return fmt.Sprintf("import %q as %s;", id.Path, id.Alias)
	}
	return fmt.Sprintf("import %q;", id.Path)
}

// ==========================================
// 辅助与类型标注
// ==========================================

// TypeAnnotation 表示显式类型标注（如 `: int`）
type TypeAnnotation struct {
	Token token.Token // 类型的起始 token
	Name  string      // 类型名，如 "int", "string", "bool"
}

func (t *TypeAnnotation) Pos() token.Position { return t.Token.Pos }
func (t *TypeAnnotation) String() string       { return t.Name }

// Param 表示函数形参
type Param struct {
	Name *Identifier
	Type *TypeAnnotation
}

func (p *Param) Pos() token.Position { return p.Name.Pos() }
func (p *Param) String() string {
	if p.Type != nil {
		return fmt.Sprintf("%s: %s", p.Name.Value, p.Type.Name)
	}
	return p.Name.Value
}

// ==========================================
// 声明与语句 (Statements & Declarations)
// ==========================================

// FuncDecl 函数声明
type FuncDecl struct {
	Token      token.Token // 'fn' 或 'func' 关键字
	Name       *Identifier
	Params     []*Param
	ReturnType *TypeAnnotation
	Body       *BlockStmt
}

func (fd *FuncDecl) Pos() token.Position { return fd.Token.Pos }
func (fd *FuncDecl) declNode()           {}
func (fd *FuncDecl) stmtNode()           {} // 函数也可以作为局部定义
func (fd *FuncDecl) String() string {
	var out bytes.Buffer
	params := make([]string, 0, len(fd.Params))
	for _, p := range fd.Params {
		params = append(params, p.String())
	}

	out.WriteString("fn " + fd.Name.Value + "(" + strings.Join(params, ", ") + ")")
	if fd.ReturnType != nil {
		out.WriteString(" -> " + fd.ReturnType.Name)
	}
	out.WriteString(" " + fd.Body.String())
	return out.String()
}

// VarDeclStmt 变量声明语句 (let a = 1; 或 mut b: int = 2;)
type VarDeclStmt struct {
	Token   token.Token // 'let', 'mut', 'var', 'const'
	Name    *Identifier
	Type    *TypeAnnotation
	Value   Expr
	IsMut   bool
	IsConst bool
}

func (v *VarDeclStmt) Pos() token.Position { return v.Token.Pos }
func (v *VarDeclStmt) stmtNode()           {}
func (v *VarDeclStmt) declNode()           {}
func (v *VarDeclStmt) String() string {
	var out bytes.Buffer
	keyword := "let"
	if v.IsMut {
		keyword = "mut"
	} else if v.Token.Literal != "" {
		keyword = v.Token.Literal
	}

	out.WriteString(keyword + " " + v.Name.Value)
	if v.Type != nil {
		out.WriteString(": " + v.Type.Name)
	}
	if v.Value != nil {
		out.WriteString(" = " + v.Value.String())
	}
	out.WriteString(";")
	return out.String()
}

// AssignStmt 赋值语句 (a = 1; 或 a += 2;)
type AssignStmt struct {
	Left     Expr
	Operator string // '=', '+=', '-=', etc.
	Right    Expr
}

func (as *AssignStmt) Pos() token.Position { return as.Left.Pos() }
func (as *AssignStmt) stmtNode()           {}
func (as *AssignStmt) String() string {
	return fmt.Sprintf("%s %s %s;", as.Left.String(), as.Operator, as.Right.String())
}

// BlockStmt 块语句 { ... }
type BlockStmt struct {
	Token      token.Token // '{'
	Statements []Stmt
}

func (bs *BlockStmt) Pos() token.Position { return bs.Token.Pos }
func (bs *BlockStmt) stmtNode()           {}
func (bs *BlockStmt) String() string {
	var out bytes.Buffer
	out.WriteString("{\n")
	for _, s := range bs.Statements {
		out.WriteString("  " + s.String() + "\n")
	}
	out.WriteString("}")
	return out.String()
}

// ExprStmt 表达式语句
type ExprStmt struct {
	Token      token.Token // 表达式的第一个 token
	Expression Expr
}

func (es *ExprStmt) Pos() token.Position { return es.Token.Pos }
func (es *ExprStmt) stmtNode()           {}
func (es *ExprStmt) String() string {
	if es.Expression != nil {
		return es.Expression.String() + ";"
	}
	return ";"
}

// IfStmt 条件语句
type IfStmt struct {
	Token       token.Token // 'if'
	Condition   Expr
	Consequence *BlockStmt
	Alternative Stmt // 可以是 *BlockStmt 或 *IfStmt (else if)
}

func (is *IfStmt) Pos() token.Position { return is.Token.Pos }
func (is *IfStmt) stmtNode()           {}
func (is *IfStmt) String() string {
	var out bytes.Buffer
	out.WriteString("if " + is.Condition.String() + " " + is.Consequence.String())
	if is.Alternative != nil {
		out.WriteString(" else " + is.Alternative.String())
	}
	return out.String()
}

// WhileStmt 循环语句
type WhileStmt struct {
	Token     token.Token // 'while'
	Condition Expr
	Body      *BlockStmt
}

func (ws *WhileStmt) Pos() token.Position { return ws.Token.Pos }
func (ws *WhileStmt) stmtNode()           {}
func (ws *WhileStmt) String() string {
	return fmt.Sprintf("while %s %s", ws.Condition.String(), ws.Body.String())
}

// ReturnStmt 返回语句
type ReturnStmt struct {
	Token       token.Token // 'return'
	ReturnValue Expr
}

func (rs *ReturnStmt) Pos() token.Position { return rs.Token.Pos }
func (rs *ReturnStmt) stmtNode()           {}
func (rs *ReturnStmt) String() string {
	if rs.ReturnValue != nil {
		return fmt.Sprintf("return %s;", rs.ReturnValue.String())
	}
	return "return;"
}

// ForInStmt for ... in 循环语句 (for x in arr { ... })
type ForInStmt struct {
	Token    token.Token // 'for'
	VarName  *Identifier // 迭代变量
	Iterable Expr        // 迭代集合或 range
	Body     *BlockStmt
}

func (fs *ForInStmt) Pos() token.Position { return fs.Token.Pos }
func (fs *ForInStmt) stmtNode()           {}
func (fs *ForInStmt) String() string {
	return fmt.Sprintf("for %s in %s %s", fs.VarName.Value, fs.Iterable.String(), fs.Body.String())
}

// BreakStmt 循环跳出语句 (break;)
type BreakStmt struct {
	Token token.Token // 'break'
}

func (bs *BreakStmt) Pos() token.Position { return bs.Token.Pos }
func (bs *BreakStmt) stmtNode()           {}
func (bs *BreakStmt) String() string       { return "break;" }

// ContinueStmt 循环继续语句 (continue;)
type ContinueStmt struct {
	Token token.Token // 'continue'
}

func (cs *ContinueStmt) Pos() token.Position { return cs.Token.Pos }
func (cs *ContinueStmt) stmtNode()           {}
func (cs *ContinueStmt) String() string       { return "continue;" }

// TryCatchStmt 异常容错捕获语句 (try { ... } catch (err) { ... })
type TryCatchStmt struct {
	Token      token.Token // 'try'
	TryBlock   *BlockStmt
	ErrVar     *Identifier // 可选，如 err
	CatchBlock *BlockStmt
}

func (ts *TryCatchStmt) Pos() token.Position { return ts.Token.Pos }
func (ts *TryCatchStmt) stmtNode()           {}
func (ts *TryCatchStmt) String() string {
	errStr := ""
	if ts.ErrVar != nil {
		errStr = "(" + ts.ErrVar.Value + ")"
	}
	return fmt.Sprintf("try %s catch %s %s", ts.TryBlock.String(), errStr, ts.CatchBlock.String())
}

// StructField 结构体字段定义
type StructField struct {
	Name *Identifier
	Type *TypeAnnotation
}

func (sf *StructField) String() string {
	if sf.Type != nil {
		return fmt.Sprintf("%s: %s", sf.Name.Value, sf.Type.Name)
	}
	return sf.Name.Value
}

// StructDecl 结构体类型声明 (struct Point { x: int, y: int })
type StructDecl struct {
	Token  token.Token // 'struct'
	Name   *Identifier
	Fields []*StructField
}

func (sd *StructDecl) Pos() token.Position { return sd.Token.Pos }
func (sd *StructDecl) declNode()           {}
func (sd *StructDecl) stmtNode()           {}
func (sd *StructDecl) String() string {
	var out bytes.Buffer
	out.WriteString("struct " + sd.Name.Value + " {\n")
	for _, f := range sd.Fields {
		out.WriteString("  " + f.String() + ",\n")
	}
	out.WriteString("}")
	return out.String()
}

// ==========================================
// 表达式 (Expressions)
// ==========================================

// Identifier 标识符
type Identifier struct {
	Token token.Token
	Value string
}

func (i *Identifier) Pos() token.Position { return i.Token.Pos }
func (i *Identifier) exprNode()           {}
func (i *Identifier) String() string       { return i.Value }

// IntegerLiteral 整型字面量
type IntegerLiteral struct {
	Token token.Token
	Value int64
}

func (il *IntegerLiteral) Pos() token.Position { return il.Token.Pos }
func (il *IntegerLiteral) exprNode()           {}
func (il *IntegerLiteral) String() string       { return il.Token.Literal }

// FloatLiteral 浮点字面量
type FloatLiteral struct {
	Token token.Token
	Value float64
}

func (fl *FloatLiteral) Pos() token.Position { return fl.Token.Pos }
func (fl *FloatLiteral) exprNode()           {}
func (fl *FloatLiteral) String() string       { return fl.Token.Literal }

// StringLiteral 字符串字面量
type StringLiteral struct {
	Token token.Token
	Value string
}

func (sl *StringLiteral) Pos() token.Position { return sl.Token.Pos }
func (sl *StringLiteral) exprNode()           {}
func (sl *StringLiteral) String() string       { return fmt.Sprintf("%q", sl.Value) }

// BooleanLiteral 布尔字面量
type BooleanLiteral struct {
	Token token.Token
	Value bool
}

func (bl *BooleanLiteral) Pos() token.Position { return bl.Token.Pos }
func (bl *BooleanLiteral) exprNode()           {}
func (bl *BooleanLiteral) String() string       { return bl.Token.Literal }

// UnaryExpr 一元运算表达式 (-x, !flag)
type UnaryExpr struct {
	Token    token.Token // 操作符
	Operator string
	Right    Expr
}

func (ue *UnaryExpr) Pos() token.Position { return ue.Token.Pos }
func (ue *UnaryExpr) exprNode()           {}
func (ue *UnaryExpr) String() string {
	return fmt.Sprintf("(%s%s)", ue.Operator, ue.Right.String())
}

// BinaryExpr 二元运算表达式 (a + b, x == y)
type BinaryExpr struct {
	Token    token.Token // 操作符
	Left     Expr
	Operator string
	Right    Expr
}

func (be *BinaryExpr) Pos() token.Position { return be.Left.Pos() }
func (be *BinaryExpr) exprNode()           {}
func (be *BinaryExpr) String() string {
	return fmt.Sprintf("(%s %s %s)", be.Left.String(), be.Operator, be.Right.String())
}

// CallExpr 函数调用表达式 add(a, b)
type CallExpr struct {
	Token     token.Token // '('
	Function  Expr        // 标识符或复合表达式
	Arguments []Expr
}

func (ce *CallExpr) Pos() token.Position { return ce.Function.Pos() }
func (ce *CallExpr) exprNode()           {}
func (ce *CallExpr) String() string {
	var out bytes.Buffer
	args := make([]string, 0, len(ce.Arguments))
	for _, a := range ce.Arguments {
		args = append(args, a.String())
	}
	out.WriteString(ce.Function.String())
	out.WriteString("(")
	out.WriteString(strings.Join(args, ", "))
	out.WriteString(")")
	return out.String()
}

// GroupedExpr 括号表达式 (a + b)
type GroupedExpr struct {
	Token      token.Token // '('
	Expression Expr
}

func (ge *GroupedExpr) Pos() token.Position { return ge.Token.Pos }
func (ge *GroupedExpr) exprNode()           {}
func (ge *GroupedExpr) String() string       { return ge.Expression.String() }

// ArrayLiteral 数组字面量 [1, 2, 3]
type ArrayLiteral struct {
	Token    token.Token // '['
	Elements []Expr
}

func (al *ArrayLiteral) Pos() token.Position { return al.Token.Pos }
func (al *ArrayLiteral) exprNode()           {}
func (al *ArrayLiteral) String() string {
	var parts []string
	for _, el := range al.Elements {
		parts = append(parts, el.String())
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// IndexExpr 下标索引表达式 arr[i]
type IndexExpr struct {
	Token token.Token // '['
	Left  Expr        // 数组或可索引对象
	Index Expr        // 下标表达式
}

func (ie *IndexExpr) Pos() token.Position { return ie.Left.Pos() }
func (ie *IndexExpr) exprNode()           {}
func (ie *IndexExpr) String() string {
	return fmt.Sprintf("%s[%s]", ie.Left.String(), ie.Index.String())
}

// MemberExpr 成员点号属性访问 (user.name)
type MemberExpr struct {
	Token    token.Token // '.'
	Object   Expr
	Property *Identifier
}

func (me *MemberExpr) Pos() token.Position { return me.Object.Pos() }
func (me *MemberExpr) exprNode()           {}
func (me *MemberExpr) String() string {
	return fmt.Sprintf("%s.%s", me.Object.String(), me.Property.Value)
}

// StructLiteral 结构体实例化 (Point { x: 10, y: 20 })
type StructLiteral struct {
	Token  token.Token // 结构体名称 Token
	Name   *Identifier
	Fields map[string]Expr
}

func (sl *StructLiteral) Pos() token.Position { return sl.Token.Pos }
func (sl *StructLiteral) exprNode()           {}
func (sl *StructLiteral) String() string {
	var parts []string
	for k, v := range sl.Fields {
		parts = append(parts, fmt.Sprintf("%s: %s", k, v.String()))
	}
	return sl.Name.Value + " { " + strings.Join(parts, ", ") + " }"
}
