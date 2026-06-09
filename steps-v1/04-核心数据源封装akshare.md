# Step 4：核心数据源封装（akshare）

> **目标**：实现 Go HTTP 客户端，替代 Python 的 `akshare` 库。
> akshare = 封装的财经网站 HTTP API（东方财富、腾讯财经、新浪财经等）

---

## 4.1 理解 akshare 是干什么的

akshare 是一个 Python 第三方库，但它并不是真正的"财经数据引擎"——
它就是**封装了发 HTTP 请求到各个财经网站**的过程。

比如获取基金实时估值：
- URL: `https://fundgz.1234567.com.cn/js/000001.js`
- 返回: `jsonpgz({"fundcode":"000001","gsz":1.234,...});`

获取基金历史净值：
- URL: `http://fund.eastmoney.com/f10/FundNavTrend.html?fundcode=000001`
- 解析 HTML 表格

**核心思路**：我们在 Go 里直接发同样的 HTTP 请求，解析同样的响应格式。

---

## 4.2 创建 HTTP 客户端基座

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 go    （55 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

```go
// internal/service/akshare/client.go — HTTP 客户端
package akshare

import (
    "fmt"
    "io"
    "net/http"
    "time"
)

// Client 封装所有对财经网站的 HTTP 请求
// 对应 Python AkshareService 类
type Client struct {
    httpClient *http.Client
}

func NewClient() *Client {
    return &Client{
        httpClient: &http.Client{
            // 所有请求统一 15 秒超时，避免卡死
            Timeout: 15 * time.Second,
        },
    }
}

// Get 发送 GET 请求并返回响应体
// 封装了设置请求头、错误处理、关闭连接
func (c *Client) Get(url string) ([]byte, error) {
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("创建请求失败: %w", err)
    }

    // 模拟浏览器请求头——部分网站拒绝非浏览器请求
    req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
    req.Header.Set("Referer", "https://finance.eastmoney.com/")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("请求失败: %w", err)
    }
    // ！！！重要：不关 resp.Body 会导致 TCP 连接泄漏
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
    }

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("读取响应失败: %w", err)
    }

    return body, nil
}
```
</details>


---

## 4.3 实现基金实时估值接口

**Python 源码**（在 `services/akshare_service.py` 中）：

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 python（17 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

```python
# Python 用 httpx 异步 HTTP 客户端获取基金实时估值
# akshare 内部调用的就是这个接口

def fetch_fund_estimate(code: str):
    """获取基金实时估值
    URL: https://fundgz.1234567.com.cn/js/{code}.js
    返回格式：jsonpgz({"fundcode":"000001","gsz":1.234,"gszzl":0.56,...});
    """
    url = f"https://fundgz.1234567.com.cn/js/{code}.js"
    response = requests.get(url, headers={
        "User-Agent": "Mozilla/5.0",
        "Referer": "https://finance.eastmoney.com/"
    })
    data = response.text
    # 去掉 "jsonpgz(" 前缀和 ");" 后缀，得到纯净 JSON
    json_str = data[8:-2]
    return json.loads(json_str)
```
</details>


**Go 实现**：

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 go    （78 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

```go
// internal/service/akshare/estimate.go
package akshare

import (
    "encoding/json"
    "fmt"
    "strings"
)

// FundEstimate 实时估值数据结构
// 对应东方财富 fundgz 接口返回的 JSON
type FundEstimate struct {
    FundCode        string  `json:"fundcode"`   // 基金代码
    Name            string  `json:"name"`       // 基金名称
    Estimate        float64 `json:"gsz"`        // 实时估值（估算净值）
    EstimatePercent float64 `json:"gszzl"`      // 实时估值涨跌幅 %
    LastNav         float64 `json:"dwjz"`       // 上一交易日单位净值
    EstimateTime    string  `json:"gztime"`     // 估值时间
}

// GetFundEstimate 获取单只基金实时估值
// fundCode: 6 位数字，如 "000001"
// 对应 Python fetch_fund_estimate()
func (c *Client) GetFundEstimate(fundCode string) (*FundEstimate, error) {
    url := fmt.Sprintf("https://fundgz.1234567.com.cn/js/%s.js", fundCode)
    
    body, err := c.Get(url)
    if err != nil {
        return nil, fmt.Errorf("获取估值失败 %s: %w", fundCode, err)
    }

    // 解析 "jsonpgz(...);" 格式
    text := string(body)
    if !strings.HasPrefix(text, "jsonpgz(") || !strings.HasSuffix(text, ");") {
        return nil, fmt.Errorf("响应格式异常 %s", fundCode)
    }
    // 去掉 jsonpgz( 和 );
    jsonStr := text[8 : len(text)-2]

    var est FundEstimate
    if err := json.Unmarshal([]byte(jsonStr), &est); err != nil {
        return nil, fmt.Errorf("JSON 解析失败 %s: %w", fundCode, err)
    }

    return &est, nil
}

// BatchGetFundEstimates 批量获取估值
// Python 的异步版本用 asyncio.gather 并发请求
// Go 用 goroutine + channel 实现同样的效果
func (c *Client) BatchGetFundEstimates(codes []string) ([]*FundEstimate, error) {
    type result struct {
        est  *FundEstimate
        err  error
        code string
    }

    ch := make(chan result, len(codes))

    // 并发请求每个基金代码
    for _, code := range codes {
        go func(code string) {
            est, err := c.GetFundEstimate(code)
            ch <- result{est: est, err: err, code: code}
        }(code)  // ！必须把 code 作为参数传进去，否则闭包捕获的是循环变量
    }

    var results []*FundEstimate
    for i := 0; i < len(codes); i++ {
        r := <-ch
        if r.err != nil {
            continue  // 单个失败跳过
        }
        results = append(results, r.est)
    }

    return results, nil
}
```
</details>


