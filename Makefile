# Get the latest git tag
GIT_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "")
# Get the current commit hash
GIT_COMMIT := $(shell git rev-parse --short HEAD)
# If no tag exists, use the last tag + commit hash
TAG := $(if $(GIT_TAG),$(GIT_TAG),$(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")-$(GIT_COMMIT))

# Image URL to use all building/pushing image targets
REPO?=graydovee/fileserver
IMG?=$(REPO):$(TAG)

clean:
	rm $(BINDIR)/*

.PHONY: docker-build
docker-build:
	docker build -t ${IMG} .

.PHONY: docker-release
docker-release:
	docker buildx build --platform linux/amd64,linux/arm64 -t ${IMG} -t ${REPO}:latest --push .
