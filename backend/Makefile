APP=zxmail-api

.PHONY: tidy build test run check

tidy:
	go mod tidy

build:
	go build ./cmd/api

test:
	go test ./...

run:
	go run ./cmd/api

check: build test
