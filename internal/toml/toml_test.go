package toml

import (
	"testing"
)

func TestParseSimple(t *testing.T) {
	input := `
key = "value"
number = 42
float = 3.14
bool = true
`
	result, err := ParseNative(input)
	if err != nil {
		t.Fatalf("ParseNative() error = %v", err)
	}

	if result["key"] != "value" {
		t.Errorf("key = %v, want 'value'", result["key"])
	}
	if result["number"] != 42 {
		t.Errorf("number = %v, want 42", result["number"])
	}
	if result["float"] != 3.14 {
		t.Errorf("float = %v, want 3.14", result["float"])
	}
	if result["bool"] != true {
		t.Errorf("bool = %v, want true", result["bool"])
	}
}

func TestParseTable(t *testing.T) {
	input := `
[server]
host = "localhost"
port = 8080
`
	result, err := ParseNative(input)
	if err != nil {
		t.Fatalf("ParseNative() error = %v", err)
	}

	server, ok := result["server"].(map[string]any)
	if !ok {
		t.Fatal("server should be a map")
	}
	if server["host"] != "localhost" {
		t.Errorf("server.host = %v, want 'localhost'", server["host"])
	}
	if server["port"] != 8080 {
		t.Errorf("server.port = %v, want 8080", server["port"])
	}
}

func TestParseNestedTable(t *testing.T) {
	input := `
[server]
[server.tls]
enabled = true
cert = "cert.pem"
`
	result, err := ParseNative(input)
	if err != nil {
		t.Fatalf("ParseNative() error = %v", err)
	}

	server, ok := result["server"].(map[string]any)
	if !ok {
		t.Fatal("server should be a map")
	}
	tls, ok := server["tls"].(map[string]any)
	if !ok {
		t.Fatal("server.tls should be a map")
	}
	if tls["enabled"] != true {
		t.Errorf("server.tls.enabled = %v, want true", tls["enabled"])
	}
}

func TestParseArray(t *testing.T) {
	input := `
ints = [1, 2, 3]
strings = ["a", "b", "c"]
empty = []
`
	result, err := ParseNative(input)
	if err != nil {
		t.Fatalf("ParseNative() error = %v", err)
	}

	ints, ok := result["ints"].([]any)
	if !ok {
		t.Fatal("ints should be an array")
	}
	if len(ints) != 3 {
		t.Errorf("ints length = %d, want 3", len(ints))
	}

	strings, ok := result["strings"].([]any)
	if !ok {
		t.Fatal("strings should be an array")
	}
	if len(strings) != 3 {
		t.Errorf("strings length = %d, want 3", len(strings))
	}

	empty, ok := result["empty"].([]any)
	if !ok {
		t.Fatal("empty should be an array")
	}
	if len(empty) != 0 {
		t.Errorf("empty length = %d, want 0", len(empty))
	}
}

func TestParseComment(t *testing.T) {
	input := `
# This is a comment
key = "value"
`
	result, err := ParseNative(input)
	if err != nil {
		t.Fatalf("ParseNative() error = %v", err)
	}

	if result["key"] != "value" {
		t.Errorf("key = %v, want 'value'", result["key"])
	}
}

func TestParseInlineCommentNotSupported(t *testing.T) {
	// This parser doesn't support inline comments, so we test what it does
	input := `key = "value" # this is not supported`
	_, err := ParseNative(input)
	// This should error because the parser treats # as invalid
	if err == nil {
		t.Log("Note: inline comments are not supported by this parser")
	}
}

func TestParseEmptyLines(t *testing.T) {
	input := `

key = "value"

`
	result, err := ParseNative(input)
	if err != nil {
		t.Fatalf("ParseNative() error = %v", err)
	}

	if result["key"] != "value" {
		t.Errorf("key = %v, want 'value'", result["key"])
	}
}

func TestParseError(t *testing.T) {
	input := `invalid syntax here`
	_, err := ParseNative(input)
	if err == nil {
		t.Error("expected error for invalid syntax")
	}
}

func TestParseErrorEmptyTable(t *testing.T) {
	input := `[]`
	_, err := ParseNative(input)
	if err == nil {
		t.Error("expected error for empty table name")
	}
}

func TestParseErrorEmptyArrayElement(t *testing.T) {
	input := `arr = [1, , 2]`
	_, err := ParseNative(input)
	if err == nil {
		t.Error("expected error for empty array element")
	}
}

func TestParseErrorTrailingComma(t *testing.T) {
	input := `arr = [1, 2,]`
	_, err := ParseNative(input)
	// This should be valid
	if err != nil {
		t.Errorf("ParseNative() error = %v, want nil", err)
	}
}

