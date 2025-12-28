package services

import (
	"context"
	"testing"

	"github.com/fitz123/mcduck-wallet/internal/database"
	"github.com/fitz123/mcduck-wallet/internal/logger"
)

func init() {
	logger.Init("debug")
}

func TestExchangeService_GetRate(t *testing.T) {
	svc := NewExchangeService()
	ctx := context.Background()

	t.Run("same currency returns 1.0", func(t *testing.T) {
		usd := &database.Currency{Code: "USD", IsReal: true}
		rate, err := svc.GetRate(ctx, usd, usd)
		if err != nil {
			t.Fatalf("GetRate() error = %v", err)
		}
		if rate != 1.0 {
			t.Errorf("GetRate() = %v, want 1.0", rate)
		}
	})

	t.Run("both made-up currencies uses fixed rates", func(t *testing.T) {
		shl := &database.Currency{Code: "SHL", IsReal: false, FixedRate: 10.0}  // 10 SHL = 1 USD
		gold := &database.Currency{Code: "GOLD", IsReal: false, FixedRate: 0.1} // 0.1 GOLD = 1 USD

		// SHL -> GOLD: to.FixedRate / from.FixedRate = 0.1 / 10 = 0.01
		// 1 SHL = 0.01 GOLD
		rate, err := svc.GetRate(ctx, shl, gold)
		if err != nil {
			t.Fatalf("GetRate() error = %v", err)
		}
		if rate != 0.01 {
			t.Errorf("GetRate() SHL->GOLD = %v, want 0.01", rate)
		}

		// GOLD -> SHL: to.FixedRate / from.FixedRate = 10 / 0.1 = 100
		// 1 GOLD = 100 SHL
		rate, err = svc.GetRate(ctx, gold, shl)
		if err != nil {
			t.Fatalf("GetRate() error = %v", err)
		}
		if rate != 100 {
			t.Errorf("GetRate() GOLD->SHL = %v, want 100", rate)
		}
	})

	t.Run("made-up currency with zero fixed rate returns error", func(t *testing.T) {
		shl := &database.Currency{Code: "SHL", IsReal: false, FixedRate: 0}
		gold := &database.Currency{Code: "GOLD", IsReal: false, FixedRate: 10}

		_, err := svc.GetRate(ctx, shl, gold)
		if err == nil {
			t.Error("GetRate() should error for zero fixed rate")
		}
	})

	t.Run("made-up to USD", func(t *testing.T) {
		shl := &database.Currency{Code: "SHL", IsReal: false, FixedRate: 10.0} // 10 SHL = 1 USD
		usd := &database.Currency{Code: "USD", IsReal: true}

		// SHL -> USD: 1 SHL = 0.1 USD (1/10)
		rate, err := svc.GetRate(ctx, shl, usd)
		if err != nil {
			t.Fatalf("GetRate() error = %v", err)
		}
		if rate != 0.1 {
			t.Errorf("GetRate() SHL->USD = %v, want 0.1", rate)
		}
	})

	t.Run("USD to made-up", func(t *testing.T) {
		usd := &database.Currency{Code: "USD", IsReal: true}
		shl := &database.Currency{Code: "SHL", IsReal: false, FixedRate: 10.0} // 10 SHL = 1 USD

		// USD -> SHL: 1 USD = 10 SHL
		rate, err := svc.GetRate(ctx, usd, shl)
		if err != nil {
			t.Fatalf("GetRate() error = %v", err)
		}
		if rate != 10.0 {
			t.Errorf("GetRate() USD->SHL = %v, want 10.0", rate)
		}
	})
}

func TestExchangeService_Convert(t *testing.T) {
	svc := NewExchangeService()
	ctx := context.Background()

	t.Run("converts between made-up currencies", func(t *testing.T) {
		shl := &database.Currency{Code: "SHL", IsReal: false, FixedRate: 10.0}
		gold := &database.Currency{Code: "GOLD", IsReal: false, FixedRate: 0.1}

		// 100 SHL = ? GOLD
		// Rate: 0.01 (1 SHL = 0.01 GOLD)
		// Result: 100 * 0.01 = 1 GOLD
		result, err := svc.Convert(ctx, shl, gold, 100)
		if err != nil {
			t.Fatalf("Convert() error = %v", err)
		}
		if result != 1.0 {
			t.Errorf("Convert() 100 SHL = %v GOLD, want 1.0", result)
		}
	})

	t.Run("converts USD to made-up", func(t *testing.T) {
		usd := &database.Currency{Code: "USD", IsReal: true}
		shl := &database.Currency{Code: "SHL", IsReal: false, FixedRate: 10.0}

		// 5 USD = ? SHL
		// Rate: 10 (1 USD = 10 SHL)
		// Result: 5 * 10 = 50 SHL
		result, err := svc.Convert(ctx, usd, shl, 5)
		if err != nil {
			t.Fatalf("Convert() error = %v", err)
		}
		if result != 50.0 {
			t.Errorf("Convert() 5 USD = %v SHL, want 50.0", result)
		}
	})

	t.Run("converts made-up to USD", func(t *testing.T) {
		shl := &database.Currency{Code: "SHL", IsReal: false, FixedRate: 10.0}
		usd := &database.Currency{Code: "USD", IsReal: true}

		// 100 SHL = ? USD
		// Rate: 0.1 (1 SHL = 0.1 USD)
		// Result: 100 * 0.1 = 10 USD
		result, err := svc.Convert(ctx, shl, usd, 100)
		if err != nil {
			t.Fatalf("Convert() error = %v", err)
		}
		if result != 10.0 {
			t.Errorf("Convert() 100 SHL = %v USD, want 10.0", result)
		}
	})
}
