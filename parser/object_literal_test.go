package parser

import (
	"strconv"
	"testing"

	"github.com/varnish/vclparser/ast"
	"github.com/varnish/vclparser/lexer"
)

// TestInlineProbeObjectLiteral tests parsing of inline probe definitions within backend declarations
// This test will fail until object literal parsing is properly implemented for backend properties
func TestInlineProbeObjectLiteral(t *testing.T) {
	input := `vcl 4.1;

backend web {
    .host = "example.com";
    .probe = {
        .url = "/health";
        .interval = 30s;
        .timeout = 5s;
        .window = 5;
        .threshold = 3;
    };
}`

	l := lexer.New(input, "test.vcl")
	p := New(l, input, "test.vcl")
	program := p.ParseProgram()

	// This will currently fail, but should pass once object literal parsing is implemented
	checkParserErrors(t, p)

	if len(program.Declarations) != 1 {
		t.Fatalf("program.Declarations does not contain 1 declaration. got=%d",
			len(program.Declarations))
	}

	decl, ok := program.Declarations[0].(*ast.BackendDecl)
	if !ok {
		t.Fatalf("program.Declarations[0] is not *ast.BackendDecl. got=%T",
			program.Declarations[0])
	}

	if decl.Name != "web" {
		t.Errorf("decl.Name = %q, want %q", decl.Name, "web")
	}

	if len(decl.Properties) != 2 {
		t.Fatalf("backend does not contain 2 properties. got=%d", len(decl.Properties))
	}

	// Check host property (simple string)
	hostProp := decl.Properties[0]
	if hostProp.Name != "host" {
		t.Errorf("property[0].Name = %q, want %q", hostProp.Name, "host")
	}

	hostValue, ok := hostProp.Value.(*ast.StringLiteral)
	if !ok {
		t.Fatalf("property[0].Value is not *ast.StringLiteral. got=%T", hostProp.Value)
	}

	if hostValue.Value != "example.com" {
		t.Errorf("property[0].Value = %q, want %q", hostValue.Value, "example.com")
	}

	// Check probe property (object literal)
	probeProp := decl.Properties[1]
	if probeProp.Name != "probe" {
		t.Errorf("property[1].Name = %q, want %q", probeProp.Name, "probe")
	}

	probeObj, ok := probeProp.Value.(*ast.ObjectExpression)
	if !ok {
		t.Fatalf("property[1].Value is not *ast.ObjectExpression. got=%T", probeProp.Value)
	}

	// Verify probe object has 5 properties
	if len(probeObj.Properties) != 5 {
		t.Fatalf("probe object does not contain 5 properties. got=%d", len(probeObj.Properties))
	}

	// Expected probe properties
	expectedProbeProps := []struct {
		key   string
		value string
		typ   string // "string", "time", or "int"
	}{
		{"url", "/health", "string"},
		{"interval", "30s", "time"},
		{"timeout", "5s", "time"},
		{"window", "5", "int"},
		{"threshold", "3", "int"},
	}

	for i, expected := range expectedProbeProps {
		prop := probeObj.Properties[i]

		// Check key (should be an identifier like url, interval, etc.)
		ident, ok := prop.Key.(*ast.Identifier)
		if !ok {
			t.Fatalf("probe property[%d].Key is not *ast.Identifier. got=%T", i, prop.Key)
		}

		if ident.Name != expected.key {
			t.Errorf("probe property[%d].Key = %q, want %q", i, ident.Name, expected.key)
		}

		// Check value based on type
		switch expected.typ {
		case "string":
			stringLit, ok := prop.Value.(*ast.StringLiteral)
			if !ok {
				t.Fatalf("probe property[%d].Value is not *ast.StringLiteral. got=%T", i, prop.Value)
			}
			if stringLit.Value != expected.value {
				t.Errorf("probe property[%d].Value = %q, want %q", i, stringLit.Value, expected.value)
			}
		case "time":
			timeLit, ok := prop.Value.(*ast.TimeExpression)
			if !ok {
				t.Fatalf("probe property[%d].Value is not *ast.TimeExpression. got=%T", i, prop.Value)
			}
			if timeLit.Value != expected.value {
				t.Errorf("probe property[%d].Value = %q, want %q", i, timeLit.Value, expected.value)
			}
		case "int":
			intLit, ok := prop.Value.(*ast.IntegerLiteral)
			if !ok {
				t.Fatalf("probe property[%d].Value is not *ast.IntegerLiteral. got=%T", i, prop.Value)
			}
			expectedInt, err := strconv.ParseInt(expected.value, 10, 64)
			if err != nil {
				t.Fatalf("failed to parse expected int value %q", expected.value)
			}
			if intLit.Value != expectedInt {
				t.Errorf("probe property[%d].Value = %d, want %d", i, intLit.Value, expectedInt)
			}
		}
	}
}

