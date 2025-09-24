REGISTRY_HOST=ghcr.io
USERNAME=apptweak
IMAGE=$(REGISTRY_HOST)/$(USERNAME)
VERSION := v$(shell cat VERSION)
GH_USER := $(shell gh api user --jq '.login')
GIT_SHA := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

all: build-read-resource build-post-resource

build-read-resource:
    docker build --platform "linux/amd64" \
        --build-arg VERSION=$(VERSION) \
        --build-arg VCS_REF=$(GIT_SHA) \
        --build-arg BUILD_DATE=$(BUILD_DATE) \
        --tag "$(IMAGE)/slack-read-resource:$(VERSION)" \
		-f read/Dockerfile .
	docker tag "$(IMAGE)/slack-read-resource:$(VERSION)" "$(IMAGE)/slack-read-resource:latest"
	gh auth token | docker login --username $(GH_USER) --password-stdin
	docker push "$(IMAGE)/slack-read-resource:$(VERSION)"
	docker push "$(IMAGE)/slack-read-resource:latest"

build-post-resource:
    docker build --platform "linux/amd64" \
        --build-arg VERSION=$(VERSION) \
        --build-arg VCS_REF=$(GIT_SHA) \
        --build-arg BUILD_DATE=$(BUILD_DATE) \
        --tag "$(IMAGE)/slack-post-resource:$(VERSION)" \
		-f post/Dockerfile .
	gh auth token | docker login --username $(GH_USER) --password-stdin
	docker tag "$(IMAGE)/slack-post-resource:$(VERSION)" "$(IMAGE)/slack-post-resource:latest"
	docker push "$(IMAGE)/slack-post-resource:$(VERSION)"
	docker push "$(IMAGE)/slack-post-resource:latest"
