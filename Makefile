## Image coordinates and build metadata
## REGISTRY_HOST/ORG_NAME form the GHCR repo prefix.
REGISTRY_HOST=ghcr.io
ORG_NAME=apptweak
## IMAGE is the full repository namespace.
IMAGE_PREFIX=$(REGISTRY_HOST)/$(ORG_NAME)/concourse
IMAGE_TAG ?= $(shell if [ "$$(git rev-parse --abbrev-ref HEAD)" = "master" ]; then echo "stable"; else echo "latest"; fi)
## VERSION is taken from the VERSION file and prefixed with 'v' (e.g., v1.2.3).
VERSION := v$(shell cat VERSION)
## GH_USER is the current GitHub username.
GH_USER := $(shell gh api user --jq '.login')
## Git metadata used to stamp OCI labels (version/revision/created).
GIT_HEAD_SHA := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")


## In GitHub Actions, we already logged in to GHCR using GITHUB_TOKEN in the workflow.
## Locally, we log in using the GitHub CLI to obtain a token, scoped to the current GitHub user.
ifdef GITHUB_ACTIONS
DOCKER_LOGIN := @true
else
DOCKER_LOGIN := gh auth token | docker login $(REGISTRY_HOST) --username $(GH_USER) --password-stdin
endif

## Build and push both Concourse resources (read/post) to GHCR with 'VERSION' and 'latest' tags.
all: build-read-resource build-post-resource

## Build the 'slack-read-resource' image, tag with version and latest, then push to GHCR.
build-read-resource:
	docker build --platform "linux/amd64" \
		--build-arg VERSION=$(VERSION) \
		--build-arg VCS_REF=$(GIT_HEAD_SHA) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--tag "$(IMAGE_PREFIX)-slack-read-resource:$(VERSION)" \
		--tag "$(IMAGE_PREFIX)-slack-read-resource:$(IMAGE_TAG)" \
		-f read/Dockerfile .
	$(DOCKER_LOGIN)
	docker push "$(IMAGE_PREFIX)-slack-read-resource:$(VERSION)"
	docker push "$(IMAGE_PREFIX)-slack-read-resource:$(IMAGE_TAG)"

## Build the 'slack-post-resource' image, tag with version and latest, then push to GHCR.
build-post-resource:
	docker build --platform "linux/amd64" \
		--build-arg VERSION=$(VERSION) \
		--build-arg VCS_REF=$(GIT_HEAD_SHA) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--tag "$(IMAGE_PREFIX)-slack-post-resource:$(VERSION)" \
		--tag "$(IMAGE_PREFIX)-slack-post-resource:$(IMAGE_TAG)" \
		-f post/Dockerfile .
	$(DOCKER_LOGIN)
	docker push "$(IMAGE_PREFIX)-slack-post-resource:$(VERSION)"
	docker push "$(IMAGE_PREFIX)-slack-post-resource:$(IMAGE_TAG)"
