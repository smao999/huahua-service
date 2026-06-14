# akshare gRPC 桥接方案

## 概述

本项目通过 gRPC 桥接方式在 Go 服务中调用 Python `akshare` 财经数据库，实现完整的 akshare 接口覆盖，无需在 Go 侧逐接口重新实现。

### 架构

```
┌──────────────────────────────────────────────────────┐
│  huahua-service (Go)                                 │
│                                                      │
│  internal/service/akshare/client.go                  │
│    │  import "huahua-service/akshare_proto/akshare"  │
│    │                                                 │
│    ▼  gRPC (protobuf, localhost:9800)                │
│                                                      │
│  akshare_server/server.py (Python 子进程)             │
│    │  import akshare                                 │
│    ▼                                                 │
│  东方财富 / 天天基金 / 新浪财经 / ...                    │
└──────────────────────────────────────────────────────┘
```

### 核心设计

| 组件 | 文件 | 职责 |
|------|------|------|
| 协议定义 | `akshare_proto/akshare.proto` | 接口契约，唯一的事实源 |
| 生成代码 | `akshare_proto/akshare/akshare.pb.go` | protoc 自动生成的消息体 |
| 生成代码 | `akshare_proto/akshare/akshare_grpc.pb.go` | protoc 自动生成的客户端接口 |
| Python 服务 | `akshare_server/server.py` | gRPC 服务端，直调 akshare |
| Go 客户端 | `internal/service/akshare/client.go` | 对上层暴露的封装 |

### 三个 gRPC 接口

| RPC | 用途 | 参数 |
|-----|------|------|
| `Call` | **通用调度器**：支持任意 akshare 函数 | `fn` (函数名) + `kwargs` (参数 map)，返回 JSON |
| `BatchGetFundEstimates` | 高频：批量实时估值 | `codes` 列表，返回强类型估值数组 |
| `GetFundHistory` | 高频：历史净值 | `code` 单只基金，返回净值记录数组 |

高频接口走强类型（类型安全、编译期检查），长尾接口走通用 `Call`（零代码新增）。

---

## 使用方法

### Go 端调用示例

```go
import (
    akshare "huahua-service/internal/service/akshare"
)

// 初始化（已在 bootstrap 中自动完成）
client, err := akshare.NewClient("localhost:9800")

// 1. 通用调用 —— 调任意 akshare 函数
data, err := client.Call("stock_zh_a_hist", map[string]string{
    "symbol": "000001",
    "period": "daily",
})

// 2. 通用调用 + 自动反序列化
var result MyStruct
err := client.CallGeneric("fund_individual_basic_info_xq",
    map[string]string{"symbol": "000001"}, &result)

// 3. 批量估值（强类型）
estimates, err := client.BatchGetFundEstimates([]string{"000001", "000002"})
for _, e := range estimates {
    fmt.Printf("%s: 估值 %.4f, 涨跌 %.2f%%\n", e.Name, e.Estimate, e.EstimatePercent)
}

// 4. 历史净值（强类型）
records, err := client.GetFundHistory("000001")
for _, r := range records {
    fmt.Printf("%s: 净值 %.4f, 涨跌 %.2f%%\n", r.Date, r.Nav, r.Change)
}
```

---

## 新增 akshare 接口

### 通过 Call 通用调度（推荐，零代码）

Python server 的 `Call` 方法通过 `getattr(ak, fn_name)` 动态执行，因此**任意 akshare 函数无需修改 server 代码**，Go 端直接：

```go
client.Call("新函数名", map[string]string{"参数": "值"})
```

### 新增带类型的 RPC

1. 在 `akshare.proto` 中定义新的 message 和 rpc
2. 重新生成代码（见环境准备）
3. 在 `akshare_server/server.py` 的 `AkshareServicer` 类中添加实现
4. 在 `internal/service/akshare/client.go` 中添加公用的封装方法

---

## 项目文件清单

```
huahua-service/
├── AKSHARE_GRPC.md                    ← 本文档
├── akshare_proto/
│   ├── akshare.proto                  ← 接口定义（proto3）
│   └── akshare/
│       ├── akshare.pb.go              ← 生成：消息体
│       └── akshare_grpc.pb.go         ← 生成：客户端/服务端接口
├── akshare_server/
│   ├── server.py                      ← Python gRPC 服务实现
│   └── requirements.txt               ← Python 依赖
├── internal/
│   └── service/
│       └── akshare/
│           └── client.go              ← Go 端 gRPC 客户端封装
└── go.mod
```

---

## 环境准备

### 前置依赖

```bash
# protoc 编译器
brew install protobuf

# Go gRPC 插件
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Go 依赖
go get google.golang.org/grpc google.golang.org/protobuf

# Python 依赖
python3 -m pip install grpcio grpcio-tools akshare pandas
```

### 代码生成（proto 变更后）

```bash
# 生成 Go 端
protoc --go_out=. --go_opt=module=huahua-service \
       --go-grpc_out=. --go-grpc_opt=module=huahua-service \
       akshare_proto/akshare.proto

# 生成 Python 端
python3 -m grpc_tools.protoc -I. \
       --python_out=akshare_server \
       --grpc_python_out=akshare_server \
       akshare_proto/akshare.proto
```

---

## 约束与注意事项

1. **端口**：Python gRPC 服务默认 `localhost:9800`，通过环境变量 `AKSHARE_GRPC_ADDR` 可覆盖
2. **超时**：Go 端每个请求 30 秒超时，Python 端依赖 akshare 自身的 HTTP 超时
3. **序列化**：DataFrame 自动转为 JSON 数组（`orient="records"`），非 DataFrame 类型走 `json.dumps(default=str)` 兜底
4. **进程管理**：Python 服务需要单独启动；详见部署文档
5. **连接重试**：Go 端 `NewClient` 失败返回 nil，调用方需判空
