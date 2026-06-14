package fund

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"huahua-service/internal/schema"
	akshareSvc "huahua-service/internal/service/akshare"
)

var ErrAkshareUnavailable = errors.New("akshare 服务不可用")

type Service struct {
	akshare *akshareSvc.Client
}

func NewService(akshare *akshareSvc.Client) *Service {
	return &Service{akshare: akshare}
}

func (s *Service) requireAkshare() error {
	if s.akshare == nil {
		return ErrAkshareUnavailable
	}
	return nil
}

// GetEstimates 批量获取基金估值
func (s *Service) GetEstimates(_ context.Context, codes []string) ([]schema.EstimateItem, error) {
	if err := s.requireAkshare(); err != nil {
		return nil, err
	}

	estimates, err := s.akshare.BatchGetFundEstimates(codes)
	if err != nil {
		return nil, err
	}

	results := make([]schema.EstimateItem, 0, len(estimates))
	for _, e := range estimates {
		results = append(results, schema.EstimateItem{
			FundCode:      e.FundCode,
			Name:          e.Name,
			Estimate:      e.Estimate,
			ChangePercent: e.EstimatePercent,
			Time:          e.EstimateTime,
		})
	}
	return results, nil
}

// GetHistory 获取基金历史净值
func (s *Service) GetHistory(_ context.Context, code string) ([]schema.NavItem, error) {
	if err := s.requireAkshare(); err != nil {
		return nil, err
	}

	records, err := s.akshare.GetFundHistory(code)
	if err != nil {
		return nil, err
	}

	items := make([]schema.NavItem, 0, len(records))
	for _, r := range records {
		items = append(items, schema.NavItem{
			Date:   r.Date,
			Nav:    r.Nav,
			Change: r.Change,
		})
	}
	return items, nil
}

// GetDividends 获取基金分红信息
func (s *Service) GetDividends(_ context.Context, code string) (json.RawMessage, error) {
	if err := s.requireAkshare(); err != nil {
		return nil, err
	}
	return s.akshare.Call("fund_announcement_dividend_em", map[string]string{
		"symbol": code,
	})
}

// GetFees 获取基金费率信息
func (s *Service) GetFees(_ context.Context, code string) (json.RawMessage, error) {
	if err := s.requireAkshare(); err != nil {
		return nil, err
	}
	return s.akshare.Call("fund_fee_em", map[string]string{
		"symbol": code,
	})
}

// GetDetail 获取基金详情（基本信息 + 估值 + 最近净值）
func (s *Service) GetDetail(ctx context.Context, code string) (map[string]any, error) {
	if err := s.requireAkshare(); err != nil {
		return nil, err
	}

	var basic []map[string]any
	if err := s.akshare.CallGeneric("fund_individual_basic_info_xq", map[string]string{
		"symbol": code,
	}, &basic); err != nil {
		return nil, fmt.Errorf("获取基金基本信息失败: %w", err)
	}

	estimates, err := s.GetEstimates(ctx, []string{code})
	if err != nil {
		return nil, err
	}

	history, err := s.GetHistory(ctx, code)
	if err != nil {
		return nil, err
	}

	recent := history
	if len(recent) > 30 {
		recent = recent[len(recent)-30:]
	}

	result := map[string]any{
		"code":     code,
		"name":     "",
		"type":     "",
		"estimate": 0.0,
		"history":  recent,
	}
	if len(basic) > 0 {
		if name, ok := basic[0]["基金简称"].(string); ok {
			result["name"] = name
		}
		if typ, ok := basic[0]["基金类型"].(string); ok {
			result["type"] = typ
		}
	}
	if len(estimates) > 0 {
		result["estimate"] = estimates[0].Estimate
		result["changePercent"] = estimates[0].ChangePercent
		if estimates[0].Name != "" {
			result["name"] = estimates[0].Name
		}
	}
	return result, nil
}

// SearchFunds 按关键字搜索基金
func (s *Service) SearchFunds(_ context.Context, key string) ([]schema.FundSearchItem, error) {
	if err := s.requireAkshare(); err != nil {
		return nil, err
	}
	if len(key) > 100 {
		return []schema.FundSearchItem{}, nil
	}

	var all []map[string]any
	if err := s.akshare.CallGeneric("fund_name_em", map[string]string{}, &all); err != nil {
		return nil, err
	}

	keyUpper := strings.ToUpper(strings.TrimSpace(key))
	results := make([]schema.FundSearchItem, 0, 20)
	for _, row := range all {
		code, _ := row["基金代码"].(string)
		name, _ := row["基金简称"].(string)
		typ, _ := row["基金类型"].(string)
		if keyUpper != "" &&
			!strings.Contains(strings.ToUpper(code), keyUpper) &&
			!strings.Contains(strings.ToUpper(name), keyUpper) {
			continue
		}
		results = append(results, schema.FundSearchItem{
			Code: code,
			Name: name,
			Type: typ,
		})
		if len(results) >= 20 {
			break
		}
	}
	return results, nil
}
