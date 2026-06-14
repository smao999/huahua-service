# 部署与本地开发

## 服务清单

| 服务 | 端口 | 说明 |
|------|------|------|
| `app` | 8080 | Go Gin 主服务 |
| `akshare` | 9800 | Python gRPC 数据桥接 |
| `postgres` | 5432 | 数据库 |
| `redis` | 6379 | 缓存 |

---

## 本地开发（Docker，推荐）

源码通过 volume 挂载，改代码无需重建镜像，restart 即可生效。

### 首次启动

```bash
# 1. 准备 .env
cp .env.example .env && vim .env

# 2. 先构建 akshare 镜像（一次性，下载 Python 依赖 + 生成 stub）
docker compose build akshare

# 3. 用开发覆盖文件启动
docker compose -f docker-compose.yml -f docker-compose.dev.yml up  # 前台，日志直出终端
```

### 修改代码后的重启

```bash
# 改 Go 代码
docker compose restart app       # go run 自动重编译，约 3-5 秒

# 改 Python server 代码
docker compose restart akshare   # 秒级生效

也支持后台启动 + 按需看日志：

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d   # 后台
docker compose logs -f app          # Go 日志
docker compose logs -f akshare      # Python 日志
docker compose logs -f              # 全部
``````

### proto 变更后

改过 `akshare_proto/akshare.proto` 后需要重新生成 stub：

```bash
# 1. 本地生成 Go stub（需安装 protoc-gen-go）
go generate ./...

# 或手动:
# protoc --go_out=. --go_opt=module=huahua-service \
#   --go-grpc_out=. --go-grpc_opt=module=huahua-service \
#   akshare_proto/akshare.proto

# 2. 重建 akshare 镜像（Python stub 在 Dockerfile 中自动生成）
docker compose build akshare

# 3. 重启
docker compose restart app akshare
```

---

## 本地开发（纯命令行）

不依赖 Docker，手动启动两个进程：

```bash
# 1. 安装依赖
go mod tidy
python3 -m pip install grpcio grpcio-tools akshare pandas

# 2. 生成 Python gRPC stub（首次或 proto 变更后）
python3 -m grpc_tools.protoc -I. \
  --python_out=akshare_server \
  --grpc_python_out=akshare_server \
  akshare_proto/akshare.proto

# 3. 终端 1：启动 Python akshare 服务
python3 akshare_server/server.py

# 4. 终端 2：启动 Go 服务
AKSHARE_GRPC_ADDR=localhost:9800 go run ./cmd/server/
```

Go 端通过 `AKSHARE_GRPC_ADDR` 环境变量连接 Python 服务，默认 `localhost:9800`。

---

## 服务器部署（docker-compose）

### 前置条件

服务器上需要 Docker 和 Docker Compose。

### 部署步骤

```bash
# 1. 准备 .env 配置文件
cp .env.example .env
vim .env

# 2. 一把拉起（Python stub 在 Docker 构建时自动生成）
docker compose up -d --build
```

### Docker 镜像说明

- **Go 服务**：`./Dockerfile`，多阶段构建。编译阶段用 `golang:1.25-alpine`，运行阶段用 `alpine:3.21`，最终镜像约 15MB
- **Python akshare**：`./akshare_server/Dockerfile`，基于 `python:3.12-slim`，构建时从 proto 生成 gRPC stub，无需手动生成

### 环境变量

Go 容器通过 compose 注入 `AKSHARE_GRPC_ADDR=akshare:9800`。Docker Compose 网络 DNS 自动将 `akshare` 解析为对应容器的 IP。

### 日常命令

```bash
docker compose up -d              # 启动
docker compose logs -f app        # 查看 Go 日志
docker compose logs -f akshare    # 查看 akshare 日志
docker compose restart app        # 重启 Go 服务
docker compose down               # 停止
docker compose down -v            # 停止并清除数据卷
```

### 端口映射

| 服务 | 宿主机 | 容器内 | 用途 |
|------|--------|--------|------|
| app | 8080 | 8080 | HTTP API |
| akshare | 9800 | 9800 | gRPC（仅内网） |
| postgres | 5432 | 5432 | 数据库 |
| redis | 6379 | 6379 | 缓存 |

---

## 数据持久化

PostgreSQL 和 Redis 数据通过 Docker volumes 持久化：

- `postgres_data` — 数据库文件
- `redis_data` — Redis AOF 持久化

`docker compose down` 不删除 volumes，数据保留。加 `-v` 参数会清除。
