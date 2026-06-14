package akshare

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "huahua-service/akshare_proto/akshare"
)

// Client wraps the gRPC connection to the Python akshare server.
// Create one instance per application lifecycle.
type Client struct {
	conn   *grpc.ClientConn
	client pb.AkshareClient
}

// NewClient connects to the Python akshare gRPC server.
// addr is typically "localhost:9800".
func NewClient(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(50*1024*1024)), // 50MB
	)
	if err != nil {
		return nil, fmt.Errorf("连接 akshare gRPC 失败: %w", err)
	}

	return &Client{
		conn:   conn,
		client: pb.NewAkshareClient(conn),
	}, nil
}

// Close releases the gRPC connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// Call invokes an arbitrary akshare function by name.
// fn: akshare function name, e.g. "stock_zh_a_hist"
// kwargs: keyword arguments passed to the function.
// Returns the raw JSON result as bytes.
func (c *Client) Call(fn string, kwargs map[string]string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := c.client.Call(ctx, &pb.CallRequest{
		Fn:     fn,
		Kwargs: kwargs,
	})
	if err != nil {
		return nil, fmt.Errorf("akshare.Call(%s): %w", fn, err)
	}
	if !resp.Ok {
		return nil, fmt.Errorf("akshare.Call(%s) error: %s", fn, resp.Error)
	}
	return []byte(resp.DataJson), nil
}

// BatchGetFundEstimates fetches real-time estimates for multiple funds.
func (c *Client) BatchGetFundEstimates(codes []string) ([]*pb.FundEstimate, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := c.client.BatchGetFundEstimates(ctx, &pb.BatchEstimateRequest{
		Codes: codes,
	})
	if err != nil {
		return nil, fmt.Errorf("BatchGetFundEstimates: %w", err)
	}
	return resp.Estimates, nil
}

// GetFundHistory fetches historical NAV records for a single fund.
func (c *Client) GetFundHistory(code string) ([]*pb.NavRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := c.client.GetFundHistory(ctx, &pb.HistoryRequest{
		Code: code,
	})
	if err != nil {
		return nil, fmt.Errorf("GetFundHistory(%s): %w", code, err)
	}
	return resp.Records, nil
}

// CallGeneric is a convenience wrapper: calls an akshare function and
// unmarshals the JSON result into the provided target.
// target must be a pointer.
func (c *Client) CallGeneric(fn string, kwargs map[string]string, target any) error {
	data, err := c.Call(fn, kwargs)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}
