all: fmt lint test

fmt:
	@find -name "*.go"|GOFLAGS=-tags=testaws xargs go tool -modfile=tools/go.mod gofumpt -extra -w
	@find -name "*.go"|GOFLAGS=-tags=testaws xargs go tool -modfile=tools/go.mod goimports -w

lint:
	@printf "Linter: "
	@go tool -modfile=tools/go.mod golangci-lint config verify
	@go tool -modfile=tools/go.mod golangci-lint run
	@#go tool -modfile tools/go.mod modernize -test ./...


test:
	@go test . -covermode=atomic -coverprofile=unit.cov
	@go build . && ./stampli -quiet

badge:
	@go build .
	@./stampli

clean:
	rm -f stampli coverage.out unit.cov coverage-badge.svg
