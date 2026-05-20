GO ?= GO111MODULE=on CGO_ENABLED=0 go
GO_TEST ?= CGO_ENABLED=1 go
GOOS ?= $(shell $(GO) version | cut -d' ' -f4 | cut -d'/' -f1)
GOARCH ?= $(shell $(GO) version | cut -d' ' -f4 | cut -d'/' -f2)

CMD_GO_FILES ?= ./cmd/ecolinker/main.go

export GO111MODULE=on

GRYPE ?= grype

BIN_DIR = $(shell pwd)/bin
TEST_DIR = "$(shell pwd)/coverage"

clean:
	@rm -rf ${BIN_DIR}
	@rm -rf ${TEST_DIR}
	@$(GO) clean -testcache

dependencies:
	$(GO) mod download

checkstyle:
	golangci-lint run

checkstyle-fix:
	golangci-lint run --fix

generate:
	$(GO) generate ./...

test:
	$(GO_TEST) test -race -shuffle on -v ./...

test-coverage:
	@make clean
	@mkdir -p ${TEST_DIR}
	$(GO_TEST) build -cover -o ${BIN_DIR}/testapp ./cmd/ecolinker
	TEST_DIR=${TEST_DIR} TEST_BINARY=${BIN_DIR}/testapp $(GO_TEST) test -coverprofile ${TEST_DIR}/coverage.unit.out -race -shuffle on -v ./...
	$(GO_TEST) tool covdata textfmt -i=${TEST_DIR} -o=${TEST_DIR}/coverage.integration.out
	@cat ${TEST_DIR}/coverage.unit.out > ${TEST_DIR}/coverage.out
	@tail -n +2 ${TEST_DIR}/coverage.integration.out >> ${TEST_DIR}/coverage.out
	@grep -v -E "dto.go|enum.go|_generated.go|_test.go|main.go" ${TEST_DIR}/coverage.out > ${TEST_DIR}/coverage.final.out || true
	$(GO_TEST) tool cover -func=${TEST_DIR}/coverage.final.out

run:
	$(GO) run ${CMD_GO_FILES} server serve

scan:
	@NO_COLOR=1 $(GRYPE) -v -o table --file bin/grype.txt --fail-on critical bin/ || true
	@cat ./bin/grype.txt

build-local:
	$(GO) build -o ${BIN_DIR}/ecolinker-${GOOS}-${GOARCH} ${CMD_GO_FILES}

build: build-all

build-all: build-freebsd-amd64 build-freebsd-arm64 build-darwin-amd64 build-darwin-arm64 build-linux-amd64 build-linux-arm64 build-windows-amd64 build-windows-arm64

build-freebsd-amd64:
	GOOS=freebsd GOARCH=amd64 $(GO) build -o ${BIN_DIR}/ecolinker-freebsd-amd64 ${CMD_GO_FILES}
build-freebsd-arm64:
	GOOS=freebsd GOARCH=arm64 $(GO) build -o ${BIN_DIR}/ecolinker-freebsd-arm64 ${CMD_GO_FILES}
build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 $(GO) build -o ${BIN_DIR}/ecolinker-darwin-amd64 ${CMD_GO_FILES}
build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 $(GO) build -o ${BIN_DIR}/ecolinker-darwin-arm64 ${CMD_GO_FILES}
build-linux-amd64:
	GOOS=linux GOARCH=amd64 $(GO) build -o ${BIN_DIR}/ecolinker-linux-amd64 ${CMD_GO_FILES}
build-linux-arm64:
	GOOS=linux GOARCH=arm64 $(GO) build -o ${BIN_DIR}/ecolinker-linux-arm64 ${CMD_GO_FILES}
build-windows-amd64:
	GOOS=windows GOARCH=amd64 $(GO) build -o ${BIN_DIR}/ecolinker-windows-amd64 ${CMD_GO_FILES}
build-windows-arm64:
	GOOS=windows GOARCH=arm64 $(GO) build -o ${BIN_DIR}/ecolinker-windows-arm64 ${CMD_GO_FILES}

.PHONY: clean dependencies generate build build-local build-all build-darwin-amd64 build-darwin-arm64 build-freebsd-amd64 build-freebsd-arm64 build-linux-amd64 build-linux-arm64 build-windows-amd64 build-windows-arm64 checkstyle checkstyle-fix scan test test-coverage run