package ast

import (
	"fmt"
	"strings"
)

// Visitor 抽象语法树访问者接口
type Visitor interface {
	Visit(node Node) (w Visitor)
}

// Walk 深度优先遍历 AST
func Walk(v Visitor, node Node) {
	if node == nil {
		return
	}

	w := v.Visit(node)
	if w == nil {
		return
	}

	switch n := node.(type) {
	case *Program:
		for _, decl := range n.Decls {
			Walk(w, decl)
		}
		for _, stmt := range n.Stmts {
			Walk(w, stmt)
		}

	case *FuncDecl:
		Walk(w, n.Name)
		for _, p := range n.Params {
			Walk(w, p.Name)
			if p.Type != nil {
				Walk(w, p.Type)
			}
		}
		if n.ReturnType != nil {
			Walk(w, n.ReturnType)
		}
		Walk(w, n.Body)

	case *VarDeclStmt:
		Walk(w, n.Name)
		if n.Type != nil {
			Walk(w, n.Type)
		}
		if n.Value != nil {
			Walk(w, n.Value)
		}

	case *AssignStmt:
		Walk(w, n.Left)
		Walk(w, n.Right)

	case *BlockStmt:
		for _, stmt := range n.Statements {
			Walk(w, stmt)
		}

	case *ExprStmt:
		if n.Expression != nil {
			Walk(w, n.Expression)
		}

	case *IfStmt:
		Walk(w, n.Condition)
		Walk(w, n.Consequence)
		if n.Alternative != nil {
			Walk(w, n.Alternative)
		}

	case *WhileStmt:
		Walk(w, n.Condition)
		Walk(w, n.Body)

	case *ReturnStmt:
		if n.ReturnValue != nil {
			Walk(w, n.ReturnValue)
		}

	case *UnaryExpr:
		Walk(w, n.Right)

	case *BinaryExpr:
		Walk(w, n.Left)
		Walk(w, n.Right)

	case *CallExpr:
		Walk(w, n.Function)
		for _, arg := range n.Arguments {
			Walk(w, arg)
		}

	case *GroupedExpr:
		Walk(w, n.Expression)
	}
}

// TreeDumper 格式化树状输出 AST 结构
type TreeDumper struct {
	depth int
	sb    strings.Builder
}

func (d *TreeDumper) indent() string {
	return strings.Repeat("  ", d.depth)
}

// DumpAST 将 AST 格式化为树状字符串
func DumpAST(node Node) string {
	if node == nil {
		return "<nil>"
	}
	d := &TreeDumper{}
	d.dump(node)
	return d.sb.String()
}

func (d *TreeDumper) dump(node Node) {
	if node == nil {
		return
	}

	prefix := d.indent()
	switch n := node.(type) {
	case *Program:
		d.sb.WriteString(fmt.Sprintf("%sProgram\n", prefix))
		d.depth++
		for _, decl := range n.Decls {
			d.dump(decl)
		}
		for _, stmt := range n.Stmts {
			d.dump(stmt)
		}
		d.depth--

	case *FuncDecl:
		ret := "void"
		if n.ReturnType != nil {
			ret = n.ReturnType.Name
		}
		d.sb.WriteString(fmt.Sprintf("%sFuncDecl '%s' -> %s (%s)\n", prefix, n.Name.Value, ret, n.Pos()))
		d.depth++
		for _, p := range n.Params {
			d.sb.WriteString(fmt.Sprintf("%sParam '%s': %s\n", d.indent(), p.Name.Value, p.Type.Name))
		}
		d.dump(n.Body)
		d.depth--

	case *VarDeclStmt:
		typeStr := "<inferred>"
		if n.Type != nil {
			typeStr = n.Type.Name
		}
		d.sb.WriteString(fmt.Sprintf("%sVarDecl '%s': %s (%s)\n", prefix, n.Name.Value, typeStr, n.Pos()))
		if n.Value != nil {
			d.depth++
			d.dump(n.Value)
			d.depth--
		}

	case *AssignStmt:
		d.sb.WriteString(fmt.Sprintf("%sAssignStmt '%s'\n", prefix, n.Operator))
		d.depth++
		d.dump(n.Left)
		d.dump(n.Right)
		d.depth--

	case *BlockStmt:
		d.sb.WriteString(fmt.Sprintf("%sBlockStmt\n", prefix))
		d.depth++
		for _, s := range n.Statements {
			d.dump(s)
		}
		d.depth--

	case *ExprStmt:
		d.sb.WriteString(fmt.Sprintf("%sExprStmt\n", prefix))
		d.depth++
		d.dump(n.Expression)
		d.depth--

	case *IfStmt:
		d.sb.WriteString(fmt.Sprintf("%sIfStmt\n", prefix))
		d.depth++
		d.sb.WriteString(fmt.Sprintf("%sCondition:\n", d.indent()))
		d.depth++
		d.dump(n.Condition)
		d.depth--
		d.sb.WriteString(fmt.Sprintf("%sThen:\n", d.indent()))
		d.depth++
		d.dump(n.Consequence)
		d.depth--
		if n.Alternative != nil {
			d.sb.WriteString(fmt.Sprintf("%sElse:\n", d.indent()))
			d.depth++
			d.dump(n.Alternative)
			d.depth--
		}
		d.depth--

	case *WhileStmt:
		d.sb.WriteString(fmt.Sprintf("%sWhileStmt\n", prefix))
		d.depth++
		d.sb.WriteString(fmt.Sprintf("%sCondition:\n", d.indent()))
		d.depth++
		d.dump(n.Condition)
		d.depth--
		d.sb.WriteString(fmt.Sprintf("%sBody:\n", d.indent()))
		d.depth++
		d.dump(n.Body)
		d.depth--
		d.depth--

	case *ReturnStmt:
		d.sb.WriteString(fmt.Sprintf("%sReturnStmt\n", prefix))
		if n.ReturnValue != nil {
			d.depth++
			d.dump(n.ReturnValue)
			d.depth--
		}

	case *BinaryExpr:
		d.sb.WriteString(fmt.Sprintf("%sBinaryExpr '%s'\n", prefix, n.Operator))
		d.depth++
		d.dump(n.Left)
		d.dump(n.Right)
		d.depth--

	case *UnaryExpr:
		d.sb.WriteString(fmt.Sprintf("%sUnaryExpr '%s'\n", prefix, n.Operator))
		d.depth++
		d.dump(n.Right)
		d.depth--

	case *CallExpr:
		d.sb.WriteString(fmt.Sprintf("%sCallExpr\n", prefix))
		d.depth++
		d.dump(n.Function)
		for _, arg := range n.Arguments {
			d.dump(arg)
		}
		d.depth--

	case *Identifier:
		d.sb.WriteString(fmt.Sprintf("%sIdent '%s'\n", prefix, n.Value))

	case *IntegerLiteral:
		d.sb.WriteString(fmt.Sprintf("%sIntLit %d\n", prefix, n.Value))

	case *FloatLiteral:
		d.sb.WriteString(fmt.Sprintf("%sFloatLit %f\n", prefix, n.Value))

	case *StringLiteral:
		d.sb.WriteString(fmt.Sprintf("%sStringLit %q\n", prefix, n.Value))

	case *BooleanLiteral:
		d.sb.WriteString(fmt.Sprintf("%sBoolLit %t\n", prefix, n.Value))

	default:
		d.sb.WriteString(fmt.Sprintf("%sNode(%s)\n", prefix, node.String()))
	}
}
