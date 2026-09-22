## Image coordinates and build metadata
## GHCR (legacy dual-publish) and optional ECR (canonical Ops registry).
GHCR_PREFIX = ghcr.io/apptweak/concourse
ECR_REGISTRY ?= 362072154386.dkr.ecr.eu-west-1.amazonaws.com
## Moving tag: stable on master, latest on other branches.
IMAGE_TAG ?= $(shell if [ "$$(git rev-parse --abbrev-ref HEAD)" = "master" ]; then echo "stable"; else echo "latest"; fi)
## VERSION is taken from the VERSION file and prefixed with 'v' (e.g., v1.2.3).
VERSION := v$(shell cat VERSION)
## GH_USER is the current GitHub username.
GH_USER := $(shell gh api user --jq '.login')
## Git metadata used to stamp OCI labels (version/revision/created).
GIT_HEAD_SHA := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

## Set ECR_PUSH=1 in CI after amazon-ecr-login to also push to Ops ECR.
ECR_PUSH ?= 0

## In GitHub Actions, GHCR login is done in the workflow; locally use gh auth token.
ifdef GITHUB_ACTIONS
DOCKER_LOGIN_GHCR := @true
else
DOCKER_LOGIN_GHCR := gh auth token | docker login ghcr.io --username $(GH_USER) --password-stdin
endif

## Build and push both Concourse resources (read/post) with VERSION + stable/latest tags.
all: build-read-resource build-post-resource

define push_image
	$(DOCKER_LOGIN_GHCR)
	docker push "$(GHCR_PREFIX)-$(1):$(VERSION)"
	docker push "$(GHCR_PREFIX)-$(1):$(IMAGE_TAG)"
	@if [ "$(ECR_PUSH)" = "1" ]; then \
		docker tag "$(GHCR_PREFIX)-$(1):$(VERSION)" "$(ECR_REGISTRY)/concourse-$(1):$(VERSION)"; \
		docker tag "$(GHCR_PREFIX)-$(1):$(IMAGE_TAG)" "$(ECR_REGISTRY)/concourse-$(1):$(IMAGE_TAG)"; \
		docker push "$(ECR_REGISTRY)/concourse-$(1):$(VERSION)"; \
		docker push "$(ECR_REGISTRY)/concourse-$(1):$(IMAGE_TAG)"; \
	fi
endef

## Build the 'slack-read-resource' image, tag with version and moving tag, then push.
build-read-resource:
	docker build --platform "linux/amd64" \
		--build-arg VERSION=$(VERSION) \
		--build-arg VCS_REF=$(GIT_HEAD_SHA) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--tag "$(GHCR_PREFIX)-slack-read-resource:$(VERSION)" \
		--tag "$(GHCR_PREFIX)-slack-read-resource:$(IMAGE_TAG)" \
		-f read/Dockerfile .
	$(call push_image,slack-read-resource)

## Build the 'slack-post-resource' image, tag with version and moving tag, then push.
build-post-resource:
	docker build --platform "linux/amd64" \
		--build-arg VERSION=$(VERSION) \
		--build-arg VCS_REF=$(GIT_HEAD_SHA) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--tag "$(GHCR_PREFIX)-slack-post-resource:$(VERSION)" \
		--tag "$(GHCR_PREFIX)-slack-post-resource:$(IMAGE_TAG)" \
		-f post/Dockerfile .
	$(call push_image,slack-post-resource)
