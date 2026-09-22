## Image coordinates and build metadata
## Canonical registry: Ops ECR (eu-west-1).
ECR_REGISTRY ?= 362072154386.dkr.ecr.eu-west-1.amazonaws.com
IMAGE_PREFIX = $(ECR_REGISTRY)/concourse
## Moving tag: stable on master, latest on other branches.
IMAGE_TAG ?= $(shell if [ "$$(git rev-parse --abbrev-ref HEAD)" = "master" ]; then echo "stable"; else echo "latest"; fi)
## VERSION is taken from the VERSION file and prefixed with 'v' (e.g., v1.2.3).
VERSION := v$(shell cat VERSION)
## Git metadata used to stamp OCI labels (version/revision/created).
GIT_HEAD_SHA := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

## In GitHub Actions, ECR login is done in the workflow. Locally, use the AWS CLI.
ifdef GITHUB_ACTIONS
DOCKER_LOGIN := @true
else
DOCKER_LOGIN := aws ecr get-login-password --region eu-west-1 | docker login --username AWS --password-stdin $(ECR_REGISTRY)
endif

## Build and push both Concourse resources (read/post) with VERSION + stable/latest tags.
all: build-read-resource build-post-resource

## Build the 'slack-read-resource' image, tag with version and moving tag, then push to ECR.
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

## Build the 'slack-post-resource' image, tag with version and moving tag, then push to ECR.
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
