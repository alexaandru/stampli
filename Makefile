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
	@go test . -covermode=atomic -coverprofile=unit.cov $(OPTS)
	@go build . && ./stampli -quiet

badge:
	@go build .
	@./stampli

sample-badges:
	@go build .
	@./stampli -coverage 85 -output testdata/badge-excellent.svg
	@./stampli -coverage 70 -output testdata/badge-fair.svg
	@./stampli -coverage 50 -output testdata/badge-good.svg
	@./stampli -coverage 0 -output testdata/badge-poor.svg

clean:
	rm -f stampli *.out *.cov
