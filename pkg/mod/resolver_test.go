package mod

import (
	"os"
	"path/filepath"
	"testing"

	"lcode/pkg/lexer"
	"lcode/pkg/parser"
)

func TestResolverStdImport(t *testing.T) {
	// 临时根目录与 std 目录
	tmpDir, err := os.MkdirTemp("", "lcode_mod_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	stdDir := filepath.Join(tmpDir, "std")
	if err := os.MkdirAll(stdDir, 0755); err != nil {
		t.Fatal(err)
	}

	mathMod := `
fn helper_sq(x: int) -> int {
    return x * x;
}
`
	if err := os.WriteFile(filepath.Join(stdDir, "math.lc"), []byte(mathMod), 0644); err != nil {
		t.Fatal(err)
	}

	mainCode := `
import "std/math";

fn main() -> void {
    let res = helper_sq(5);
    println("res:", res);
}
`
	mainFile := filepath.Join(tmpDir, "main.lc")
	if err := os.WriteFile(mainFile, []byte(mainCode), 0644); err != nil {
		t.Fatal(err)
	}

	l := lexer.New("main.lc", mainCode)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}

	resolver := NewResolver(tmpDir, stdDir)
	merged, err := resolver.ResolveAll(mainFile, prog)
	if err != nil {
		t.Fatalf("ResolveAll error: %v", err)
	}

	if len(merged.Decls) < 2 {
		t.Fatalf("Expected at least 2 declarations after merge, got %d", len(merged.Decls))
	}
}
