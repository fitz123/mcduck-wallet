package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/fitz123/mcduck-wallet/internal/database"
	"github.com/fitz123/mcduck-wallet/internal/logger"
)

type ExchangeService interface {
	// GetRate returns the exchange rate from one currency to another
	GetRate(ctx context.Context, from, to *database.Currency) (float64, error)
	// Convert calculates the amount in target currency
	Convert(ctx context.Context, from, to *database.Currency, amount float64) (float64, error)
}

type exchangeService struct {
	httpClient *http.Client
	cache      *rateCache
}

type rateCache struct {
	mu        sync.RWMutex
	rates     map[string]float64 // key: "USD_EUR", value: rate
	fetchedAt time.Time
	ttl       time.Duration
}

type exchangeRateAPIResponse struct {
	Result   string             `json:"result"`
	BaseCode string             `json:"base_code"`
	Rates    map[string]float64 `json:"rates"`
}

func NewExchangeService() ExchangeService {
	return &exchangeService{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		cache: &rateCache{
			rates: make(map[string]float64),
			ttl:   2 * time.Hour,
		},
	}
}

func (s *exchangeService) GetRate(ctx context.Context, from, to *database.Currency) (float64, error) {
	if from.Code == to.Code {
		return 1.0, nil
	}

	// Both real currencies - use Frankfurter
	if from.IsReal && to.IsReal {
		return s.getRealRate(ctx, from.Code, to.Code)
	}

	// Both made-up currencies - use fixed rates
	// FixedRate = units per 1 USD, so to get "how many TO per 1 FROM":
	// rate = to.FixedRate / from.FixedRate
	if !from.IsReal && !to.IsReal {
		if from.FixedRate == 0 {
			return 0, fmt.Errorf("currency %s has no fixed rate set", from.Code)
		}
		return to.FixedRate / from.FixedRate, nil
	}

	// Mixed: real -> made-up
	if from.IsReal && !to.IsReal {
		// Get rate from real currency to USD first
		rateToUSD := 1.0
		if from.Code != "USD" {
			var err error
			rateToUSD, err = s.getRealRate(ctx, from.Code, "USD")
			if err != nil {
				return 0, err
			}
		}
		// Then apply made-up currency's fixed rate
		return rateToUSD * to.FixedRate, nil
	}

	// Mixed: made-up -> real
	if !from.IsReal && to.IsReal {
		if from.FixedRate == 0 {
			return 0, fmt.Errorf("currency %s has no fixed rate set", from.Code)
		}
		// Convert to USD equivalent first
		usdAmount := 1.0 / from.FixedRate
		// Then get rate from USD to target real currency
		if to.Code == "USD" {
			return usdAmount, nil
		}
		rateFromUSD, err := s.getRealRate(ctx, "USD", to.Code)
		if err != nil {
			return 0, err
		}
		return usdAmount * rateFromUSD, nil
	}

	return 0, fmt.Errorf("unable to calculate rate between %s and %s", from.Code, to.Code)
}

func (s *exchangeService) Convert(ctx context.Context, from, to *database.Currency, amount float64) (float64, error) {
	rate, err := s.GetRate(ctx, from, to)
	if err != nil {
		return 0, err
	}
	return amount * rate, nil
}

func (s *exchangeService) getRealRate(ctx context.Context, fromCode, toCode string) (float64, error) {
	cacheKey := fromCode + "_" + toCode

	// Check cache first
	s.cache.mu.RLock()
	if time.Since(s.cache.fetchedAt) < s.cache.ttl {
		if rate, ok := s.cache.rates[cacheKey]; ok {
			s.cache.mu.RUnlock()
			return rate, nil
		}
	}
	s.cache.mu.RUnlock()

	// Fetch from ExchangeRate-API
	rate, err := s.fetchFromExchangeRateAPI(ctx, fromCode, toCode)
	if err != nil {
		// Try to use stale cache on error
		s.cache.mu.RLock()
		if rate, ok := s.cache.rates[cacheKey]; ok {
			s.cache.mu.RUnlock()
			logger.Error("ExchangeRate API error, using stale cache", "error", err)
			return rate, nil
		}
		s.cache.mu.RUnlock()
		return 0, err
	}

	// Update cache
	s.cache.mu.Lock()
	s.cache.rates[cacheKey] = rate
	s.cache.fetchedAt = time.Now()
	s.cache.mu.Unlock()

	return rate, nil
}

func (s *exchangeService) fetchFromExchangeRateAPI(ctx context.Context, fromCode, toCode string) (float64, error) {
	// ExchangeRate-API returns rates relative to base currency
	url := fmt.Sprintf("https://open.er-api.com/v6/latest/%s", fromCode)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch exchange rate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("exchange rate API returned status %d", resp.StatusCode)
	}

	var result exchangeRateAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Result != "success" {
		return 0, fmt.Errorf("exchange rate API error: %s", result.Result)
	}

	rate, ok := result.Rates[toCode]
	if !ok {
		return 0, fmt.Errorf("rate for %s not found in response", toCode)
	}

	return rate, nil
}
