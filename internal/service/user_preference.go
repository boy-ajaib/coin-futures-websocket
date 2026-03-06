package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"coin-futures-websocket/internal/cache"
)

// UserPreferenceClient defines the interface for fetching user futures preference
type UserPreferenceClient interface {
	GetQuotePreference(ctx context.Context, ajaibID string) (string, error)
}

// HTTPUserPreferenceClient fetches user futures preference from coin-setting-svc
type HTTPUserPreferenceClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
	cache      *cache.TTLCache[string]
}

// NewHTTPUserPreferenceClient creates a new user preference client
func NewHTTPUserPreferenceClient(baseURL string, cacheTTL time.Duration, logger *slog.Logger) *HTTPUserPreferenceClient {
	return &HTTPUserPreferenceClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		logger: logger,
		cache:  cache.NewTTLCache[string](cacheTTL),
	}
}

// UserPreferenceResponse represents the API response from coin-setting-svc
type UserPreferenceResponse struct {
	ErrCode    string               `json:"err_code"`
	ErrMessage string               `json:"err_message"`
	Result     UserPreferenceResult `json:"result"`
}

// UserPreferenceResult contains the preference data
type UserPreferenceResult struct {
	QuotePreference string `json:"quote_preference"`
}

// GetQuotePreference retrieves the user's futures quote preference
func (c *HTTPUserPreferenceClient) GetQuotePreference(ctx context.Context, ajaibID string) (string, error) {
	if cached, ok := c.cache.Get(ajaibID); ok {
		c.logger.Debug("user preference cache hit", "ajaib_id", ajaibID)
		return cached, nil
	}

	url := fmt.Sprintf("%s/api/v1/internal/coin-setting/user-futures-preference", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Id", ajaibID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("failed to fetch user preference",
			"ajaib_id", ajaibID,
			"error", err)
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response UserPreferenceResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if response.ErrCode != "EC0000000" {
		return "", fmt.Errorf("API error: %s - %s", response.ErrCode, response.ErrMessage)
	}

	if response.Result.QuotePreference == "" {
		return "", fmt.Errorf("quote preference not found for ajaib_id: %s", ajaibID)
	}

	pref := response.Result.QuotePreference
	c.cache.Set(ajaibID, pref)

	c.logger.Debug("fetched user quote preference",
		"ajaib_id", ajaibID,
		"quote_preference", pref)

	return pref, nil
}
