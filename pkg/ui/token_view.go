package ui

import (
	"fmt"
	"strings"

	"lcode/pkg/token"
)

// RenderTokenTable 将 Tokens 序列渲染为精致的高颜值终端表格
func RenderTokenTable(tokens []token.Token) string {
	var sb strings.Builder

	header := fmt.Sprintf("%-5s │ %-12s │ %-20s │ %-10s", "序号", "类别", "字面量 (Literal)", "位置 (Pos)")
	divider := strings.Repeat("─", 5) + "┼" + strings.Repeat("─", 14) + "┼" + strings.Repeat("─", 22) + "┼" + strings.Repeat("─", 12)

	sb.WriteString(ColorCyan + header + Reset + "\n")
	sb.WriteString(ColorMuted + divider + Reset + "\n")

	for i, tok := range tokens {
		catColor := getTokenColor(tok.Type)
		typeBadge := Paint(catColor+Bold, fmt.Sprintf("%-12s", tok.Type))
		litStr := fmt.Sprintf("%-20q", tok.Literal)
		if tok.Type == token.IDENT || tok.Type == token.INT || tok.Type == token.FLOAT {
			litStr = fmt.Sprintf("%-20s", tok.Literal)
		}
		posStr := fmt.Sprintf("%-10s", fmt.Sprintf("%d:%d", tok.Pos.Line, tok.Pos.Column))

		line := fmt.Sprintf("%s%4d%s  │ %s │ %s%s%s │ %s%s%s",
			ColorMuted, i+1, Reset,
			typeBadge,
			FgHiWhite, litStr, Reset,
			ColorMuted, posStr, Reset,
		)
		sb.WriteString(line + "\n")
	}

	return sb.String()
}

func getTokenColor(t token.TokenType) string {
	switch t {
	case token.FN, token.FUNC, token.LET, token.MUT, token.CONST, token.IF, token.ELSE, token.WHILE, token.RETURN:
		return ColorViolet
	case token.IDENT:
		return ColorCyan
	case token.INT, token.FLOAT, token.STRING, token.BOOL:
		return ColorEmerald
	case token.ADD, token.SUB, token.MUL, token.QUO, token.REM, token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ, token.ASSIGN, token.ARROW:
		return ColorAmber
	default:
		return ColorMuted
	}
}
