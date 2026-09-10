package eval

import (
	"fmt"
	"strings"
)

// CallFrame 表示单个函数调用栈帧
type CallFrame struct {
	FuncName string            // 函数名，如 "calculate"
	Filename string            // 源文件名
	Line     int               // 调用发生行号
	Args     []string          // 传入实参值字符串
	Locals   map[string]string // 局部变量快照
}

// CallStack 调用栈管理器
type CallStack struct {
	frames   []*CallFrame
	maxDepth int
}

// NewCallStack 创建新的调用栈 (默认最大深度 1000)
func NewCallStack(maxDepth int) *CallStack {
	if maxDepth <= 0 {
		maxDepth = 1000
	}
	return &CallStack{
		frames:   make([]*CallFrame, 0, 32),
		maxDepth: maxDepth,
	}
}

// Push 压入栈帧
func (cs *CallStack) Push(frame *CallFrame) error {
	if len(cs.frames) >= cs.maxDepth {
		return fmt.Errorf("栈溢出错误 (Stack Overflow): 调用栈层数超过上限 %d 层", cs.maxDepth)
	}
	cs.frames = append(cs.frames, frame)
	return nil
}

// Pop 弹出栈帧
func (cs *CallStack) Pop() *CallFrame {
	if len(cs.frames) == 0 {
		return nil
	}
	top := cs.frames[len(cs.frames)-1]
	cs.frames = cs.frames[:len(cs.frames)-1]
	return top
}

// Depth 获取当前调用栈深度
func (cs *CallStack) Depth() int {
	return len(cs.frames)
}

// Frames 获取当前所有栈帧副本
func (cs *CallStack) Frames() []*CallFrame {
	cp := make([]*CallFrame, len(cs.frames))
	copy(cp, cs.frames)
	return cp
}

// FormatTrace 格式化输出调用栈回溯文本
func (cs *CallStack) FormatTrace() string {
	if len(cs.frames) == 0 {
		return "  (调用栈为空)"
	}
	var sb strings.Builder
	for i := len(cs.frames) - 1; i >= 0; i-- {
		f := cs.frames[i]
		argList := strings.Join(f.Args, ", ")
		sb.WriteString(fmt.Sprintf("  #%d  %s(%s) [%s:%d]\n", i, f.FuncName, argList, f.Filename, f.Line))
	}
	return strings.TrimSuffix(sb.String(), "\n")
}
