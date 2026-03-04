GO_FILES_NO_MOCKS_NO_SQL_NO_PB := $(shell find . -type f -iname '*.go' -not -iname "mock_*.go" -not -iname "*.sql.go" -not -iname "*.pb.go")

GREEN   := $(shell tput -Txterm setaf 2)
YELLOW  := $(shell tput -Txterm setaf 3)
WHITE   := $(shell tput -Txterm setaf 7)
RESET   := $(shell tput -Txterm sgr0)

.PHONY: all run run.dev test test.verbose test.coverage fmt build help

all: help

.PHONY: run
run:
	go run cmd/server/main.go

.PHONY: run.dev
run.dev:
	ENV=development go run cmd/server/main.go

.PHONY: test
test:
	go test ./...

.PHONY: test.race
test.race:
	go test -race ./...

.PHONY: test.verbose
test.verbose:
	go test -v -race ./...

.PHONY: test.coverage
test.coverage:
	-go test ./... -covermode=count -coverprofile=coverage.out ; go tool cover -html=coverage.out
	-rm coverage.out

.PHONY: fmt
fmt:
	@gofumpt -l -w . $(GO_FILES_NO_MOCKS_NO_SQL_NO_PB)

.PHONY: lint
lint:
	@golangci-lint run ./...

.PHONY: lint.fix
lint.fix:
	@golangci-lint run --fix ./...

.PHONY: build
build:
	@go build -o coin-futures-websocket ./cmd/server/main.go

help:
	@echo ''
	@echo 'Usage:'
	@echo '  ${YELLOW}make${RESET} ${GREEN}<target>${RESET}'
	@echo ''
	@echo 'Targets:'
	@echo "  ${YELLOW}help             ${RESET} ${GREEN}Show this help message${RESET}"
	@echo "  ${YELLOW}run              ${RESET} ${GREEN}Run the server using default config${RESET}"
	@echo "  ${YELLOW}run.dev          ${RESET} ${GREEN}Run the server using development config${RESET}"
	@echo "  ${YELLOW}test             ${RESET} ${GREEN}Run the tests of the project${RESET}"
	@echo "  ${YELLOW}test.race        ${RESET} ${GREEN}Run the tests of the project while also checking race conditions${RESET}"
	@echo "  ${YELLOW}test.verbose     ${RESET} ${GREEN}Run the tests of the project while also checking race conditions (verbose)${RESET}"
	@echo "  ${YELLOW}test.coverage    ${RESET} ${GREEN}Run the tests of the project and export the coverage${RESET}"
	@echo "  ${YELLOW}fmt              ${RESET} ${GREEN}Format '*.go' files with gofumpt${RESET}"
	@echo "  ${YELLOW}lint             ${RESET} ${GREEN}Run linter using golangci-lint${RESET}"
	@echo "  ${YELLOW}lint.fix         ${RESET} ${GREEN}Run linter using golangci-lint and fix it${RESET}"
	@echo "  ${YELLOW}build            ${RESET} ${GREEN}Build the server${RESET}"
	@echo ""
