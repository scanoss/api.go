
#vars
IMAGE_NAME=scanoss-go-api
REPO=scanoss
DOCKER_FULLNAME=${REPO}/${IMAGE_NAME}
GHCR_FULLNAME=ghcr.io/${REPO}/${IMAGE_NAME}
VERSION=$(shell ./version.sh)

# HELP
# This will output the help for each task
# thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help

help: ## This help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help

clean:  ## Clean all dev data
	@echo "Removing dev data..."
	@rm -f pkg/cmd/version.txt version.txt target/*

clean_testcache:  ## Expire all Go test caches
	@echo "Cleaning test caches..."
	go clean -testcache ./...

version:  ## Produce dependency version text file
	@echo "Writing version file..."
	echo $(VERSION) > pkg/cmd/version.txt

unit_test:  ## Run all unit tests in the pkg folder
	@echo "Running unit test framework..."
	go test -v ./pkg/...

int_test: clean_testcache  ## Run all integration tests in the tests folder
	@echo "Running integration test framework..."
	go test -v ./tests

run_local:  ## Launch the API locally for test
	@echo "Launching API locally..."
	go run cmd/server/main.go -json-config config/app-config-dev.json -debug

docker_build_test:
	@echo "Building test image..."
	docker build . -t scanoss_api_go_service_test --target=test

e2e_test: docker_build_test clean_testcache
	@echo "Running End-to-End tests..."
	docker-compose down
	docker-compose up -d
	docker-compose exec -T http go test -v -tags="integration e2e" ./tests
	docker-compose down

ghcr_build: version  ## Build GitHub container image
	@echo "Building GHCR container image..."
	docker build --no-cache -t $(GHCR_FULLNAME) --platform linux/amd64 .

ghcr_tag:  ## Tag the latest GH container image with the version from Git tag
	@echo "Tagging GHCR latest image with $(VERSION)..."
	docker tag $(GHCR_FULLNAME):latest $(GHCR_FULLNAME):$(VERSION)

ghcr_push:  ## Push the GH container image to GH Packages
	@echo "Publishing GHCR container $(VERSION)..."
	docker push $(GHCR_FULLNAME):$(VERSION)
	docker push $(GHCR_FULLNAME):latest

ghcr_all: ghcr_build ghcr_tag ghcr_push  ## Execute all GitHub Package container actions

build_amd: version  ## Build an AMD 64 binary
	@echo "Building AMD binary $(VERSION)..."
	go generate ./pkg/cmd/server.go
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o ./target/scanoss-go-api-linux-amd64 ./cmd/server

build_arm: version  ## Build an ARM 64 binary
	@echo "Building ARM binary $(VERSION)..."
	go generate ./pkg/cmd/server.go
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-w -s" -o ./target/scanoss-go-api-linux-arm64 ./cmd/server

package: package_amd  ## Build & Package an AMD 64 binary

package_amd: version  ## Build & Package an AMD 64 binary
	@echo "Building AMD binary $(VERSION) and placing into scripts..."
	go generate ./pkg/cmd/server.go
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o ./scripts/scanoss-go-api ./cmd/server

package_arm: version  ## Build & Package an ARM 64 binary
	@echo "Building ARM binary $(VERSION) and placing into scripts..."
	go generate ./pkg/cmd/server.go
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-w -s" -o ./scripts/scanoss-go-api ./cmd/server
