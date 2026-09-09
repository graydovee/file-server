# ---- 前端构建（纯 JS 构建与架构无关，跑在本机平台） ----
FROM --platform=$BUILDPLATFORM node:22-alpine AS frontend
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm config set registry https://registry.npmmirror.com && npm ci
COPY web/ ./
RUN npm run build

# ---- 后端构建（交叉编译出目标架构二进制，避免 QEMU 模拟） ----
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /source
ENV CGO_ENABLED=0 GOPROXY=https://goproxy.cn,direct

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY pkg/ pkg/

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /fileserver cmd/main.go

# ---- 运行时 ----
FROM alpine:3.22
WORKDIR /app
COPY --from=builder /fileserver /app/fileserver
COPY --from=frontend /web/dist /app/web/dist
ENTRYPOINT ["/app/fileserver"]
