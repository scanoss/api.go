
#vars
IMAGE_NAME=scanoss-wayuu2
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
	@rm -f pkg/cmd/version.txt version.txt

version:  ## Produce dependency version text file
	@echo "Writing version file..."
	echo $(VERSION) > pkg/cmd/version.txt

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
