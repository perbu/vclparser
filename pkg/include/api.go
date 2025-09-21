package include

import "github.com/perbu/vclparser/pkg/ast"

// ResolveFile is a convenience function that parses a VCL file and resolves all includes
// using default settings. For more control, use NewResolver with options.
func ResolveFile(filename string) (*ast.Program, error) {
	resolver := NewResolver()
	return resolver.ResolveFile(filename)
}

// ResolveFileWithBasePath parses a VCL file and resolves includes relative to the given base path
func ResolveFileWithBasePath(filename, basePath string) (*ast.Program, error) {
	resolver := NewResolver(WithBasePath(basePath))
	return resolver.ResolveFile(filename)
}

// ResolveProgram takes an already-parsed AST program and resolves any include statements
func ResolveProgram(program *ast.Program) (*ast.Program, error) {
	resolver := NewResolver()
	return resolver.Resolve(program)
}

// ResolveProgramWithBasePath resolves includes in a program using the given base path
func ResolveProgramWithBasePath(program *ast.Program, basePath string) (*ast.Program, error) {
	resolver := NewResolver(WithBasePath(basePath))
	return resolver.Resolve(program)
}
