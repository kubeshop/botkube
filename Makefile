.DEFAULT_GOAL := build
.PHONY: container-image test test-integration-slack test-integration-discord build pre-build publish lint lint-fix go-import-fmt system-check save-images load-and-push-images gen-grpc-resources gen-plugins-index build-plugins build-plugins-single

# Show this help.
help:
	@awk '/^#/{c=substr($$0,3);next}c&&/^[[:alpha:]][[:alnum:]_-]+:/{print substr($$1,1,index($$1,":")),c}1{c=0}' $(MAKEFILE_LIST) | column -s: -t

lint-fix: go-import-fmt
	@go mod tidy
	@go mod verify
	@golangci-lint run --fix "./..."

go-import-fmt:
	@./hack/fmt-imports.sh

# test
test: system-check
	@go test -v  -race ./...

test-integration-slack: system-check
	@go test -v -tags=integration -race -count=1 ./test/e2e/... -run "TestSlack"

test-integration-discord: system-check
	@go test -v -tags=integration -race -count=1 ./test/e2e/... -run "TestDiscord"

test-migration-tool: system-check
	@go test -v -race -count=1 ./test/migration/e2e/...

# Build the binary
build: pre-build
	@cd cmd/botkube;GOOS_VAL=$(shell go env GOOS) CGO_ENABLED=0 GOARCH_VAL=$(shell go env GOARCH) go build -o $(shell go env GOPATH)/bin/botkube
	@echo "Build completed successfully"

# Build Botkube official plugins for all supported platforms.
build-plugins: pre-build
	@echo "Building plugins binaries"
	@./hack/goreleaser.sh build_plugins
	@echo "Build completed successfully"

# Build Botkube official plugins only for current GOOS and GOARCH.
build-plugins-single: pre-build
	@echo "Building single target plugins binaries"
	@./hack/goreleaser.sh build_plugins_single
	@echo "Build completed successfully"

# Build the image
container-image: pre-build
	@echo "Building docker image"
	@./hack/goreleaser.sh build
	@echo "Docker image build successful"

# Build the image
container-image-single: pre-build
	@echo "Building single target docker image"
	@./hack/goreleaser.sh build_single
	@echo "Single target docker image build successful"

# Build project and push dev images with v9.99.9-dev tag
release-snapshot:
	@./hack/goreleaser.sh release_snapshot

# Build botkube cli
release-botkube-cli:
	@./hack/goreleaser.sh build_botkube_cli

# Build project and save images with IMAGE_TAG tag
save-images:
	@./hack/goreleaser.sh save_images

# Load project and push images with IMAGE_TAG tag
load-and-push-images:
	@./hack/goreleaser.sh load_and_push_images

# system checks
system-check:
	@echo "Checking system information"
	@if [ -z "$(shell go env GOOS)" ] || [ -z "$(shell go env GOARCH)" ] ; \
	then \
	echo 'ERROR: Could not determine the system architecture.' && exit 1 ; \
	else \
	echo 'GOOS: $(shell go env GOOS)' ; \
	echo 'GOARCH: $(shell go env GOARCH)' ; \
	echo 'System information checks passed.'; \
	fi ;

# Generate gRPC Go code for client and server.
gen-grpc-resources:
	@./hack/gen-grpc-resources.sh

# Generate plugins YAML index files for both all plugins and end-user ones.
gen-plugins-index: build-plugins
	go run ./hack/gen-plugin-index.go -output-path ./plugins-dev-index.yaml
	go run ./hack/gen-plugin-index.go -output-path ./plugins-index.yaml -plugin-name-filter 'kubectl|helm|kubernetes|prometheus'

# Pre-build checks
pre-build: system-check

# Run chart lint & helm-docs
process-chart:
	@./hack/process-chart.sh
