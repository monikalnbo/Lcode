package cpp

import (
	"strings"
	"testing"

	"lcode/pkg/lexer"
	"lcode/pkg/parser"
)

func TestCppCodegen(t *testing.T) {
	input := `
fn add(a: int, b: int) -> int {
    return a + b;
}

fn main() -> void {
    let x: int = 10;
    mut y: int = 20;
    y = y + add(x, 5);
    println("Result is:", y);
}
`
	l := lexer.New("test.lc", input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}

	gen := NewGenerator()
	cppCode, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Codegen error: %v", err)
	}

	if !strings.Contains(cppCode, "#include \"lcode_cpp.hpp\"") {
		t.Errorf("Expected #include lcode_cpp.hpp")
	}
	if !strings.Contains(cppCode, "int64_t add(int64_t a, int64_t b)") {
		t.Errorf("Expected add function signature in C++")
	}
	if !strings.Contains(cppCode, "const int64_t x = 10LL;") {
		t.Errorf("Expected const int64_t x")
	}
	if !strings.Contains(cppCode, "int64_t y = 20LL;") {
		t.Errorf("Expected mutable int64_t y")
	}
	if !strings.Contains(cppCode, "lcode::io::println") {
		t.Errorf("Expected lcode::io::println")
	}
}