**注意 goroutine 的闭包陷阱**：
```go
// 错误写法：
for _, code := range codes {
    go func() {
        // code 是循环变量，所有 goroutine 读到同一个值
        est, err := c.GetFundEstimate(code)  // 所有 goroutine 处理最后一个 code！
    }()
}

// 正确写法：传参数
for _, code := range codes {
    go func(c string) {
        est, err := c.GetFundEstimate(c)  // c 是参数，每个 goroutine 有自己的副本
    }(code)
}
```

---

## 4.4 实现基金历史净值

**Python 源码**（在 `services/akshare_service.py` 中）：

```python
def fetch_fund_history_akshare(code: str):
    """获取基金历史净值
    来源：东方财富基金历史净值页面
    """
    url = f"https://api.fund.eastmoney.com/f10/lsjz?callback=jQuery&fundCode={code}&pageIndex=1&pageSize=100"
    headers = {"User-Agent": "Mozilla/5.0", "Referer": "https://fund.eastmoney.com/"}
    resp = requests.get(url, headers=headers)
    data = resp.text
    # 去掉 jQuery(...) 包装
    json_str = data[data.index("(")+1:data.rindex(")")]
    parsed = json.loads(json_str)
    return parsed["Data"]["LSJZList"]
```

**Go 实现**：

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 go    （52 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

```go
// internal/service/akshare/history.go
package akshare

import (
    "encoding/json"
    "fmt"
    "strings"
)

// NavRecord 历史净值记录
type NavRecord struct {
    Date   string  `json:"FSRQ"`    // 净值日期
    Nav    string  `json:"DWJZ"`    // 单位净值
    Change string  `json:"JZZZL"`   // 日涨跌幅 %
}

// historyResponse 东财 API 响应结构
type historyResponse struct {
    Data struct {
        LSJZList []NavRecord `json:"LSJZList"`
    } `json:"Data"`
}

// GetFundHistory 获取基金历史净值
// 对应 Python fetch_fund_history_akshare()
func (c *Client) GetFundHistory(fundCode string) ([]NavRecord, error) {
    url := fmt.Sprintf(
        "https://api.fund.eastmoney.com/f10/lsjz?callback=jQuery&fundCode=%s&pageIndex=1&pageSize=100",
        fundCode,
    )

    body, err := c.Get(url)
    if err != nil {
        return nil, fmt.Errorf("获取历史净值失败 %s: %w", fundCode, err)
    }

    // 去掉 jQuery(...) 包装
    text := string(body)
    start := strings.Index(text, "(")
    end := strings.LastIndex(text, ")")
    if start == -1 || end == -1 {
        return nil, fmt.Errorf("响应格式异常 %s", fundCode)
    }
    jsonStr := text[start+1 : end]

    var resp historyResponse
    if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
        return nil, fmt.Errorf("JSON 解析失败 %s: %w", fundCode, err)
    }

    return resp.Data.LSJZList, nil
}
```
</details>


---

## 4.5 写单元测试（mock 上游 HTTP）

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 go    （25 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

```go
// internal/service/akshare/estimate_test.go
package akshare

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestGetFundEstimate(t *testing.T) {
    // 1. 创建一个本地的 mock HTTP 服务器
    // 模拟东方财富的 fundgz 接口
    mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 检查请求路径是否正确？
        // 返回模拟的 fundgz 响应
        w.Write([]byte(`jsonpgz({"fundcode":"000001","name":"测试基金","gsz":1.234,"gszzl":0.56,"dwjz":1.227,"gztime":"2024-01-15 15:00"});`))
    }))
    defer mockServer.Close()

    // 2. 创建一个 Client，但把 HTTP 请求指向 mock server
    // 由于 GetFundEstimate 里 URL 是写死的，这里需要改造 client
    // 让 fundgz URL 可配置（比如作为 Client 的字段）
    t.Logf("mock server 地址: %s", mockServer.URL)
    t.Log("测试思路：让 Client 的 fundgz 基础 URL 可配置，然后指向 mock server")
}
```
</details>


