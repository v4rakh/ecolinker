GO ?= GO111MODULE=on CGO_ENABLED=0 go
GO_TEST ?= CGO_ENABLED=1 go
GOOS ?= $(shell $(GO) version | cut -d' ' -f4 | cut -d'/' -f1)
GOARCH ?= $(shell $(GO) version | cut -d' ' -f4 | cut -d'/' -f2)

CMD_GO_FILES ?= ./cmd/ecolinker/main.go

export GO111MODULE=on

BIN_DIR = $(shell pwd)/bin

clean:
	@rm -rf ${BIN_DIR}
	@$(GO) clean -testcache

dependencies:
	@$(GO) mod download

checkstyle:
	@$(GO) vet ./...

generate:
	@$(GO) generate ./...

test-unit:
	@$(GO_TEST) test -race -shuffle on ./...

run:
	@$(GO) run ${CMD_GO_FILES} server serve

audit:
	@$(GO) install github.com/securego/gosec/v2/cmd/gosec@latest
	@$$(go env GOPATH)/bin/gosec -quiet -sort -severity medium -confidence high ./...

build-local:
	@$(GO) build -tags prod -o ${BIN_DIR}/ecolinker-${GOOS}-${GOARCH} ${CMD_GO_FILES}

build: build-all

build-all: build-freebsd-amd64 build-freebsd-arm64 build-darwin-amd64 build-darwin-arm64 build-linux-amd64 build-linux-arm64 build-windows-amd64 build-windows-arm64

build-freebsd-amd64:
	@GOOS=freebsd GOARCH=amd64 $(GO) build -tags prod -o ${BIN_DIR}/ecolinker-freebsd-amd64 ${CMD_GO_FILES}
build-freebsd-arm64:
	@GOOS=freebsd GOARCH=arm64 $(GO) build -tags prod -o ${BIN_DIR}/ecolinker-freebsd-arm64 ${CMD_GO_FILES}
build-darwin-amd64:
	@GOOS=darwin GOARCH=amd64 $(GO) build -tags prod -o ${BIN_DIR}/ecolinker-darwin-amd64 ${CMD_GO_FILES}
build-darwin-arm64:
	@GOOS=darwin GOARCH=arm64 $(GO) build -tags prod -o ${BIN_DIR}/ecolinker-darwin-arm64 ${CMD_GO_FILES}
build-linux-amd64:
	@GOOS=linux GOARCH=amd64 $(GO) build -tags prod -o ${BIN_DIR}/ecolinker-linux-amd64 ${CMD_GO_FILES}
build-linux-arm64:
	@GOOS=linux GOARCH=arm64 $(GO) build -tags prod -o ${BIN_DIR}/ecolinker-linux-arm64 ${CMD_GO_FILES}
build-windows-amd64:
	@GOOS=windows GOARCH=amd64 $(GO) build -tags prod -o ${BIN_DIR}/ecolinker-windows-amd64 ${CMD_GO_FILES}
build-windows-arm64:
	@GOOS=windows GOARCH=arm64 $(GO) build -tags prod -o ${BIN_DIR}/ecolinker-windows-arm64 ${CMD_GO_FILES}

.PHONY: clean test-unit dependencies checkstyle build build-local build-all build-darwin-amd64 build-darwin-arm64 build-freebsd-amd64 build-freebsd-arm64 build-linux-amd64 build-linux-arm64 build-windows-amd64 build-windows-arm64 run generate audit