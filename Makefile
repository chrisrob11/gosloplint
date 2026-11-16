GOLANGCI_LINT ?= golangci-lint
# COVER_FLOOR is the lowest total statement coverage `make cover-check` (and so `make ci`)
# accepts, in percent, measured across the whole module (-coverpkg), so code reached only
# through another package's tests counts. It sits about a point under the measured total
# (94.3%) because linux and macos differ slightly. Raise it when coverage rises; never
# lower it to pass.
COVER_FLOOR ?= 93

.PHONY: build test cover cover-check lint fmt fmt-check ci install plugin-e2e

build:
	go build -o bin/gosloplint ./cmd/gosloplint

install:
	go install ./cmd/gosloplint

test:
	go test -race -count=1 ./...

cover:
	go test -race -count=1 -coverpkg=./... -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -1

cover-check: cover
	@total=$$(go tool cover -func=coverage.out | tail -1 | awk '{sub(/%/, "", $$NF); print $$NF}'); \
	if awk -v t="$$total" -v f="$(COVER_FLOOR)" 'BEGIN { exit !(t < f) }'; then \
		echo "coverage $$total% is below the floor of $(COVER_FLOOR)%"; exit 1; \
	else \
		echo "coverage $$total% meets the floor of $(COVER_FLOOR)%"; \
	fi

lint:
	$(GOLANGCI_LINT) run ./...

fmt:
	$(GOLANGCI_LINT) fmt ./...

fmt-check:
	@out=$$($(GOLANGCI_LINT) fmt --diff ./...); if [ -n "$$out" ]; then echo "$$out"; echo "run: make fmt"; exit 1; fi

# ci is exactly what the GitHub Actions ci job runs; run it before pushing.
ci: fmt-check lint cover-check build

# plugin-e2e builds golangci-lint with the plugin compiled in (bin/custom-gcl,
# from example/.custom-gcl.yml) and checks its findings on example/ against
# example/want.txt. It needs the network to clone golangci-lint, so it is a
# separate CI job rather than part of ci.
plugin-e2e:
	cd example && $(GOLANGCI_LINT) custom
	@cd example && { ../bin/custom-gcl run --output.text.print-issued-lines=false --show-stats=false ./... > ../bin/plugin-e2e.txt; status=$$?; \
	  if [ $$status -ne 1 ]; then cat ../bin/plugin-e2e.txt; echo "custom-gcl exited $$status, want 1 (issues found)"; exit 1; fi; }
	@diff -u example/want.txt bin/plugin-e2e.txt && cat bin/plugin-e2e.txt && echo "plugin-e2e: findings match example/want.txt"
