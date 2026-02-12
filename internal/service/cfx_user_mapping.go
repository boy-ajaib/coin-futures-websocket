package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// CfxUserMappingClient defines the interface for mapping Ajaib user IDs to CFX user IDs
type CfxUserMappingClient interface {
	GetCfxUserID(ctx context.Context, ajaibID int64) (string, error)
}

// HTTPCfxUserMappingClient fetches CFX user ID from coin-cfx-adapter API
type HTTPCfxUserMappingClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewHTTPCfxUserMappingClient creates a new CFX user mapping client
func NewHTTPCfxUserMappingClient(baseURL string, logger *slog.Logger) *HTTPCfxUserMappingClient {
	return &HTTPCfxUserMappingClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		logger: logger,
	}
}

// CfxMappingResponse represents the API response from coin-cfx-adapter
type CfxMappingResponse struct {
	ErrCode    string           `json:"err_code"`
	ErrMessage string           `json:"err_message"`
	Result     CfxMappingResult `json:"result"`
}

// CfxMappingResult contains the mapping data
type CfxMappingResult struct {
	AjaibID   int64  `json:"ajaib_id"`
	CfxUserID string `json:"cfx_user_id"`
}

// GetCfxUserID retrieves the CFX user ID for a given Ajaib user ID
func (c *HTTPCfxUserMappingClient) GetCfxUserID(ctx context.Context, ajaibID int64) (string, error) {
	url := fmt.Sprintf("%s/api/v1/internal/coin-cfx-adapter/user/%d/cfx", c.baseURL, ajaibID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("failed to fetch CFX user mapping",
			"ajaib_id", ajaibID,
			"error", err)
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response CfxMappingResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if response.ErrCode != "EC0000000" {
		return "", fmt.Errorf("API error: %s - %s", response.ErrCode, response.ErrMessage)
	}

	if response.Result.CfxUserID == "" {
		return "", fmt.Errorf("CFX user ID not found for ajaib_id: %d", ajaibID)
	}

	c.logger.Debug("mapped ajaib_id to cfx_user_id",
		"ajaib_id", ajaibID,
		"cfx_user_id", response.Result.CfxUserID)

	return response.Result.CfxUserID, nil
}
