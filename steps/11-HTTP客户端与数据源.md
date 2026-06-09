# Step 11：HTTP 客户端与数据源

> **目标**：封装外部 HTTP 请求客户端，用于获取实时数据。
> 参考 `steps-v1/04-核心数据源封装akshare.md`。

---

## 11.1 为什么需要独立的 HTTP 客户端

项目需要从外部 API（东方财富、新浪财经等）获取数据。
Go 的 `net/http` 可以直接用，但每次都要重复设置超时、请求头、错误处理，很啰嗦。

封装到 `pkg/httpx/` 复用：

```go
// pkg/httpx/client.go
package httpx

import (
    "fmt"
    "io"
    "net/http"
    "time"
)

type Client struct {
    httpClient *http.Client
    headers    map[string]string
}

func New(timeout time.Duration) *Client {
    return &Client{
        httpClient: &http.Client{Timeout: timeout},
        headers:    make(map[string]string),
    }
}

func (c *Client) SetHeader(k, v string) { c.headers[k] = v }

func (c *Client) Get(url string) ([]byte, error) {
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("请求失败: %w", err)
    }
    for k, v := range c.headers {
        req.Header.Set(k, v)
    }
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
    }
    return io.ReadAll(resp.Body)
}
```

## 11.2 Akshare 数据源客户端

```go
// internal/service/akshare/client.go
package akshare

import (
    "encoding/json"
    "fmt"
    "strings"
    "time"
    "huahua-service/pkg/httpx"
)

type Client struct {
    http *httpx.Client
}

func NewClient() *Client {
    c := httpx.New(15 * time.Second)
    c.SetHeader("User-Agent", "Mozilla/5.0")
    c.SetHeader("Referer", "https://finance.eastmoney.com/")
    return &Client{http: c}
}

// GetFundEstimate 获取基金实时估值
func (c *Client) GetFundEstimate(code string) (*FundEstimate, error) {
    url := fmt.Sprintf("https://fundgz.1234567.com.cn/js/%s.js", code)
    body, err := c.http.Get(url)
    if err != nil {
        return nil, err
    }
    text := string(body)
    jsonStr := text[8 : len(text)-2] // 去掉 jsonpgz( 和 );
    var est FundEstimate
    json.Unmarshal([]byte(jsonStr), &est)
    return &est, nil
}

type FundEstimate struct {
    FundCode        string  `json:"fundcode"`
    Name            string  `json:"name"`
    Estimate        float64 `json:"gsz"`
    EstimatePercent float64 `json:"gszzl"`
    LastNav         float64 `json:"dwjz"`
    EstimateTime    string  `json:"gztime"`
}

// GetFundHistory 获取历史净值
func (c *Client) GetFundHistory(code string) ([]NavRecord, error) {
    url := fmt.Sprintf("https://api.fund.eastmoney.com/f10/lsjz?callback=jQuery&fundCode=%s&pageIndex=1&pageSize=100", code)
    body, err := c.http.Get(url)
    if err != nil {
        return nil, err
    }
    // 解析 jQuery(...) 包装的 JSON
    text := string(body)
    start, end := strings.Index(text, "("), strings.LastIndex(text, ")")
    var resp struct {
        Data struct {
            LSJZList []NavRecord `json:"LSJZList"`
        } `json:"Data"`
    }
    json.Unmarshal([]byte(text[start+1:end]), &resp)
    return resp.Data.LSJZList, nil
}

type NavRecord struct {
    Date   string `json:"FSRQ"`
    Nav    string `json:"DWJZ"`
    Change string `json:"JZZZL"`
}
```

---

## 本步总结

- ✅ pkg/httpx 通用 HTTP 客户端
- ✅ akshare 客户端（实时估值 + 历史净值）
- ✅ 代码简洁，一次封装到处用
