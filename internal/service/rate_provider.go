package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// RateProvider defines the interface for fetching exchange rates
type RateProvider interface {
	GetUSDTToIDRRate(ctx context.Context) (float64, error)
}

// baseResponse represents the base API response wrapper from Coin Data API
type baseResponse struct {
	Result priceRateDto `json:"result"`
}

// priceRateDto represents the exchange rate data from Coin Data API
type priceRateDto struct {
	BaseCurrency  string          `json:"base_currency"`
	QuoteCurrency string          `json:"quote_currency"`
	Amount        float64         `json:"amount"`
	UpdatedAt     json.RawMessage `json:"updated_at"`
}

// HTTPRateProvider implements RateProvider using HTTP requests to an external API
type HTTPRateProvider struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewHTTPRateProvider creates a new HTTPRateProvider
func NewHTTPRateProvider(baseURL string, logger *slog.Logger) *HTTPRateProvider {
	return &HTTPRateProvider{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

// GetUSDTToIDRRate fetches the current USDT to IDR exchange rate from the configured API
func (p *HTTPRateProvider) GetUSDTToIDRRate(ctx context.Context) (float64, error) {
	url := fmt.Sprintf("%s/api/v1/coin-data/futures-exchange-rate/USDT/IDR", p.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch rate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var baseResp baseResponse
	if err := json.NewDecoder(resp.Body).Decode(&baseResp); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	rate := baseResp.Result.Amount
	if rate <= 0 {
		return 0, fmt.Errorf("invalid rate received: %f", rate)
	}

	p.logger.Debug("fetched USDT to IDR rate",
		"rate", rate,
		"base", baseResp.Result.BaseCurrency,
		"quote", baseResp.Result.QuoteCurrency)

	return rate, nil
}
