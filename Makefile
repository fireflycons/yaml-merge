GO=go
GOCOVER=$(GO) tool cover
GOTEST=$(GO) test


all: fmt lint build test

build:
	$(GO) build -v ./...

# See https://golangci-lint.run/
lint:
	golangci-lint run

fmt:
	gofmt -s -w -e .

test:
	$(GOTEST) -v -cover -timeout=120s -parallel=4 ./...

.PHONY: test/cover
test/cover:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCOVER) -func=coverage.out
	$(GOCOVER) -html=coverage.out

