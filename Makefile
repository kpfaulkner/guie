
# Run tests across every package and write a per-package coverage summary plus a
# browsable HTML report (cover.html).
.PHONY: coverage
coverage:
	go test -coverprofile cover.out ./...
	go tool cover -html cover.out -o cover.html

.PHONY: test
test:
	go test -v ./...

# Run just the headless harness package (guitest) tests.
.PHONY: guitest
guitest:
	go test -v ./guitest/...

.PHONY: lint
lint:
	golangci-lint run
