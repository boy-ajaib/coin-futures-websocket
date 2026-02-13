package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"coin-futures-websocket/internal/types"
)

const (
	idrPerpSuffix = "IDR-PERP"
)

// TransformerInterface defines the interface for transforming Kafka message data
type TransformerInterface interface {
	TransformUserMargin(data []byte, cfxUserID string) ([]byte, error)
	TransformUserPosition(data []byte, cfxUserID string) ([]byte, error)
}

// Transformer provides data transformation capabilities for Kafka messages
type Transformer struct {
	currencyService CurrencyService
	cfxUsdtAsset    string
	logger          *slog.Logger
}

// NewTransformer creates a new Transformer
func NewTransformer(currencyService CurrencyService, cfxUsdtAsset string, logger *slog.Logger) *Transformer {
	return &Transformer{
		currencyService: currencyService,
		cfxUsdtAsset:    cfxUsdtAsset,
		logger:          logger,
	}
}

// TransformUserMargin transforms UserMargin data, converting USDT to IDR when needed
func (t *Transformer) TransformUserMargin(data []byte, cfxUserID string) ([]byte, error) {
	var margin types.UserMargin
	if err := json.Unmarshal(data, &margin); err != nil {
		return nil, fmt.Errorf("failed to unmarshal UserMargin: %w", err)
	}

	// Only transform when asset is IDR (values are USDT but labeled as IDR)
	if margin.Asset != "IDR" {
		t.logger.Debug("skipping margin transformation, asset is not IDR (values already in correct currency)",
			"cfx_user_id", cfxUserID,
			"asset", margin.Asset)
		return data, nil
	}

	ctx := context.Background()
	rate, err := t.currencyService.GetCurrentRate(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange rate: %w", err)
	}

	// Convert the currency fields (USDT -> IDR)
	margin.TotalPositionValue = margin.TotalPositionValue * rate
	margin.MarginBalance = margin.MarginBalance * rate
	margin.OrderMargin = margin.OrderMargin * rate
	margin.MaintenanceMargin = margin.MaintenanceMargin * rate
	margin.UnrealizedPnl = margin.UnrealizedPnl * rate
	margin.AvailableMargin = margin.AvailableMargin * rate
	margin.WalletBalance = margin.WalletBalance * rate
	margin.WithdrawableMargin = margin.WithdrawableMargin * rate

	transformedData, err := json.Marshal(margin)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transformed UserMargin: %w", err)
	}

	t.logger.Debug("transformed user margin to IDR",
		"cfx_user_id", cfxUserID,
		"asset", margin.Asset,
		"rate", rate)

	return transformedData, nil
}

// TransformUserPosition transforms UserPosition data, converting USDT to IDR when needed
func (t *Transformer) TransformUserPosition(data []byte, cfxUserID string) ([]byte, error) {
	var position types.UserPosition
	if err := json.Unmarshal(data, &position); err != nil {
		return nil, fmt.Errorf("failed to unmarshal UserPosition: %w", err)
	}

	// Only transform when symbol ends with IDR-PERP (values are USDT-PERP but labeled as IDR-PERP)
	if !strings.HasSuffix(position.Symbol, idrPerpSuffix) {
		t.logger.Debug("skipping position transformation, not an IDR-PERP symbol (values already in correct currency)",
			"cfx_user_id", cfxUserID,
			"symbol", position.Symbol)
		return data, nil
	}

	ctx := context.Background()
	rate, err := t.currencyService.GetCurrentRate(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange rate: %w", err)
	}

	// Convert the currency fields (USDT -> IDR)
	position.Value = position.Value * rate
	position.MaintenanceMargin = position.MaintenanceMargin * rate
	position.RealisedPnl = position.RealisedPnl * rate
	position.UnrealisedPnl = position.UnrealisedPnl * rate
	position.OrderMargin = position.OrderMargin * rate

	transformedData, err := json.Marshal(position)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transformed UserPosition: %w", err)
	}

	t.logger.Debug("transformed user position to IDR",
		"cfx_user_id", cfxUserID,
		"symbol", position.Symbol,
		"rate", rate)

	return transformedData, nil
}
