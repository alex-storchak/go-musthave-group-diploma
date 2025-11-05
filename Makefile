.PHONY: help
.PHONY: lint lint-fix lint-verbose lint-accrual
.PHONY: generate-mocks test test-static
.PHONY: build-gophermart build-accrual build test-gophermart
.PHONY: build-gophermart-windows build-accrual-windows build-windows test-gophermart-windows
.PHONY: clean clean-bin clean-mocks

BUILD_VCS ?= true

GOPHERMART_PORT ?= 8080
GOPHERMART_DB_DSN ?= "postgres://gophermart:StrongPassword02-gophermart@localhost:54321/gophermart?sslmode=disable&search_path=public"

ACCRUAL_PORT ?= 8081
ACCRUAL_DB_DSN ?= "postgres://accrual:StrongPassword02-accrual@localhost:54322/accrual?sslmode=disable&search_path=public"

GOLANGCI_LINT = golangci-lint

build-gophermart: generate-mocks
	cd cmd/gophermart && go build -buildvcs=$(BUILD_VCS) -o gophermart
build-gophermart-windows: generate-mocks
	cd cmd/gophermart && go build -buildvcs=$(BUILD_VCS) -o gophermart.exe

build-accrual: generate-mocks
	cd cmd/accrual && go build -buildvcs=$(BUILD_VCS) -o accrual
build-accrual-windows: generate-mocks
	cd cmd/accrual && go build -buildvcs=$(BUILD_VCS) -o accrual.exe

build: build-gophermart build-accrual
build-windows: build-gophermart-windows build-accrual-windows

lint: generate-mocks
	$(GOLANGCI_LINT) run

lint-fix: generate-mocks
	$(GOLANGCI_LINT) run --fix

lint-verbose: generate-mocks
	$(GOLANGCI_LINT) run -v

lint-accrual: generate-mocks
	$(GOLANGCI_LINT) run ./internal/accrual/...

generate-mocks:
	mockery

clean-mocks:
	@echo "Removing generated mocks..."
	# Находим каталоги 'mocks' и удаляем mock_*.go внутри
	@find . -type d -name mocks -prune -exec sh -c ' \
		for d; do \
			find "$$d" -maxdepth 1 -type f -name "mock_*.go" -print -delete; \
			rmdir "$$d" 2>/dev/null || true; \
		done' sh {} +
	@echo "Done."

clean-bin:
	rm -f cmd/gophermart/gophermart
	rm -f cmd/accrual/accrual

clean: clean-mocks clean-bin

test: generate-mocks
	go test ./...

test-static: generate-mocks
	go vet -vettool=$(which statictest) ./...

test-gophermart: build
	gophermarttest \
		-test.v -test.run=^TestGophermart$ \
		-gophermart-binary-path=cmd/gophermart/gophermart \
		-gophermart-host=localhost \
		-gophermart-port=$(GOPHERMART_PORT) \
		-gophermart-database-uri=$(GOPHERMART_DB_DSN) \
		-accrual-binary-path=cmd/accrual/accrual \
		-accrual-host=localhost \
		-accrual-port=$(ACCRUAL_PORT) \
		-accrual-database-uri=$(ACCRUAL_DB_DSN)

test-gophermart-windows: build-windows
	gophermarttest \
		-test.v -test.run=^TestGophermart$ \
		-gophermart-binary-path=cmd/gophermart/gophermart.exe \
		-gophermart-host=localhost \
		-gophermart-port=$(GOPHERMART_PORT) \
		-gophermart-database-uri="$(GOPHERMART_DB_DSN)" \
		-accrual-binary-path=cmd/accrual/accrual.exe \
		-accrual-host=localhost \
		-accrual-port=$(ACCRUAL_PORT) \
		-accrual-database-uri="$(ACCRUAL_DB_DSN)"

help:
	@echo ""
	@echo "Usage: make <target>"
	@echo "  make test-gophermart"
	@echo "  make test-gophermart GOPHERMART_DB_DSN=\"postgres://gophermart/dsn\" ACCRUAL_DB_DSN=\"postgres://accrual/dsn\""
	@echo ""
	@echo "Targets:"
	@echo "  build-gophermart   Build gophermart binary"
	@echo "  build-accrual      Build accrual binary"
	@echo "  build              Build both binaries"
	@echo ""
	@echo "  lint               Run golangci-lint for the whole project"
	@echo "  lint-fix           Run golangci-lint with --fix flag"
	@echo "  lint-verbose       Run golangci-lint with -v flag"
	@echo "  lint-accrual       Run golangci-lint on internal accrual package"
	@echo ""
	@echo "  generate-mocks     Generate mocks for interfaces"
	@echo ""
	@echo "  clean              Clean up mocks and binaries"
	@echo "  clean-mocks        Clean up mocks"
	@echo "  clean-bin          Clean up binaries"
	@echo ""
	@echo "  test               Run all tests"
	@echo "  test-static        Run required autotest statictest (binary file \"statictest\" must be available from PATH)"
	@echo "  test-gophermart    Run required autotest gophermarttest (binary file \"gophermarttest\" must be available from PATH)"
	@echo ""