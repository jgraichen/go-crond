#!/usr/bin/env make -f

.PHONY: clean
clean:
	git clean -Xfd .

.PHONY: build
build:
	go build -v

.PHONY: vendor
vendor:
	go mod tidy
	go mod vendor
	go mod verify

.PHONY: check
check: vendor lint test

.PHONY: test
test:
	go test ./...

.PHONY: lint
lint:
	golangci-lint run --verbose --print-resources-usage