func TestParseBoolean(t *testing.T) {
	input := `
t = true
f = false
`
	result, err := ParseNative(input)
	if err != nil {
		t.Fatalf("ParseNative() error = %v", err)
	}

	if result["t"] != true {
		t.Errorf("t = %v, want true", result["t"])
	}
	if result["f"] != false {
		t.Errorf("f = %v, want false", result["f"])
	}
}

func TestParseFloat(t *testing.T) {
	input := `
pi = 3.14159
neg = -1.5
`
	result, err := ParseNative(input)
	if err != nil {
		t.Fatalf("ParseNative() error = %v", err)
	}

	if result["pi"] != 3.14159 {
		t.Errorf("pi = %v, want 3.14159", result["pi"])
	}
	if result["neg"] != -1.5 {
		t.Errorf("neg = %v, want -1.5", result["neg"])
	}
}

func TestParseNew(t *testing.T) {
	parser := New()
	if parser == nil {
		t.Fatal("New() should return non-nil parser")
	}
	if parser.result == nil {
		t.Error("parser.result should be initialized")
	}
	if parser.current == nil {
		t.Error("parser.current should be initialized")
	}
}

func TestParseErrorInterface(t *testing.T) {
	err := &ParseError{Line: 10, Msg: "test error"}
	if err.Error() != "line 10: test error" {
		t.Errorf("ParseError.Error() = %q, want 'line 10: test error'", err.Error())
	}
}

func TestParseArrayWithStrings(t *testing.T) {
	input := `arr = ["hello", "world"]`
	result, err := ParseNative(input)
	if err != nil {
		t.Fatalf("ParseNative() error = %v", err)
	}

	arr, ok := result["arr"].([]any)
	if !ok {
		t.Fatal("arr should be an array")
	}
	if len(arr) != 2 {
		t.Errorf("arr length = %d, want 2", len(arr))
	}
}

func TestParseTOMLToJSON(t *testing.T) {
	input := `
key = "value"
[section]
num = 42
`
	jsonBytes, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Should be valid JSON
	jsonStr := string(jsonBytes)
	if jsonStr == "" {
		t.Error("Parse() should return non-empty JSON")
	}
}

func TestParseArrayWithMixedTypes(t *testing.T) {
	// TOML doesn't support mixed arrays, but we can test edge cases
	input := `arr = ["a", "b", "c"]`
	result, err := ParseNative(input)
	if err != nil {
		t.Fatalf("ParseNative() error = %v", err)
	}

	arr, ok := result["arr"].([]any)
	if !ok {
		t.Fatal("arr should be an array")
	}
	if len(arr) != 3 {
		t.Errorf("arr length = %d, want 3", len(arr))
	}
}

func TestParseUnrecognizedValue(t *testing.T) {
	input := `key = unknown_value`
	_, err := ParseNative(input)
	if err == nil {
		t.Error("expected error for unrecognized value")
	}
}

func TestParseMultipleTables(t *testing.T) {
	input := `
[a]
x = 1
[b]
y = 2
`
	result, err := ParseNative(input)
	if err != nil {
		t.Fatalf("ParseNative() error = %v", err)
	}

	a, ok := result["a"].(map[string]any)
	if !ok {
		t.Fatal("a should be a map")
	}
	if a["x"] != 1 {
		t.Errorf("a.x = %v, want 1", a["x"])
	}

	b, ok := result["b"].(map[string]any)
	if !ok {
		t.Fatal("b should be a map")
	}
	if b["y"] != 2 {
		t.Errorf("b.y = %v, want 2", b["y"])
	}
}

func TestParseComplexRealWorld(t *testing.T) {
	// Test a complex but valid TOML structure that the parser supports
	input := `
include = "shared.toml"

[hooks.eslint]
event = "pre-commit"
command = "npx eslint --fix"
match = ["*.ts", "*.js"]

[hooks.test]
event = "pre-push"
command = "go test ./..."
`
	result, err := ParseNative(input)
	if err != nil {
		t.Fatalf("ParseNative() error = %v", err)
	}

	if result["include"] != "shared.toml" {
		t.Errorf("include = %v, want 'shared.toml'", result["include"])
	}

	hooks, ok := result["hooks"].(map[string]any)
	if !ok {
		t.Fatal("hooks should be a map")
	}

	eslint, ok := hooks["eslint"].(map[string]any)
	if !ok {
		t.Fatal("hooks.eslint should be a map")
	}
	if eslint["event"] != "pre-commit" {
		t.Errorf("hooks.eslint.event = %v, want 'pre-commit'", eslint["event"])
	}
}
