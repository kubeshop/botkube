TAG=$(shell cut -d'=' -f2- .release)

.DEFAULT_GOAL := build
.PHONY: release git-tag check-git-status container-image test test-integration build pre-build publish lint lint-fix system-check save-images load-and-push-images

# Show this help.
help:
	@awk '/^#/{c=substr($$0,3);next}c&&/^[[:alpha:]][[:alnum:]_-]+:/{print substr($$1,1,index($$1,":")),c}1{c=0}' $(MAKEFILE_LIST) | column -s: -t

# Docker Tasks
# Make a release
release: check-git-status test git-tag gorelease
	@echo "Successfully released version $(TAG)"

# Create a git tag
git-tag:
	@echo "Creating a git tag"
	@git add .release helm/botkube CHANGELOG.md
	@git commit -m "Release $(TAG)" ;
	@git tag $(TAG) ;
	@git push --tags origin main;
	@echo 'Git tag pushed successfully' ;

# Check git status
check-git-status:
	@echo "Checking git status"
	@if [ -n "$(shell git tag | grep $(TAG))" ] ; then echo 'ERROR: Tag already exists' && exit 1 ; fi
	@if [ -z "$(shell git remote -v)" ] ; then echo 'ERROR: No remote to push tags to' && exit 1 ; fi
	@if [ -z "$(shell git config user.email)" ] ; then echo 'ERROR: Unable to detect git credentials' && exit 1 ; fi

lint:
	@golangci-lint run "./..."

lint-fix:
	@golangci-lint run --fix "./..."

# test
test: system-check
	@go test -v  -race ./...

test-integration: system-check
	@go test -v -tags=integration -race -count=1 ./test/...

# Build the binary
build: pre-build
	@cd cmd/botkube;GOOS_VAL=$(shell go env GOOS) CGO_ENABLED=0 GOARCH_VAL=$(shell go env GOARCH) go build -o $(shell go env GOPATH)/bin/botkube
	@echo "Build completed successfully"

# Build the image
container-image: pre-build
	@echo "Building docker image"
	@./hack/goreleaser.sh build
	@echo "Docker image build successfully"

# Publish release using goreleaser
gorelease:
	@echo "Publishing release with goreleaser"
	@./hack/goreleaser.sh release

# Build project and push dev images with v9.99.9-dev tag
release-snapshot:
	@./hack/goreleaser.sh release_snapshot

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

# Pre-build checks
pre-build: system-check

# Run chart lint & helm-docs
process-chart:
	@./hack/process-chart.sh
