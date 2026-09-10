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
	TypeAny     = &BasicType{TypeName: "any"}
	TypeArray   = &ArrayType{ElementType: TypeAny}
	TypeUnknown = &BasicType{TypeName: "unknown"}
)

// ArrayType 数组类型表示
type ArrayType struct {
	ElementType Type
}

func (a *ArrayType) Name() string {
	if a.ElementType == nil || a.ElementType.Name() == "any" {
		return "array"
	}
	return "[]" + a.ElementType.Name()
}

func (a *ArrayType) Equals(other Type) bool {
	if other == nil {
		return false
	}
	if other.Name() == "array" || a.Name() == "array" {
		return true
	}
	otherArr, ok := other.(*ArrayType)
	if !ok {
		return false
	}
	return a.ElementType.Equals(otherArr.ElementType)
}

// StructType 结构体类型表示
type StructType struct {
	StructName string
	Fields     map[string]Type
}

func (s *StructType) Name() string {
	return s.StructName
}

func (s *StructType) Equals(other Type) bool {
	if other == nil {
		return false
	}
	return s.StructName == other.Name()
}

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
	case "array":
		return TypeArray
	case "any":
		return TypeAny
	default:
		return nil
	}
}
