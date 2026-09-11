# Get the latest git tag
GIT_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "")
# Get the current commit hash
GIT_COMMIT := $(shell git rev-parse --short HEAD)
# If no tag exists, use the last tag + commit hash
TAG := $(if $(GIT_TAG),$(GIT_TAG),$(shell git describe --tags --abbrev=0 2>/dev/null || echo "0.0.0")-$(GIT_COMMIT))

# Image URL to use all building/pushing image targets
REPO?=harbor.graydove.cn/apps/fileserver
IMG?=$(REPO):$(TAG)

# 只构建真实源码包，避免 uploads/ 等运行时数据目录里的 .go 文件干扰
PKGS=./cmd/... ./pkg/...

.PHONY: test
test:
	go vet $(PKGS)
	go test $(PKGS)

.PHONY: web-build
web-build:
	cd web && npm ci && npm run build

.PHONY: clean
clean:
	rm -f fileserver

.PHONY: docker-build
docker-build:
	docker build -t ${IMG} .

.PHONY: docker-release
docker-release:
	docker buildx build --builder builder --platform linux/amd64,linux/arm64 -t ${IMG} -t ${REPO}:latest --push .
