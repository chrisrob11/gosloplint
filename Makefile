GOLANGCI_LINT ?= golangci-lint

.PHONY: build test lint fmt fmt-check ci install plugin-e2e

build:
	go build -o bin/gosloplint ./cmd/gosloplint

install:
	go install ./cmd/gosloplint

test:
	go test -race -count=1 ./...

lint:
	$(GOLANGCI_LINT) run ./...

fmt:
	$(GOLANGCI_LINT) fmt ./...

fmt-check:
	@out=$$($(GOLANGCI_LINT) fmt --diff ./...); if [ -n "$$out" ]; then echo "$$out"; echo "run: make fmt"; exit 1; fi

# ci is exactly what the GitHub Actions ci job runs; run it before pushing.
ci: fmt-check lint test build

# plugin-e2e builds golangci-lint with the plugin compiled in (bin/custom-gcl,
# from example/.custom-gcl.yml) and checks its findings on example/ against
# example/want.txt. It needs the network to clone golangci-lint, so it is a
# separate CI job rather than part of ci.
plugin-e2e:
	cd example && $(GOLANGCI_LINT) custom
	@cd example && { ../bin/custom-gcl run --output.text.print-issued-lines=false --show-stats=false ./... > ../bin/plugin-e2e.txt; status=$$?; \
	  if [ $$status -ne 1 ]; then cat ../bin/plugin-e2e.txt; echo "custom-gcl exited $$status, want 1 (issues found)"; exit 1; fi; }
	@diff -u example/want.txt bin/plugin-e2e.txt && cat bin/plugin-e2e.txt && echo "plugin-e2e: findings match example/want.txt"