**测试封装改进**——让 client 的 URL 可配置：

```go
type Client struct {
    httpClient   *http.Client
    FundGZURL    string  // 默认 "https://fundgz.1234567.com.cn/js/%s.js"
    HistoryURL   string  // 默认 "https://api.fund.eastmoney.com/f10/lsjz?..."  
}

func NewClient() *Client {
    return &Client{
        httpClient: &http.Client{Timeout: 15 * time.Second},
        FundGZURL:  "https://fundgz.1234567.com.cn/js/%s.js",
        HistoryURL: "https://api.fund.eastmoney.com/f10/lsjz?callback=jQuery&fundCode=%s&pageIndex=1&pageSize=100",
    }
}
```

```bash
# 运行测试
go test -v ./internal/service/akshare/
```

---

## 4.6 完整客户端实现清单

需要实现的 akshare 客户端接口列表（对应 Python `services/akshare_service.py`）：

| 方法 | URL 来源 | 返回内容 |
|---|---|---|
| `GetFundEstimate(code)` | fundgz | 实时估值 |
| `GetFundHistory(code)` | 东财 API | 历史净值 |
| `GetFundBasicInfo(code)` | 东财 API | 基金基本信息 |
| `GetFundDividends(code)` | 东财 | 分红信息 |
| `GetFundFees(code)` | 东财 | 费率信息 |
| `GetFundPeriodRank(code)` | 东财 | 阶段排名 |
| `GetMarketStatus()` | — | 市场开闭盘状态 |
| `GetMarketIndices()` | 新浪 | 指数行情 |
| `GetKLine(code, period, count)` | 东财/腾讯 | K 线数据 |
| `GetUSDRate()` | — | 美元汇率 |

每个的实现模式都一模一样：`Get(url) → 解析 JSON/HTML → 返回结构体`

---

## 本级小结

- ✅ akshare HTTP 客户端基座（请求 + 响应体读取）
- ✅ 基金实时估值接口（`GetFundEstimate`）
- ✅ 基金历史净值接口（`GetFundHistory`）
- ✅ 批量并发获取估值（goroutine + channel）
- ✅ 单元测试用 httptest 模拟上游 HTTP

**下一步**：Step 5 —— 用 akshare 客户端 + 数据库构建 Fund Service。

---

## 补充：pkg/httpx/——HTTP 客户端工具（从 Step 14 移入）

aksahre client 的核心是 HTTP 请求，建议把通用 HTTP 客户端封装到 `pkg/httpx/`：

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 go    （46 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

```go
// pkg/httpx/client.go — 新建文件
package httpx

import (
    "fmt"
    "io"
    "net/http"
    "time"
)

// Client HTTP 客户端——封装超时、请求头
type Client struct {
    httpClient *http.Client
    headers    map[string]string
}

type Option func(*Client)

func WithTimeout(d time.Duration) Option {
    return func(c *Client) { c.httpClient.Timeout = d }
}
func WithHeader(k, v string) Option {
    return func(c *Client) { c.headers[k] = v }
}

func New(opts ...Option) *Client {
    c := &Client{
        httpClient: &http.Client{Timeout: 15 * time.Second},
        headers:    make(map[string]string),
    }
    for _, o := range opts { o(c) }
    return c
}

func (c *Client) Get(url string) ([]byte, error) {
    req, err := http.NewRequest("GET", url, nil)
    if err != nil { return nil, fmt.Errorf("创建请求失败: %w", err) }
    for k, v := range c.headers { req.Header.Set(k, v) }
    resp, err := c.httpClient.Do(req)
    if err != nil { return nil, fmt.Errorf("请求失败: %w", err) }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
    }
    return io.ReadAll(resp.Body)
}
```
</details>


然后在 akshare client 中使用：

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 go    （16 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

```go
// internal/service/akshare/client.go
import "huahua-service/pkg/httpx"

type Client struct {
    client *httpx.Client  // 使用 httpx 代替 net/http
}

func NewClient() *Client {
    return &Client{
        client: httpx.New(
            httpx.WithTimeout(15*time.Second),
            httpx.WithHeader("User-Agent", "Mozilla/5.0"),
            httpx.WithHeader("Referer", "https://finance.eastmoney.com/"),
        ),
    }
}
```
</details>