// TestMixedBackendProperties tests a backend with both simple properties and an inline probe
func TestMixedBackendProperties(t *testing.T) {
	input := `vcl 4.1;

backend api {
    .host = "api.example.com";
    .port = "8080";
    .probe = {
        .url = "/healthz";
        .interval = 10s;
    };
    .max_connections = 100;
}`

	l := lexer.New(input, "test.vcl")
	p := New(l, input, "test.vcl")
	program := p.ParseProgram()

	// This will currently fail, but should pass once object literal parsing is implemented
	checkParserErrors(t, p)

	if len(program.Declarations) != 1 {
		t.Fatalf("program.Declarations does not contain 1 declaration. got=%d",
			len(program.Declarations))
	}

	decl, ok := program.Declarations[0].(*ast.BackendDecl)
	if !ok {
		t.Fatalf("program.Declarations[0] is not *ast.BackendDecl. got=%T",
			program.Declarations[0])
	}

	if len(decl.Properties) != 4 {
		t.Fatalf("backend does not contain 4 properties. got=%d", len(decl.Properties))
	}

	// Verify property types
	expectedTypes := []string{"string", "string", "object", "int"}
	expectedNames := []string{"host", "port", "probe", "max_connections"}

	for i, expectedName := range expectedNames {
		prop := decl.Properties[i]
		if prop.Name != expectedName {
			t.Errorf("property[%d].Name = %q, want %q", i, prop.Name, expectedName)
		}

		switch expectedTypes[i] {
		case "string":
			if _, ok := prop.Value.(*ast.StringLiteral); !ok {
				t.Errorf("property[%d] (%s) is not string literal. got=%T", i, expectedName, prop.Value)
			}
		case "object":
			if _, ok := prop.Value.(*ast.ObjectExpression); !ok {
				t.Errorf("property[%d] (%s) is not object expression. got=%T", i, expectedName, prop.Value)
			}
		case "int":
			if _, ok := prop.Value.(*ast.IntegerLiteral); !ok {
				t.Errorf("property[%d] (%s) is not integer literal. got=%T", i, expectedName, prop.Value)
			}
		}
	}
}

// TestSimpleInlineProbe tests the most basic inline probe case
func TestSimpleInlineProbe(t *testing.T) {
	input := `vcl 4.1;

backend simple {
    .probe = {
        .url = "/";
    };
}`

	l := lexer.New(input, "test.vcl")
	p := New(l, input, "test.vcl")
	program := p.ParseProgram()

	// This will currently fail, but should pass once object literal parsing is implemented
	checkParserErrors(t, p)

	if len(program.Declarations) != 1 {
		t.Fatalf("program.Declarations does not contain 1 declaration. got=%d",
			len(program.Declarations))
	}

	decl, ok := program.Declarations[0].(*ast.BackendDecl)
	if !ok {
		t.Fatalf("program.Declarations[0] is not *ast.BackendDecl. got=%T",
			program.Declarations[0])
	}

	if len(decl.Properties) != 1 {
		t.Fatalf("backend does not contain 1 property. got=%d", len(decl.Properties))
	}

	probeProp := decl.Properties[0]
	if probeProp.Name != "probe" {
		t.Errorf("property.Name = %q, want %q", probeProp.Name, "probe")
	}

	probeObj, ok := probeProp.Value.(*ast.ObjectExpression)
	if !ok {
		t.Fatalf("property.Value is not *ast.ObjectExpression. got=%T", probeProp.Value)
	}

	if len(probeObj.Properties) != 1 {
		t.Fatalf("probe object does not contain 1 property. got=%d", len(probeObj.Properties))
	}
}
