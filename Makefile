.PHONY: build run test lint fmt clean help

# show help
help:
	@echo "downboy build targets:"
	@echo ""
	@echo "  make build        - build application for current OS"
	@echo "  make run          - build and run application"
	@echo "  make test         - run all unit tests"
	@echo "  make lint         - run go vet static checks"
	@echo "  make fmt          - format code with gofmt"
	@echo "  make clean        - remove build artifacts"
	@echo ""

# build for linux
build:
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

# run static analysis
lint:
	go vet ./...
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
