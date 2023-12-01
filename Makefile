TAG?=0.0.1
# Image URL to use all building/pushing image targets
REPO?=graydovee/fileserver
IMG?=$(REPO):$(TAG)

clean:
	rm $(BINDIR)/*

.PHONY: docker-build
docker-build:
	docker buildx build --platform linux/amd64,linux/arm64 -t ${IMG} -t ${REPO}:latest --push .