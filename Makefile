BINARY_NAME=recall
BUILD_DIR=bin
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE=$(shell date -u '+%Y-%m-%d')
LDFLAGS=-ldflags "-X github.com/Om-Rohilla/recall/cmd.Version=$(VERSION) -X github.com/Om-Rohilla/recall/cmd.BuildDate=$(BUILD_DATE)"

.PHONY: build test clean install lint vet fmt

build:
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

test:
	go test ./... -v -count=1

clean:
	rm -rf $(BUILD_DIR)
	go clean -testcache

install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/ 2>/dev/null || cp $(BUILD_DIR)/$(BINARY_NAME) ~/go/bin/

vet:
	go vet ./...

fmt:
	gofmt -w .

lint: vet
	@which staticcheck > /dev/null 2>&1 || echo "install staticcheck: go install honnef.co/go/tools/cmd/staticcheck@latest"
	staticcheck ./... 2>/dev/null || true

run:
	go run . $(ARGS)
