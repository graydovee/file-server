# AGENTS.md

简单文件传输服务：在无公网 IP 的设备之间通过 HTTP 完成文件上传/下载（返回可直接复制的 wget 命令），附带代码片段分享功能（`/code`）。前端为 React 18 + TypeScript SPA（`web/`），后端为 Go Echo，只提供 JSON API 与静态资源。无鉴权（README 功能清单中"鉴权"未完成，属有意现状）。

## 技术栈与常用命令

- Go（`go.mod` 的 go 指令当前为 1.26，由依赖决定；本地工具链 1.27，Docker 构建器 `golang:1.27-alpine`）。
- Web 框架 **Echo v4**（gin 已移除）；存储 SDK 为 aws-sdk-go-v2（兼容 S3/OSS/MinIO）。
- 前端：React 18 + Vite + Tailwind CSS 4（`@tailwindcss/vite`）+ lucide-react 图标 + npm 版 highlight.js；npm registry 走 npmmirror。
- **构建/测试必须限定包路径 `./cmd/... ./pkg/...`（Makefile 的 `$(PKGS)`），不要用 `./...`**：`uploads/`、`uploads/code/` 等运行时数据目录里可能有用户上传的 `.go` 文件，会被 Go 当成源码包导致编译失败。
- 构建：`go build -o fileserver cmd/main.go`（Dockerfile 分三阶段：node:22-alpine 构建前端 → golang:1.27-alpine 编译后端 → alpine:3.22 运行时，产物为二进制 + web/dist）
- 测试 / 静态检查：`make test`（= `go vet` + `go test`，限定 `$(PKGS)`）；S3 集成测试在未配置 `STORE_TYPE=s3` 时自动 skip
- 前端：`cd web && npm ci && npm run build`（typecheck + vitest + vite build）；`npm run dev` 起 vite 并把 `/api /upload /download /delete /archive /script` 代理到本地 Go
- 本地运行：`go run cmd/main.go`（godotenv 自动加载 `.env`，默认监听 `:8080`）
- 镜像：`make docker-build`、`make docker-release`（amd64+arm64 双架构，推送 `harbor.graydove.cn/graydove/fileserver`）

## 目录结构

- `cmd/main.go` — cobra 入口，所有命令行 flag 在此定义
- `pkg/httpserver.go` — Echo 引擎初始化、store 类型 switch、SubServer 组装
- `pkg/config/` — 配置：env 默认值（`GetDefault`）+ cobra flag 覆盖
- `pkg/server/` — HTTP 层：
  - `file.go` FileServer：上传/下载/删除 + `/api/info`、`/api/files`（分页）、`/api/files/search`、`/archive/*`（zip 流式）、`/script/download`（批量下载脚本）
  - `code.go` CodeServer：`/api/code` 上传与读取（JSON）
  - `script.go` bash/PowerShell 批量下载脚本生成器
  - `web.go` 前端静态资源 + SPA 回退（自定义 HTTPErrorHandler：非 `/api`、`/script`、`/assets` 的 GET 404 时回落 `web/dist/index.html`）
  - `helper.go`、`helper_test.go`
- `pkg/store/` — 存储抽象：`store.go` 定义 `Store`（`ListPage` 游标分页、`Walk` 可提前终止遍历）与 `Presigner` 接口；`localstore.go`、`s3store.go` 两个实现
- `web/` — React 前端源码（`src/views/` 四个视图：Home/Files/CodeUpload/CodeShow；`src/api.ts` 为后端 API 的类型化封装；`src/utils/format.ts` 路径与命令拼装纯函数）
- `uploads/`、`uploads/code/` — 本地 store 运行时数据（gitignore 已忽略）
- `template/`、`assert/` 已删除：旧 Go html/template 与静态资源由 Vite 打包替代

## 架构约定

- handler 只依赖 `store.Store` / `store.Presigner` 接口，不直接触碰本地文件系统或 S3。
- 新增配置项要同步三处：`pkg/config/config.go`（env 读取与默认值）、`cmd/main.go`（cobra flag）、实际 `.env`。
- presign 能力探测：`store.Presigner` 接口断言 + `cfg.Store.S3.PresignDownload` 开关同时满足才启用；local store 自动回退代理转发。
- presign URL 的 host = 签发时的 endpoint。集群内 MinIO 用内网 endpoint 时，必须配 `STORE_S3_PRESIGN_ENDPOINT`（公网入口如 `https://minio-api.qaer.io`），否则外部设备打不开直链。
- 批量下载双模式由前端按 `/api/info` 的 `presignEnabled` 切换：非 presign → `/archive` zip（浏览器直接下载）；presign → `/script/download` 脚本（内嵌 presign 直链，数据不过服务器）。
- 前端新增后端交互先在 `web/src/api.ts` 补类型化封装；路径一律走 `encodePath` 逐段转义。
- 日志：handler 内用 `c.Logger()`（echo），启动日志用标准库 `log`。

## 已知坑

- Go 工具链扫描整个 module 目录，运行时数据目录中的 `.go` 文件会破坏 `go build ./...`（见上，用 `$(PKGS)`）。
- `--tls` / `EnableTls` 只影响生成的 URL scheme（http/https），服务本身始终明文 HTTP 监听（TLS 由集群 ingress 终结）。
- 上传文件加时间戳前缀并按 `YYYY/MM/` 路径存放；生成下载链接时路径逐段 `url.PathEscape`。
- CodeServer 片段文件名 = `时间戳-md5前8位`，扩展名由 `extMap` 白名单限定；展示路由 `/code/:lang/:hash` 是前端路由，API 在 `/api/code/:lang/:hash`。
- aws-sdk-go-v2 较新版本的 `Size`/`ContentLength` 是 `*int64`，用 `aws.ToInt64` 取值；自定义 endpoint 优先用 `s3.Options.BaseEndpoint` 传递。
- `.env` / `agora.env` 含真实 S3 凭据，已被 `*.env` 忽略——不要提交，也不要把其中的凭据复制进代码、日志或回复里。
- 部署：charts 仓库 `apps/fileserver`（ArgoCD 自动同步，ns `file-manager`，域名 `s.qaer.cn`/`s.qaer.io`），镜像发 `harbor.graydove.cn/graydove/fileserver`，chart 探针 GET `/`（SPA 回退保证 200）。
