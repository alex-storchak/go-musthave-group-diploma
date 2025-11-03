.PHONY: help
.PHONY: lint lint-fix lint-verbose lint-accrual
.PHONY: mock-generate test test-static
.PHONY: build-gophermart build-accrual build test-gophermart
.PHONY: build-gophermart-windows build-accrual-windows build-windows test-gophermart-windows
.PHONY: clean clean-bin clean-mock

BUILD_VCS ?= true

GOPHERMART_PORT ?= 8080
GOPHERMART_DB_DSN ?= "postgres://gophermart:StrongPassword02-gophermart@localhost:54321/gophermart?sslmode=disable&search_path=public"

ACCRUAL_PORT ?= 8081
ACCRUAL_DB_DSN ?= "postgres://accrual:StrongPassword02-accrual@localhost:54322/accrual?sslmode=disable&search_path=public"

GOLANGCI_LINT = golangci-lint

build-gophermart:
	cd cmd/gophermart && go build -buildvcs=$(BUILD_VCS) -o gophermart
build-gophermart-windows:
	cd cmd/gophermart && go build -buildvcs=$(BUILD_VCS) -o gophermart.exe

build-accrual:
	cd cmd/accrual && go build -buildvcs=$(BUILD_VCS) -o accrual
build-accrual-windows:
	cd cmd/accrual && go build -buildvcs=$(BUILD_VCS) -o accrual.exe

build: build-gophermart build-accrual
build-windows: build-gophermart-windows build-accrual-windows

lint:
	$(GOLANGCI_LINT) run

lint-fix:
	$(GOLANGCI_LINT) run --fix

lint-verbose:
	$(GOLANGCI_LINT) run -v

lint-accrual:
	$(GOLANGCI_LINT) run ./internal/accrual/...

mock-generate:
	mockery

clean-mock:
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

clean: clean-mock clean-bin

test:
	go test ./...

test-static:
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
	@echo "  mock-generate      Generate mocks for interfaces"
	@echo ""
	@echo "  clean              Clean up mocks and binaries"
	@echo "  clean-mock         Clean up mocks"
	@echo "  clean-bin          Clean up binaries"
	@echo ""
	@echo "  test               Run all tests"
	@echo "  test-static        Run required autotest statictest (binary file \"statictest\" must be available from PATH)"
	@echo "  test-gophermart    Run required autotest gophermarttest (binary file \"gophermarttest\" must be available from PATH)"
	@echo ""