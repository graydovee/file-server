# 简单文件管理系统

## 开发目的
存在多个设备之间传输文件的需求，设备之间经常都没有公网IP和第三方文件下载工具，所以需要一个简单的文件管理系统，可以通过简单命令实现文件上传下载。

## 功能
- [x] Web 界面（React 18 + TypeScript + Tailwind CSS）：文件浏览、分页、服务端模糊搜索
- [x] 首页快速复制命令
- [x] 文件上传（multipart 表单 / PUT 流式）
- [x] 文件下载
- [x] 文件删除
- [x] 文件列表（服务端分页，大量文件不卡顿）
- [x] 模糊搜索（服务端扫描，按需返回前 N 条）
- [x] 批量下载：非 presign 模式提供 zip 打包流式下载；presign 模式提供批量下载脚本（bash / PowerShell）
- [x] 代码片段分享（语法高亮展示 + 复制 + 下载）
- [x] S3/OSS 预签名直链下载（可选，数据不经过服务器转发）
- [ ] 鉴权

## 快速开始

### 本地开发

```bash
# 前端（web/ 目录，Node >= 20）
cd web && npm install && npm run build   # 产物输出到 web/dist

# 后端（自动加载 .env）
go run cmd/main.go
# 打开 http://localhost:8080
```

前端单独调试：`cd web && npm run dev`（vite 代理后端路由到 localhost:8080）。

### 配置

支持环境变量（`.env`，参考下方列表）与同名 cobra flag（见 `go run cmd/main.go --help`）：

| 环境变量 | 说明 | 默认值 |
|---|---|---|
| `SERVER_LISTEN_ADDRESS` | 监听地址 | `:8080` |
| `SERVER_ENABLE_TLS` | 生成的 URL 使用 https | 关 |
| `INTERNAL_HOST` | 内网下载地址的 host | `127.0.0.1` |
| `WEB_DIST_DIR` | 前端构建产物目录 | `./web/dist` |
| `STORE_TYPE` | 存储类型：`local` / `s3` | — |
| `STORE_LOCAL_UPLOAD_DIR` | 本地存储目录 | `./uploads` |
| `STORE_S3_ENDPOINT` | S3/OSS endpoint | — |
| `STORE_S3_ACCESS_KEY_ID` / `STORE_S3_SECRET_ACCESS_KEY` / `STORE_S3_BUCKET` | S3 凭据 | — |
| `STORE_S3_DISABLE_PATH_STYLE` / `STORE_S3_DISABLE_SSL` | S3 兼容选项 | 关 |
| `STORE_S3_PRESIGN_DOWNLOAD` | 开启预签名直链下载（仅 s3 生效） | 关 |
| `STORE_S3_PRESIGN_EXPIRY` | 预签名链接有效期 | `1h` |
| `STORE_S3_PRESIGN_ENDPOINT` | 预签名专用 endpoint（如公网入口，不填用 S3 endpoint） | — |

### 常用命令

```bash
# 上传（二选一）
curl -F "file=@本地文件" http://<host>:8080/upload
curl -H "X-Filename: 本地文件" --upload-file 本地文件 http://<host>:8080/upload

# 下载 / 浏览
wget http://<host>:8080/download/<路径> -O 保存名
# 浏览器打开 http://<host>:8080/download 进入文件管理页

# 批量下载（presign 模式，bash 执行，数据不过服务器）
bash <(curl -fsSL 'http://<host>:8080/script/download?path=<目录>&os=bash')

# 代码片段分享
# 浏览器打开 http://<host>:8080/code
```

### API 一览

| 方法 | 路径 | 说明 |
|---|---|---|
| POST/PUT | `/upload` | 上传（表单 / `X-Filename` 流式） |
| GET | `/download/<path>` | 下载文件；目录或不存在时回落前端页面 |
| DELETE | `/delete/<path>` | 删除 |
| GET | `/archive/<dir>` | 目录打包 zip 流式下载 |
| GET | `/script/download?path=&os=bash\|powershell[&download=1]` | 批量下载脚本（`download=1` 时作为附件下载） |
| GET | `/api/info` | 服务信息（地址、presign 状态） |
| GET | `/api/files?path=&cursor=&size=` | 目录单层分页列表 |
| GET | `/api/files/search?path=&q=&limit=` | 模糊搜索（前缀范围内，前 N 条 + truncated） |
| POST | `/api/code` | 上传代码片段（JSON） |
| GET | `/api/code/<lang>/<hash>` | 读取代码片段 |

### Docker

```bash
make docker-build   # 本地构建
make docker-release # buildx 双架构 (amd64+arm64) 推送镜像
make test           # go vet + go test
```

## 部署

通过 [charts 仓库](https://git.graydove.cn/graydove/charts) `apps/fileserver` 经 ArgoCD 部署到 raspberry 集群（namespace `file-manager`，域名 `s.qaer.cn` / `s.qaer.io`）。
