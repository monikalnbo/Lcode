package sema

import (
	"fmt"

	"lcode/pkg/token"
)

// SymbolKind 符号分类
type SymbolKind string

const (
	SymVar    SymbolKind = "VAR"
	SymMutVar SymbolKind = "MUT"
	SymConst  SymbolKind = "LET"
	SymFunc   SymbolKind = "FN"
	SymParam  SymbolKind = "PARAM"
	SymStruct SymbolKind = "STRUCT"
)

// Symbol 符号结构体
type Symbol struct {
	Name  string
	Type  Type
	Kind  SymbolKind
	IsMut bool
	Pos   token.Position
}

// Scope 作用域树节点
type Scope struct {
	parent  *Scope
	symbols map[string]*Symbol
}

// NewScope 创建新作用域
func NewScope(parent *Scope) *Scope {
	return &Scope{
		parent:  parent,
		symbols: make(map[string]*Symbol),
	}
}

// Define 在当前作用域内定义新符号（如果在同级已存在则报错）
func (s *Scope) Define(sym *Symbol) error {
	if old, exists := s.symbols[sym.Name]; exists {
		return fmt.Errorf("%s: 标识符 %q 重复定义，先前定义于 %s", sym.Pos, sym.Name, old.Pos)
	}
	s.symbols[sym.Name] = sym
	return nil
}

// Resolve 递归向上查找符号
func (s *Scope) Resolve(name string) (*Symbol, bool) {
	if sym, ok := s.symbols[name]; ok {
		return sym, true
	}
	if s.parent != nil {
		return s.parent.Resolve(name)
	}
	return nil, false
}

// Parent 返回父级作用域
func (s *Scope) Parent() *Scope {
	return s.parent
}

// Symbols 返回当前作用域的所有符号列表
func (s *Scope) Symbols() []*Symbol {
	list := make([]*Symbol, 0, len(s.symbols))
	for _, sym := range s.symbols {
		list = append(list, sym)
	}
	return list
}
