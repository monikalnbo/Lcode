package ui

import (
	"fmt"
	"strings"

	"lcode/pkg/ast"
)

// RenderColorAST 将 AST 节点渲染为具有树形分支线 (├──, └──) 的高颜值彩色语法树
func RenderColorAST(node ast.Node) string {
	var sb strings.Builder
	renderNode(node, "", true, &sb)
	return sb.String()
}

func renderNode(node ast.Node, prefix string, isLast bool, sb *strings.Builder) {
	if node == nil {
		return
	}

	branch := "├── "
	if isLast {
		branch = "└── "
	}
	nextPrefix := prefix
	if isLast {
		nextPrefix += "    "
	} else {
		nextPrefix += "│   "
	}

	linePrefix := prefix + ColorMuted + branch + Reset

	switch n := node.(type) {
	case *ast.Program:
		sb.WriteString(Bold + ColorViolet + "Program" + Reset + " " + ColorMuted + "(Root)" + Reset + "\n")
		total := len(n.Decls) + len(n.Stmts)
		idx := 0
		for _, decl := range n.Decls {
			idx++
			renderNode(decl, "", idx == total, sb)
		}
		for _, stmt := range n.Stmts {
			idx++
			renderNode(stmt, "", idx == total, sb)
		}

	case *ast.FuncDecl:
		ret := "void"
		if n.ReturnType != nil {
			ret = n.ReturnType.Name
		}
		sb.WriteString(fmt.Sprintf("%s%sFuncDecl%s %s%s%s -> %s%s%s %s(%s)%s\n",
			linePrefix,
			Bold+ColorViolet, Reset,
			Bold+FgHiWhite, n.Name.Value, Reset,
			ColorCyan, ret, Reset,
			ColorMuted, n.Pos(), Reset))

		paramsAndBody := len(n.Params) + 1
		for _, p := range n.Params {
			pType := "any"
			if p.Type != nil {
				pType = p.Type.Name
			}
			pLine := fmt.Sprintf("%s%s├── %sParam%s %s%s%s: %s%s%s\n",
				nextPrefix, ColorMuted,
				ColorAmber, Reset,
				FgHiWhite, p.Name.Value, Reset,
				ColorCyan, pType, Reset)
			sb.WriteString(pLine)
		}
		renderNode(n.Body, nextPrefix, paramsAndBody == len(n.Params)+1, sb)

	case *ast.VarDeclStmt:
		typeStr := "<推导>"
		if n.Type != nil {
			typeStr = n.Type.Name
		}
		kindBadge := Paint(ColorEmerald, "let")
		if n.IsMut {
			kindBadge = Paint(ColorRose, "mut")
		}
		sb.WriteString(fmt.Sprintf("%s%sVarDecl%s [%s] %s%s%s: %s%s%s %s(%s)%s\n",
			linePrefix,
			Bold+ColorCyan, Reset,
			kindBadge,
			Bold+FgHiWhite, n.Name.Value, Reset,
			ColorCyan, typeStr, Reset,
			ColorMuted, n.Pos(), Reset))
		if n.Value != nil {
			renderNode(n.Value, nextPrefix, true, sb)
		}

	case *ast.AssignStmt:
		sb.WriteString(fmt.Sprintf("%s%sAssignStmt%s %s%q%s\n",
			linePrefix,
			Bold+ColorAmber, Reset,
			FgHiWhite, n.Operator, Reset))
		renderNode(n.Left, nextPrefix, false, sb)
		renderNode(n.Right, nextPrefix, true, sb)

	case *ast.BlockStmt:
		sb.WriteString(fmt.Sprintf("%s%sBlockStmt%s %s{ %d 条语句 }%s\n",
			linePrefix,
			Bold+ColorMuted, Reset,
			ColorMuted, len(n.Statements), Reset))
		for i, s := range n.Statements {
			renderNode(s, nextPrefix, i == len(n.Statements)-1, sb)
		}

	case *ast.ExprStmt:
		sb.WriteString(fmt.Sprintf("%s%sExprStmt%s\n", linePrefix, Bold+ColorMuted, Reset))
		if n.Expression != nil {
			renderNode(n.Expression, nextPrefix, true, sb)
		}

	case *ast.IfStmt:
		sb.WriteString(fmt.Sprintf("%s%sIfStmt%s\n", linePrefix, Bold+ColorAmber, Reset))
		renderNode(n.Condition, nextPrefix, false, sb)
		renderNode(n.Consequence, nextPrefix, n.Alternative == nil, sb)
		if n.Alternative != nil {
			renderNode(n.Alternative, nextPrefix, true, sb)
		}

	case *ast.WhileStmt:
		sb.WriteString(fmt.Sprintf("%s%sWhileStmt%s\n", linePrefix, Bold+ColorAmber, Reset))
		renderNode(n.Condition, nextPrefix, false, sb)
		renderNode(n.Body, nextPrefix, true, sb)

	case *ast.ReturnStmt:
		sb.WriteString(fmt.Sprintf("%s%sReturnStmt%s\n", linePrefix, Bold+ColorRose, Reset))
		if n.ReturnValue != nil {
			renderNode(n.ReturnValue, nextPrefix, true, sb)
		}

	case *ast.BinaryExpr:
		sb.WriteString(fmt.Sprintf("%s%sBinaryExpr%s %s%q%s\n",
			linePrefix,
			Bold+ColorAmber, Reset,
			Bold+FgHiYellow, n.Operator, Reset))
		renderNode(n.Left, nextPrefix, false, sb)
		renderNode(n.Right, nextPrefix, true, sb)

	case *ast.UnaryExpr:
		sb.WriteString(fmt.Sprintf("%s%sUnaryExpr%s %s%q%s\n",
			linePrefix,
			Bold+ColorAmber, Reset,
			Bold+FgHiYellow, n.Operator, Reset))
		renderNode(n.Right, nextPrefix, true, sb)

	case *ast.CallExpr:
		sb.WriteString(fmt.Sprintf("%s%sCallExpr%s\n", linePrefix, Bold+ColorViolet, Reset))
		renderNode(n.Function, nextPrefix, len(n.Arguments) == 0, sb)
		for i, arg := range n.Arguments {
			renderNode(arg, nextPrefix, i == len(n.Arguments)-1, sb)
		}

	case *ast.Identifier:
		sb.WriteString(fmt.Sprintf("%s%sIdent%s %s%s%s\n",
			linePrefix,
			ColorCyan, Reset,
			Bold+FgHiWhite, n.Value, Reset))

	case *ast.IntegerLiteral:
		sb.WriteString(fmt.Sprintf("%s%sIntLit%s %s%d%s\n",
			linePrefix,
			ColorEmerald, Reset,
			Bold+FgHiGreen, n.Value, Reset))

	case *ast.FloatLiteral:
		sb.WriteString(fmt.Sprintf("%s%sFloatLit%s %s%f%s\n",
			linePrefix,
			ColorEmerald, Reset,
			Bold+FgHiGreen, n.Value, Reset))

	case *ast.StringLiteral:
		sb.WriteString(fmt.Sprintf("%s%sStringLit%s %s%q%s\n",
			linePrefix,
			ColorEmerald, Reset,
			Bold+FgHiGreen, n.Value, Reset))

	case *ast.BooleanLiteral:
		sb.WriteString(fmt.Sprintf("%s%sBoolLit%s %s%t%s\n",
			linePrefix,
			ColorAmber, Reset,
			Bold+FgHiYellow, n.Value, Reset))

	default:
		sb.WriteString(fmt.Sprintf("%sNode(%s)\n", linePrefix, node.String()))
	}
}
