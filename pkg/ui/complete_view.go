package ui

import (
	"encoding/json"
	"fmt"
	"strings"

	"lcode/pkg/complete"
)

// RenderCompletionCard 渲染自动补全建议卡片
func RenderCompletionCard(items []complete.CompletionItem, prefix string) string {
	if len(items) == 0 {
		return Paint(ColorMuted, fmt.Sprintf("未找到以 %q 开头的补全候选项", prefix))
	}

	var sb strings.Builder
	for i, it := range items {
		kindColor := ColorViolet
		switch it.Kind {
		case "Keyword":
			kindColor = ColorViolet
		case "Type":
			kindColor = ColorEmerald
		case "Function":
			kindColor = ColorAmber
		case "Variable", "Constant":
			kindColor = ColorCyan
		case "Module":
			kindColor = ColorRose
		}

		kindBadge := Paint(kindColor+Bold, fmt.Sprintf("%-10s", "["+it.Kind+"]"))
		labelStr := Paint(Bold+FgHiWhite, fmt.Sprintf("%-18s", it.Label))
		detailStr := Paint(ColorCyan, fmt.Sprintf("%-32s", it.Detail))
		docStr := Paint(ColorMuted, it.Documentation)

		sb.WriteString(fmt.Sprintf("%s %s %s %s", kindBadge, labelStr, detailStr, docStr))
		if i < len(items)-1 {
			sb.WriteString("\n")
		}
	}

	title := fmt.Sprintf("💡 智能代码补全建议: %q (匹配 %d 项)", prefix, len(items))
	return BoxWithTitle(title, sb.String(), ColorCyan)
}

// RenderCompletionJSON 输出机器可读的 JSON 格式
func RenderCompletionJSON(items []complete.CompletionItem) (string, error) {
	bytes, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
