help:
	@echo "You can perform the following:"
	@echo ""
	@echo "  check         Format, lint, vet, and test Go code"
	@echo "  test          Test Go code"
	@echo "  cover         Show test coverage"

# Format, lint, vet, and test the Go code
check:
	@echo 'Formatting, linting, vetting, and testing Go code'
	go fmt ./...
	golint ./...
	go vet ./...
	go test ./... -cover

# Test the Go code
test:
	@echo 'Test Go code'
	go test ./... -cover

cover:
	@echo 'Test coverage in html'
	go test -coverprofile=coverage.out
	go tool cover -html=coverage.out
