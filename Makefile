.PHONY: lint lint-fix lint-verbose

GOLANGCI_LINT = golangci-lint

lint:
	$(GOLANGCI_LINT) run

lint-fix:
	$(GOLANGCI_LINT) run --fix

lint-verbose:
	$(GOLANGCI_LINT) run -v