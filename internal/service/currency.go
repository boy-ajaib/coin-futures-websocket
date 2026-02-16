package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// CurrencyService defines the interface for currency conversion operations
type CurrencyService interface {
	GetCurrentRate(ctx context.Context) (float64, error)
}

// CachedCurrencyService implements CurrencyService with a background scheduler that periodically refreshes the exchange rate
type CachedCurrencyService struct {
	rateProvider RateProvider
	rate         float64
	mu           sync.RWMutex
	logger       *slog.Logger
	stop         chan struct{}
}

// NewCachedCurrencyService creates a CurrencyService that refreshes the exchange rate in the background at the given interval
func NewCachedCurrencyService(rateProvider RateProvider, refreshInterval time.Duration, logger *slog.Logger) *CachedCurrencyService {
	s := &CachedCurrencyService{
		rateProvider: rateProvider,
		logger:       logger,
		stop:         make(chan struct{}),
	}

	s.refresh()

	go s.run(refreshInterval)

	return s
}

// run is the background loop that refreshes the rate on a fixed schedule
func (s *CachedCurrencyService) run(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.refresh()
		case <-s.stop:
			return
		}
	}
}

// refresh fetches the latest rate from the provider and updates the cache
func (s *CachedCurrencyService) refresh() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rate, err := s.rateProvider.GetUSDTToIDRRate(ctx)
	if err != nil {
		s.logger.Warn("failed to refresh exchange rate, using last known rate", "error", err)
		return
	}

	s.mu.Lock()
	s.rate = rate
	s.mu.Unlock()

	s.logger.Info("refreshed exchange rate", "rate", rate)
}

// GetCurrentRate returns the latest cached exchange rate
func (s *CachedCurrencyService) GetCurrentRate(ctx context.Context) (float64, error) {
	s.mu.RLock()
	rate := s.rate
	s.mu.RUnlock()

	if rate == 0 {
		return 0, fmt.Errorf("no exchange rate available")
	}

	return rate, nil
}

// Stop shuts down the background refresh goroutine
func (s *CachedCurrencyService) Stop() {
	close(s.stop)
}
