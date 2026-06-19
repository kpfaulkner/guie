
.PHONY: coverage
coverage:
	go test -v -coverprofile cover.out .\geom .\internal\ebiten .\render .\theme .\ui
	go tool cover -html cover.out -o cover.html

.PHONY: test
test:
	go test -v ./...

.PHONY: lint
lint:
	golangci-lint run