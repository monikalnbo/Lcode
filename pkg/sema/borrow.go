package sema

import (
	"fmt"

	"lcode/pkg/token"
)

// OwnershipState 所有权状态枚举
type OwnershipState int

const (
	StateOwned       OwnershipState = iota // 正常持有所有权
	StateMoved                             // 所有权已被移动转移 (再次使用即触发 E0382 错误)
	StateBorrowed                          // 处于共享不可变借用中
	StateMutBorrowed                       // 处于独占可变借用中
)

func (s OwnershipState) String() string {
	switch s {
	case StateOwned:
		return "Owned(正常持有)"
	case StateMoved:
		return "Moved(已移动)"
	case StateBorrowed:
		return "Borrowed(借用中)"
	case StateMutBorrowed:
		return "MutBorrowed(独占可变借用)"
	default:
		return "Unknown"
	}
}

// MoveRecord 记录所有权移动发生的具体细节
type MoveRecord struct {
	VarName    string         // 发生移动的原变量名
	TargetName string         // 接收所有权的新变量名
	MovedPos   token.Position // 移动发生的源码位置
}

// BorrowChecker 编译期借用检查器
type BorrowChecker struct {
	states      map[string]OwnershipState
	moveHistory map[string]*MoveRecord
	isResource  map[string]bool // 是否属于需要严格所有权追踪的堆内存或复合资源
	Errors      []string
}

// NewBorrowChecker 创建借用检查器
func NewBorrowChecker() *BorrowChecker {
	return &BorrowChecker{
		states:      make(map[string]OwnershipState),
		moveHistory: make(map[string]*MoveRecord),
		isResource:  make(map[string]bool),
		Errors:      make([]string, 0),
	}
}

// RegisterResource 注册变量为所有权托管资源 (如 alloc 结果、堆缓冲区、复合结构)
func (bc *BorrowChecker) RegisterResource(varName string, pos token.Position) {
	bc.states[varName] = StateOwned
	bc.isResource[varName] = true
}

// Move 所有权转移：将 src 的所有权转移至 dest
func (bc *BorrowChecker) Move(src string, dest string, pos token.Position) error {
	state, exists := bc.states[src]
	if !exists || !bc.isResource[src] {
		// 基础 Copy 类型 (如纯整型字面量) 默认具有 Copy 语义，不强制转移
		return nil
	}

	if state == StateMoved {
		prev := bc.moveHistory[src]
		msg := fmt.Sprintf("%s: [E0382] 非法使用已移动的值: 变量 %q 的所有权已经在 %s 处转移给 %q，无法再次移动",
			pos, src, prev.MovedPos, prev.TargetName)
		bc.Errors = append(bc.Errors, msg)
		return fmt.Errorf(msg)
	}

	// 标记旧变量为已移动
	bc.states[src] = StateMoved
	bc.moveHistory[src] = &MoveRecord{
		VarName:    src,
		TargetName: dest,
		MovedPos:   pos,
	}

	// 新变量获得所有权
	bc.states[dest] = StateOwned
	bc.isResource[dest] = true
	return nil
}

// Borrow 不可变借用 (&x)
func (bc *BorrowChecker) Borrow(varName string, pos token.Position) error {
	state, exists := bc.states[varName]
	if !exists || !bc.isResource[varName] {
		return nil
	}

	if state == StateMoved {
		prev := bc.moveHistory[varName]
		msg := fmt.Sprintf("%s: [E0382] 非法借用已移动的值: 变量 %q 的所有权已在 %s 处移走，禁止借用",
			pos, varName, prev.MovedPos)
		bc.Errors = append(bc.Errors, msg)
		return fmt.Errorf(msg)
	}

	return nil
}

// AccessRead 检查变量是否可以合法读取访问
func (bc *BorrowChecker) AccessRead(varName string, pos token.Position) error {
	state, exists := bc.states[varName]
	if !exists || !bc.isResource[varName] {
		return nil
	}

	if state == StateMoved {
		prev := bc.moveHistory[varName]
		targetInfo := ""
		if prev != nil {
			targetInfo = fmt.Sprintf(" (所有权已于第 %d 行转移至 `%s`)", prev.MovedPos.Line, prev.TargetName)
		}
		msg := fmt.Sprintf("%s: [E0382] 使用已移动的所有权变量 (borrow of moved value): 变量 `%s` 的资源已被转移%s，禁止二次访问！\n  --> 提示: 可通过借用 `&%s` 或避免过早转移来保持所有权可用。",
			pos, varName, targetInfo, varName)
		bc.Errors = append(bc.Errors, msg)
		return fmt.Errorf(msg)
	}

	return nil
}

// Drop 显式释放并销毁所有权 (如 drop(x) 或 free(x))
func (bc *BorrowChecker) Drop(varName string, pos token.Position) error {
	state, exists := bc.states[varName]
	if !exists || !bc.isResource[varName] {
		return nil
	}

	if state == StateMoved {
		prev := bc.moveHistory[varName]
		targetInfo := "已被释放或转移"
		if prev != nil {
			targetInfo = fmt.Sprintf("已在第 %d 行转移至 `%s`", prev.MovedPos.Line, prev.TargetName)
		}
		msg := fmt.Sprintf("%s: [E0382] 非法重复释放或操作已失效的值: 变量 `%s` %s，禁止再次操作！",
			pos, varName, targetInfo)
		bc.Errors = append(bc.Errors, msg)
		return fmt.Errorf(msg)
	}

	bc.states[varName] = StateMoved
	bc.moveHistory[varName] = &MoveRecord{
		VarName:    varName,
		TargetName: "<dropped>",
		MovedPos:   pos,
	}
	return nil
}

// IsResource 检查变量是否为受所有权管理跟踪的资源
func (bc *BorrowChecker) IsResource(varName string) bool {
	return bc.isResource[varName]
}

// IsMoved 检查变量所有权是否已移走
func (bc *BorrowChecker) IsMoved(varName string) bool {
	return bc.states[varName] == StateMoved
}

// State 获取变量当前所有权状态
func (bc *BorrowChecker) State(varName string) OwnershipState {
	if s, ok := bc.states[varName]; ok {
		return s
	}
	return StateOwned
}
