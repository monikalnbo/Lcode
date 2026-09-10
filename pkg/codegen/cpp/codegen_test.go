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

func TestBackendFeaturesCodegen(t *testing.T) {
	input := `
struct Point {
    x: int,
    y: int,
}

fn main() -> void {
    let p: Point = Point { x: 10, y: 20 };
    mut arr = [1, 2, 3];
    arr[0] = 100;
    let first = arr[0];

    for val in arr {
        if val > 50 {
            break;
        } else {
            continue;
        }
    }

    try {
        let bad = 10 / 0;
    } catch (e) {
        println("caught");
    }
}
`
	l := lexer.New("test_backend_cg.lc", input)
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

	expectedSnippets := []string{
		"struct Point {",
		"int64_t x;",
		"int64_t y;",
		"Point _s{};",
		"std::vector<int64_t>{1LL, 2LL, 3LL}",
		"arr[0LL] = 100LL;",
		"for (auto& val : arr) {",
		"break;",
		"continue;",
		"try {",
		"} catch (const std::exception& e) {",
	}

	for _, s := range expectedSnippets {
		if !strings.Contains(cppCode, s) {
			t.Errorf("Expected C++ codegen to contain %q, but got:\n%s", s, cppCode)
		}
	}
}

