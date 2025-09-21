package analyzer

import (
	"strings"
	"testing"

	"github.com/perbu/vclparser/pkg/ast"
	"github.com/perbu/vclparser/pkg/metadata"
)

func TestVersionValidatorExtractVCLVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected int
	}{
		{"VCL 4.0", "4.0", 40},
		{"VCL 4.1", "4.1", 41},
		{"Invalid format", "4", 0},
		{"Invalid major", "x.0", 0},
		{"Invalid minor", "4.x", 0},
	}

	loader := metadata.New()
	validator := NewVersionValidator(loader)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program := &ast.Program{}
			if tt.version != "" && !strings.Contains(tt.version, "x") && len(strings.Split(tt.version, ".")) == 2 {
				program.VCLVersion = &ast.VCLVersionDecl{Version: tt.version}
			} else if tt.version != "" {
				program.VCLVersion = &ast.VCLVersionDecl{Version: tt.version}
			}

			result := validator.extractVCLVersion(program)
			if result != tt.expected {
				t.Errorf("Expected version %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestVersionValidatorValidateVariableVersions(t *testing.T) {
	loader := metadata.New()

	tests := []struct {
		name          string
		vclVersion    string
		variableName  string
		expectError   bool
		errorContains string
	}{
		{
			name:          "VCL 4.0 with 4.1 variable",
			vclVersion:    "4.0",
			variableName:  "local.endpoint", // available from VCL 4.1+
			expectError:   true,
			errorContains: "requires VCL version 4.1",
		},
		{
			name:         "VCL 4.1 with 4.1 variable",
			vclVersion:   "4.1",
			variableName: "local.endpoint", // available from VCL 4.1+
			expectError:  false,
		},
		{
			name:          "VCL 4.1 with deprecated variable",
			vclVersion:    "4.1",
			variableName:  "req.esi", // deprecated after VCL 4.0
			expectError:   true,
			errorContains: "not available in VCL version 4.1",
		},
		{
			name:         "VCL 4.0 with compatible variable",
			vclVersion:   "4.0",
			variableName: "req.method", // available in all versions
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewVersionValidator(loader)

			// Create a program with version and variable access
			program := &ast.Program{
				VCLVersion: &ast.VCLVersionDecl{Version: tt.vclVersion},
				Declarations: []ast.Declaration{
					&ast.SubDecl{
						Name: "vcl_recv",
						Body: &ast.BlockStatement{
							Statements: []ast.Statement{
								&ast.SetStatement{
									Variable: createVariableExpression(tt.variableName),
									Value:    &ast.StringLiteral{Value: "test"},
								},
							},
						},
					},
				},
			}

			errors := validator.Validate(program)

			if tt.expectError {
				if len(errors) == 0 {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(errors[0], tt.errorContains) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorContains, errors[0])
				}
			} else {
				if len(errors) > 0 {
					t.Errorf("Expected no errors but got: %v", errors)
				}
			}
		})
	}
}

func TestVersionValidatorNormalizeDynamicVariableName(t *testing.T) {
	loader := metadata.New()
	validator := NewVersionValidator(loader)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"HTTP header", "req.http.host", "req.http."},
		{"Backend HTTP header", "bereq.http.authorization", "bereq.http."},
		{"Storage variable", "storage.malloc.free_space", "storage.malloc.*"},
		{"Regular variable", "req.method", ""},
		{"Unknown pattern", "custom.variable", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.normalizeDynamicVariableName(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// Helper function to create variable expressions for testing
func createVariableExpression(varName string) ast.Expression {
	parts := strings.Split(varName, ".")
	if len(parts) == 0 {
		return &ast.Identifier{Name: varName}
	}
	if len(parts) == 1 {
		return &ast.Identifier{Name: parts[0]}
	}

	var expr ast.Expression = &ast.Identifier{Name: parts[0]}
	for i := 1; i < len(parts); i++ {
		expr = &ast.MemberExpression{
			Object:   expr,
			Property: &ast.Identifier{Name: parts[i]},
		}
	}
	return expr
}
