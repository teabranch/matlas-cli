.PHONY: build vet fmt lint test test-unit test-integration test-e2e test-all clean clean-cache clean-all coverage coverage-atlas

BIN_DIR := bin
BINARY := $(BIN_DIR)/matlas-cli

build:
	./scripts/build/build.sh

$(BINARY):
	@mkdir -p $(BIN_DIR)
	go build -o $(BINARY) .

test:
	./scripts/test.sh unit

test-unit:
	./scripts/test.sh unit

test-integration:
	./scripts/test.sh integration

test-e2e:
	./scripts/test.sh e2e

test-all:
	./scripts/test.sh all

vet:
	go vet ./...

fmt:
	go fmt ./...
	
lint:
	golangci-lint run --no-config --enable-only=errcheck,gosec,ineffassign --timeout=5m

test-short:
	./scripts/test.sh unit

coverage:
	./scripts/test.sh unit --coverage

coverage-atlas:
	go clean -testcache
	go test -coverprofile=atlas-coverage.out ./internal/atlas/...
	@COVERAGE=$$(go tool cover -func=atlas-coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	echo "Atlas package coverage: $$COVERAGE%"; \
	if [ $$(echo "$$COVERAGE < 90" | bc -l) -eq 1 ]; then \
		echo "❌ Coverage below 90% threshold ($$COVERAGE%)"; \
		exit 1; \
	else \
		echo "✅ Coverage meets 90% threshold ($$COVERAGE%)"; \
	fi

coverage-ci: coverage-atlas
	@echo "CI coverage check passed"

clean:
	rm -f bin/matlas-cli
	rm -f coverage.out coverage.html atlas-coverage.out

clean-cache:
	./scripts/utils/clean.sh cache

clean-all: clean clean-cache
	./scripts/utils/clean.sh all

generate-mocks:
	./scripts/generate-mocks.sh

install-hooks:
	mkdir -p .git/hooks
	cp scripts/pre-commit .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit 