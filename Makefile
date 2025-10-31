.PHONY: help build-gophermart build-accrual build lint lint-fix lint-verbose lint-accrual test-static test-gophermart

BUILD_VCS ?= true

GOPHERMART_PORT ?= 8080
GOPHERMART_DB_DSN ?= "postgres://gophermart:StrongPassword02-gophermart@localhost:54321/gophermart?sslmode=disable&search_path=public"

ACCRUAL_PORT ?= 8081
ACCRUAL_DB_DSN ?= "postgres://accrual:StrongPassword02-accrual@localhost:54322/accrual?sslmode=disable&search_path=public"

GOLANGCI_LINT = golangci-lint

build-gophermart:
	cd cmd/gophermart && go build -buildvcs=$(BUILD_VCS) -o gophermart

build-accrual:
	cd cmd/accrual && go build -buildvcs=$(BUILD_VCS) -o accrual

build: build-gophermart build-accrual

lint:
	$(GOLANGCI_LINT) run

lint-fix:
	$(GOLANGCI_LINT) run --fix

lint-verbose:
	$(GOLANGCI_LINT) run -v

lint-accrual:
	$(GOLANGCI_LINT) run ./internal/accrual/...

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
	@echo "  test-static        Run required autotest statictest (binary file \"statictest\" must be available from PATH)"
	@echo "  test-gophermart    Run required autotest gophermarttest (binary file \"gophermarttest\" must be available from PATH)"
	@echo ""