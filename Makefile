.PHONY: help test build build-linux run clean fmt lint

# show help
help:
	@echo "downboy build targets:"
	@echo ""
	@echo "  make build        - build application for current OS"
	@echo "  make build-linux  - build application for Linux"
	@echo "  make run          - build and run application"
	@echo "  make test         - run all unit tests"
	@echo "  make test-v       - run tests with verbose output"
	@echo "  make test-race    - run tests with race detector"
	@echo "  make test-cover   - run tests with coverage report"
	@echo "  make lint         - run go vet static checks"
	@echo "  make fmt          - format code with gofmt"
	@echo "  make clean        - remove build artifacts"
	@echo ""

# build for current OS (darwin/linux/windows)
build:
	@echo "building downboy for $(shell go env GOOS)..."
	@mkdir -p dist
	go build -o dist/downboy cmd/downboy/main.go
	@echo "✓ binary saved to dist/downboy"

# build for linux (used in CI)
build-linux:
	@echo "building downboy for Linux..."
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -o dist/downboy cmd/downboy/main.go
	@echo "✓ binary saved to dist/downboy"

# build and run
run: build
	@echo "running downboy..."
	./dist/downboy

# run all tests
test:
	go test ./...

# run tests with verbose output
test-v:
	go test -v ./...

# run tests with race detector
test-race:
	go test -race ./...

# run tests with coverage report
test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✓ coverage report saved to coverage.html"

# run static analysis
lint:
	go vet ./..
	@echo "✓ lint complete"


# format code
fmt:
	go fmt ./...
	@echo "✓ code formatted"

# remove build artifacts
clean:
	rm -rf dist/
	rm -f coverage.out coverage.html
	@echo "✓ clean complete"

# development target: test, build, and report (useful before commit)
check: fmt test build
	@echo "✓ all checks passed"