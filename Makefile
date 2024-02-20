.DEFAULT_GOAL := build
.PHONY: container-image test test-integration-slack test-integration-discord build pre-build publish lint lint-fix go-import-fmt system-check save-images load-and-push-images gen-grpc-resources gen-plugins-index build-plugins build-plugins-single gen-docs-cli gen-plugins-goreleaser

# Show this help.
help:
	@awk '/^#/{c=substr($$0,3);next}c&&/^[[:alpha:]][[:alnum:]_-]+:/{print substr($$1,1,index($$1,":")),c}1{c=0}' $(MAKEFILE_LIST) | column -s: -t

lint-fix: go-import-fmt
	@go mod tidy
	@go mod verify
	@golangci-lint run --timeout=10m --fix "./..."

go-import-fmt:
	@./hack/fmt-imports.sh

# test
test: system-check
	@go test -v  -race ./...

test-integration-slack: system-check
	@cd ./test; go test -timeout=20m -v -tags=integration -race -count=1 ./e2e/... -run "TestSlack"

test-integration-discord: system-check
	@cd ./test; go test -timeout=20m -v -tags=integration -race -count=1 ./e2e/... -run "TestDiscord"

test-integration-teams: system-check

	@cd ./test; go test -timeout=20m -v -tags=integration -race -count=1 ./e2e/... -run "TestTeams"

test-cli-migration-e2e: system-check
	@cd ./test; go test -v -tags=migration -race -count=1 ./e2e/...

test-cloud-slack-dev-e2e: system-check
	@cd ./test; go test -tags=cloud_slack_dev_e2e -race -p 1 -v -timeout 30m ./cloud-slack-dev-e2e/...

test-cloud-slack-dev-e2e-show-browser: system-check
	@cd ./test; go test -tags=cloud_slack_dev_e2e -race -p 1 -v -timeout 30m -rod=show ./cloud-slack-dev-e2e/...

# Build Botkube official plugins for all supported platforms.
build-plugins: pre-build gen-plugins-goreleaser
	@echo "Building plugins binaries"
	go run ./hack/target/build-plugins/main.go
	@echo "Build completed successfully"

# Build Botkube official plugins only for current GOOS and GOARCH.
build-plugins-single: pre-build gen-plugins-goreleaser
	@echo "Building single target plugins binaries"
	go run ./hack/target/build-plugins/main.go --single-platform --output-mode binary
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

build-single-arch-cli:
	@./hack/goreleaser.sh build_single_arch_cli

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
	go run ./hack/gen-plugin-index.go -output-path ./plugins-index.yaml -plugin-name-filter 'kubectl|kubernetes'

gen-docs-cli:
	rm -f ./cmd/cli/docs/*
	go run -ldflags="-X go.szostok.io/version.name=botkube" cmd/cli/main.go gen-usage-docs

gen-plugins-goreleaser:
	go run ./hack/target/gen-goreleaser/main.go

# Pre-build checks
pre-build: system-check

# Run chart lint & helm-docs
process-chart:
	@./hack/process-chart.sh
