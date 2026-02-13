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

// rateCache holds a cached exchange rate with expiration
type rateCache struct {
	rate      float64
	timestamp time.Time
	mu        sync.RWMutex
	ttl       time.Duration
}

// newRateCache creates a new rate cache with the specified TTL
func newRateCache(ttl time.Duration) *rateCache {
	return &rateCache{
		ttl: ttl,
	}
}

// get returns the cached rate if it's still valid, otherwise returns an error
func (c *rateCache) get() (float64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.rate == 0 {
		return 0, fmt.Errorf("no cached rate available")
	}

	if time.Since(c.timestamp) > c.ttl {
		return 0, fmt.Errorf("cached rate has expired")
	}

	return c.rate, nil
}

// set stores a new rate in the cache
func (c *rateCache) set(rate float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rate = rate
	c.timestamp = time.Now()
}

// isExpired checks if the cached rate has expired
func (c *rateCache) isExpired() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.rate == 0 || time.Since(c.timestamp) > c.ttl
}

// cachedCurrencyService implements CurrencyService with rate caching
type cachedCurrencyService struct {
	rateProvider RateProvider
	cache        *rateCache
	logger       *slog.Logger
	mu           sync.RWMutex
}

// NewCachedCurrencyService creates a new CurrencyService with rate caching
func NewCachedCurrencyService(rateProvider RateProvider, cacheTTL time.Duration, logger *slog.Logger) CurrencyService {
	return &cachedCurrencyService{
		rateProvider: rateProvider,
		cache:        newRateCache(cacheTTL),
		logger:       logger,
	}
}

// GetCurrentRate returns the current exchange rate, fetching a new one if the cache has expired
func (s *cachedCurrencyService) GetCurrentRate(ctx context.Context) (float64, error) {
	if !s.cache.isExpired() {
		if rate, err := s.cache.get(); err == nil {
			s.logger.Debug("using cached exchange rate", "rate", rate)
			return rate, nil
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.cache.isExpired() {
		if rate, err := s.cache.get(); err == nil {
			return rate, nil
		}
	}

	s.logger.Debug("fetching new exchange rate from provider")

	rate, err := s.rateProvider.GetUSDTToIDRRate(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch rate from provider: %w", err)
	}

	s.cache.set(rate)
	s.logger.Info("updated exchange rate cache", "rate", rate, "ttl", s.cache.ttl)

	return rate, nil
}
