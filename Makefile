GOCMD=go
GOTEST=$(GOCMD) test

.PHONY: build
go-build: ## Build
	go build

.PHONY: go-lint
go-lint: ## Lint
	golangci-lint --version
	golangci-lint run

.PHONY: go-install-lint
go-lint-install: ## Install go linter `golangci-lint`
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin latest

.PHONY: go-mod-tidy
go-mod-tidy: ## Mod tidy
	@go mod tidy

.PHONY: go-mod-check
go-mod-check: ## Mod tidy check
	go mod tidy
	git diff --exit-code -- go.mod go.sum

.PHONY: go-mod-update
go-mod-update: ## Mod Update
	@go get -u ./...

.PHONY: go-test
go-test: 
ifdef RUN_INTEGRATION_TEST
	$(GOTEST) -tags=integration -timeout 120m -v ./... $(OUTPUT_OPTIONS)
else
	$(GOTEST) -timeout 180s -v ./... $(OUTPUT_OPTIONS)
endif

.PHONY: docker-info
docker-info: ## Display Docker image information
	@echo "Docker image: registry.zondax.dev/fil-trace-check:$(shell git describe --tags --always --dirty)"
	@echo "Build date: $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')"
	@echo "Git commit: $(shell git rev-parse HEAD)"
	@echo "Go version: $(shell go version | cut -d' ' -f3)"

.PHONY: docker-publish
docker-publish: ## Build and publish Docker image
	@echo "Building Docker image..."
	docker build -t zondax/fil-trace-check:$(shell git describe --tags --always --dirty) .
	docker tag zondax/fil-trace-check:$(shell git describe --tags --always --dirty) zondax/fil-trace-check:latest
	@echo "Publishing to registry.zondax.dev..."
	docker push zondax/fil-trace-check:$(shell git describe --tags --always --dirty)
	docker push zondax/fil-trace-check:latest
	@echo "Docker image published: registry.zondax.dev/fil-trace-check:$(shell git describe --tags --always --dirty)"