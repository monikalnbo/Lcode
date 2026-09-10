package mod

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"lcode/pkg/ast"
	"lcode/pkg/lexer"
	"lcode/pkg/parser"
)

// Module 表示一个已加载并解析的模块
type Module struct {
	Name     string       // 模块名，如 "math"
	Path     string       // 规范化文件路径
	AST      *ast.Program // 模块的抽象语法树
	Exports  map[string]ast.Decl
}

// Resolver 模块与依赖解析加载器
type Resolver struct {
	baseDir     string
	stdDir      string
	modules     map[string]*Module
	importStack []string
	Errors      []string
}

// NewResolver 创建解析器
func NewResolver(baseDir string, stdDir string) *Resolver {
	if stdDir == "" {
		stdDir = filepath.Join(baseDir, "std")
	}
	return &Resolver{
		baseDir:     baseDir,
		stdDir:      stdDir,
		modules:     make(map[string]*Module),
		importStack: make([]string, 0),
		Errors:      make([]string, 0),
	}
}

// ResolveAll 从根程序出发，递归解析加载所有 import 依赖并合并至主 AST
func (r *Resolver) ResolveAll(rootFile string, rootProg *ast.Program) (*ast.Program, error) {
	mergedProg := &ast.Program{
		Imports: rootProg.Imports,
		Decls:   make([]ast.Decl, 0),
		Stmts:   make([]ast.Stmt, 0),
	}

	// 记录根文件在加载栈中
	absRoot, _ := filepath.Abs(rootFile)
	r.importStack = append(r.importStack, absRoot)

	// 递归加载所有依赖模块
	for _, imp := range rootProg.Imports {
		mod, err := r.loadModule(filepath.Dir(absRoot), imp)
		if err != nil {
			r.Errors = append(r.Errors, err.Error())
			return nil, err
		}
		if mod != nil {
			// 将模块的顶层声明注入到合并程序中
			mergedProg.Decls = append(mergedProg.Decls, mod.AST.Decls...)
		}
	}

	// 加入根程序自身的声明与语句
	mergedProg.Decls = append(mergedProg.Decls, rootProg.Decls...)
	mergedProg.Stmts = append(mergedProg.Stmts, rootProg.Stmts...)

	r.importStack = r.importStack[:len(r.importStack)-1]
	return mergedProg, nil
}

// loadModule 解析单个 import 声明并递归加载
func (r *Resolver) loadModule(currentDir string, imp *ast.ImportDecl) (*Module, error) {
	filePath, err := r.resolveFilePath(currentDir, imp.Path)
	if err != nil {
		return nil, fmt.Errorf("%s: 模块加载失败: %v", imp.Pos(), err)
	}

	absPath, _ := filepath.Abs(filePath)

	// 1. 循环依赖检测 (Cycle Detection)
	for _, stacked := range r.importStack {
		if stacked == absPath {
			return nil, fmt.Errorf("%s: 检测到模块循环依赖: %s -> %s", imp.Pos(), strings.Join(r.importStack, " -> "), absPath)
		}
	}

	// 2. 缓存检查 (如果已加载过，直接复用)
	if mod, ok := r.modules[absPath]; ok {
		return mod, nil
	}

	// 3. 读取并解析模块文件
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("%s: 无法读取模块文件 %s: %v", imp.Pos(), absPath, err)
	}

	l := lexer.New(filepath.Base(absPath), string(content))
	p := parser.New(l)
	prog := p.ParseProgram()

	if len(l.Errors) > 0 {
		return nil, fmt.Errorf("模块 %s 词法错误: %v", absPath, l.Errors[0])
	}
	if len(p.Errors()) > 0 {
		return nil, fmt.Errorf("模块 %s 语法错误: %v", absPath, p.Errors()[0])
	}

	// 4. 递归加载该模块自身的 import
	r.importStack = append(r.importStack, absPath)
	for _, subImp := range prog.Imports {
		subMod, err := r.loadModule(filepath.Dir(absPath), subImp)
		if err != nil {
			return nil, err
		}
		prog.Decls = append(subMod.AST.Decls, prog.Decls...)
	}
	r.importStack = r.importStack[:len(r.importStack)-1]

	// 5. 提取导出符号
	modName := filepath.Base(imp.Path)
	modName = strings.TrimSuffix(modName, ".lc")
	if imp.Alias != "" {
		modName = imp.Alias
	}

	mod := &Module{
		Name:    modName,
		Path:    absPath,
		AST:     prog,
		Exports: make(map[string]ast.Decl),
	}

	for _, decl := range prog.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			mod.Exports[fn.Name.Value] = fn
		}
	}

	r.modules[absPath] = mod
	return mod, nil
}

// resolveFilePath 计算导入模块的实际磁盘路径
func (r *Resolver) resolveFilePath(currentDir string, importPath string) (string, error) {
	// 如果以 std/ 开头，从标准库目录查找
	if strings.HasPrefix(importPath, "std/") {
		trimmed := strings.TrimPrefix(importPath, "std/")
		cand1 := filepath.Join(r.stdDir, trimmed)
		cand2 := filepath.Join(r.stdDir, trimmed+".lc")
		if fileExists(cand2) {
			return cand2, nil
		}
		if fileExists(cand1) {
			return cand1, nil
		}
	}

	// 本地相对路径查找
	cand1 := filepath.Join(currentDir, importPath)
	cand2 := filepath.Join(currentDir, importPath+".lc")
	if fileExists(cand2) {
		return cand2, nil
	}
	if fileExists(cand1) {
		return cand1, nil
	}

	// 项目根目录查找
	cand3 := filepath.Join(r.baseDir, importPath)
	cand4 := filepath.Join(r.baseDir, importPath+".lc")
	if fileExists(cand4) {
		return cand4, nil
	}
	if fileExists(cand3) {
		return cand3, nil
	}

	return "", fmt.Errorf("未找到模块 %q (搜索路径包括: %s, %s, %s)", importPath, currentDir, r.stdDir, r.baseDir)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
