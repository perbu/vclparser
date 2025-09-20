# Makefile for VCL Parser

.PHONY: lint vet nilaway golangci all clean test build

# Run all linting tools
lint: vet nilaway golangci

# Go vet - built-in static analysis
vet:
	go vet ./...

# nilaway - nil pointer analysis
nilaway:
	nilaway ./...

# golangci-lint - comprehensive linter
golangci:
	golangci-lint run

# Run all quality checks
all: lint test

# Clean build artifacts
clean:
	go clean ./...
	rm -f parse_vcl

# Run tests
test:
	go test ./...

# Build example parser
build:
	go build -o parse_vcl ./examples/parse_vcl.go