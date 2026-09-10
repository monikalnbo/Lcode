package sema

import (
	"fmt"
	"strings"
)

// Type 表示语义分析阶段的类型接口
type Type interface {
	Name() string
	Equals(other Type) bool
}

// BasicType 基本类型
type BasicType struct {
	TypeName string
}

func (b *BasicType) Name() string {
	return b.TypeName
}

func (b *BasicType) Equals(other Type) bool {
	if other == nil {
		return false
	}
	return b.TypeName == other.Name()
}

// 预定义基本类型常量
var (
	TypeInt     = &BasicType{TypeName: "int"}
	TypeFloat   = &BasicType{TypeName: "float"}
	TypeBool    = &BasicType{TypeName: "bool"}
	TypeString  = &BasicType{TypeName: "string"}
	TypeVoid    = &BasicType{TypeName: "void"}
	TypeUnknown = &BasicType{TypeName: "unknown"}
)

// FuncType 函数类型表示
type FuncType struct {
	ParamTypes []Type
	ReturnType Type
}

func (f *FuncType) Name() string {
	params := make([]string, 0, len(f.ParamTypes))
	for _, p := range f.ParamTypes {
		params = append(params, p.Name())
	}
	ret := "void"
	if f.ReturnType != nil {
		ret = f.ReturnType.Name()
	}
	return fmt.Sprintf("func(%s) -> %s", strings.Join(params, ", "), ret)
}

func (f *FuncType) Equals(other Type) bool {
	otherFunc, ok := other.(*FuncType)
	if !ok {
		return false
	}
	if len(f.ParamTypes) != len(otherFunc.ParamTypes) {
		return false
	}
	for i, pt := range f.ParamTypes {
		if !pt.Equals(otherFunc.ParamTypes[i]) {
			return false
		}
	}
	if f.ReturnType == nil && otherFunc.ReturnType == nil {
		return true
	}
	if f.ReturnType == nil || otherFunc.ReturnType == nil {
		return false
	}
	return f.ReturnType.Equals(otherFunc.ReturnType)
}

// LookupBasicType 根据类型名字符串查找基本类型
func LookupBasicType(name string) Type {
	switch name {
	case "int":
		return TypeInt
	case "float":
		return TypeFloat
	case "bool":
		return TypeBool
	case "string":
		return TypeString
	case "void":
		return TypeVoid
	default:
		return nil
	}
}
