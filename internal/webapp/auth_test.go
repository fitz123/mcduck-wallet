package webapp

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"

	"github.com/fitz123/mcduck-wallet/internal/logger"
)

func init() {
	logger.Init("debug")
}

const testBotToken = "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"

func TestAuthService_AuthMiddleware(t *testing.T) {
	authService := NewAuthService(testBotToken)

	t.Run("rejects request without init data header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/dashboard", nil)
		rr := httptest.NewRecorder()

		handler := authService.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called")
		}))

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Status = %v, want %v", rr.Code, http.StatusUnauthorized)
		}
		if !strings.Contains(rr.Body.String(), "Unauthorized") {
			t.Errorf("Body should contain 'Unauthorized'")
		}
	})

	t.Run("rejects request with invalid init data", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/dashboard", nil)
		req.Header.Set("X-Telegram-Init-Data", "invalid_data")
		rr := httptest.NewRecorder()

		handler := authService.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called")
		}))

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Status = %v, want %v", rr.Code, http.StatusUnauthorized)
		}
	})

	t.Run("accepts valid init data and sets user ID in context", func(t *testing.T) {
		initData := generateValidInitData(t, testBotToken, 12345, "testuser")
		req := httptest.NewRequest("GET", "/dashboard", nil)
		req.Header.Set("X-Telegram-Init-Data", initData)
		rr := httptest.NewRecorder()

		var capturedUserID int64
		handler := authService.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedUserID = GetUserIDFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Status = %v, want %v", rr.Code, http.StatusOK)
		}
		if capturedUserID != 12345 {
			t.Errorf("UserID = %v, want 12345", capturedUserID)
		}
	})

	t.Run("rejects tampered hash", func(t *testing.T) {
		initData := generateValidInitData(t, testBotToken, 12345, "testuser")
		// Tamper with hash
		initData = strings.Replace(initData, "hash=", "hash=tampered", 1)

		req := httptest.NewRequest("GET", "/dashboard", nil)
		req.Header.Set("X-Telegram-Init-Data", initData)
		rr := httptest.NewRecorder()

		handler := authService.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called")
		}))

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Status = %v, want %v", rr.Code, http.StatusUnauthorized)
		}
	})
}

func TestAuthService_validateInitData(t *testing.T) {
	authService := NewAuthService(testBotToken)

	t.Run("validates correct signature", func(t *testing.T) {
		initData := generateValidInitData(t, testBotToken, 12345, "testuser")
		if !authService.validateInitData(initData) {
			t.Error("validateInitData() should return true for valid data")
		}
	})

	t.Run("rejects incorrect signature", func(t *testing.T) {
		initData := "user=%7B%22id%22%3A12345%7D&hash=invalidhash"
		if authService.validateInitData(initData) {
			t.Error("validateInitData() should return false for invalid hash")
		}
	})

	t.Run("rejects malformed query string", func(t *testing.T) {
		if authService.validateInitData("%%%invalid") {
			t.Error("validateInitData() should return false for malformed data")
		}
	})
}

func TestAuthService_getDataCheckString(t *testing.T) {
	authService := NewAuthService(testBotToken)

	t.Run("excludes hash from data check string", func(t *testing.T) {
		values := url.Values{
			"user":      []string{`{"id":123}`},
			"auth_date": []string{"1234567890"},
			"hash":      []string{"somehash"},
		}

		result := authService.getDataCheckString(values)
		if strings.Contains(result, "hash=") {
			t.Error("getDataCheckString() should exclude hash")
		}
	})

	t.Run("sorts keys alphabetically", func(t *testing.T) {
		values := url.Values{
			"zebra": []string{"z"},
			"alpha": []string{"a"},
			"beta":  []string{"b"},
		}

		result := authService.getDataCheckString(values)
		expected := "alpha=a\nbeta=b\nzebra=z"
		if result != expected {
			t.Errorf("getDataCheckString() = %v, want %v", result, expected)
		}
	})

	t.Run("joins with newlines", func(t *testing.T) {
		values := url.Values{
			"a": []string{"1"},
			"b": []string{"2"},
		}

		result := authService.getDataCheckString(values)
		if !strings.Contains(result, "\n") {
			t.Error("getDataCheckString() should join with newlines")
		}
	})
}

func TestAuthService_getUserIDFromInitData(t *testing.T) {
	authService := NewAuthService(testBotToken)

	t.Run("extracts user ID from valid data", func(t *testing.T) {
		userData := map[string]interface{}{"id": float64(12345), "username": "testuser"}
		userJSON, _ := json.Marshal(userData)
		initData := fmt.Sprintf("user=%s&auth_date=123", url.QueryEscape(string(userJSON)))

		userID, err := authService.getUserIDFromInitData(initData)
		if err != nil {
			t.Fatalf("getUserIDFromInitData() error = %v", err)
		}
		if userID != 12345 {
			t.Errorf("getUserIDFromInitData() = %v, want 12345", userID)
		}
	})

	t.Run("returns error for missing user field", func(t *testing.T) {
		initData := "auth_date=123"

		_, err := authService.getUserIDFromInitData(initData)
		if err == nil {
			t.Error("getUserIDFromInitData() should error for missing user")
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		initData := "user=invalid_json&auth_date=123"

		_, err := authService.getUserIDFromInitData(initData)
		if err == nil {
			t.Error("getUserIDFromInitData() should error for invalid JSON")
		}
	})

	t.Run("returns error for missing id in user", func(t *testing.T) {
		userData := map[string]interface{}{"username": "testuser"}
		userJSON, _ := json.Marshal(userData)
		initData := fmt.Sprintf("user=%s&auth_date=123", url.QueryEscape(string(userJSON)))

		_, err := authService.getUserIDFromInitData(initData)
		if err == nil {
			t.Error("getUserIDFromInitData() should error for missing id")
		}
	})
}

func TestGetUserIDFromContext(t *testing.T) {
	t.Run("returns user ID when present", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "userID", int64(12345))
		userID := GetUserIDFromContext(ctx)
		if userID != 12345 {
			t.Errorf("GetUserIDFromContext() = %v, want 12345", userID)
		}
	})

	t.Run("returns zero when not present", func(t *testing.T) {
		ctx := context.Background()
		userID := GetUserIDFromContext(ctx)
		if userID != 0 {
			t.Errorf("GetUserIDFromContext() = %v, want 0", userID)
		}
	})

	t.Run("returns zero for wrong type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "userID", "string_value")
		userID := GetUserIDFromContext(ctx)
		if userID != 0 {
			t.Errorf("GetUserIDFromContext() = %v, want 0 for wrong type", userID)
		}
	})
}

// Helper function to generate valid Telegram init data for testing
func generateValidInitData(t *testing.T, botToken string, userID int64, username string) string {
	t.Helper()

	userData := map[string]interface{}{
		"id":       float64(userID),
		"username": username,
	}
	userJSON, _ := json.Marshal(userData)

	values := url.Values{
		"user":      []string{string(userJSON)},
		"auth_date": []string{"1234567890"},
	}

	// Build data check string (sorted, excluding hash)
	dataToCheck := make([]string, 0, len(values))
	for k, v := range values {
		dataToCheck = append(dataToCheck, fmt.Sprintf("%s=%s", k, v[0]))
	}
	sort.Strings(dataToCheck)
	dataCheckString := strings.Join(dataToCheck, "\n")

	// Generate HMAC secret
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(botToken))
	hmacSecret := secret.Sum(nil)

	// Generate hash
	h := hmac.New(sha256.New, hmacSecret)
	h.Write([]byte(dataCheckString))
	hash := hex.EncodeToString(h.Sum(nil))

	values.Set("hash", hash)
	return values.Encode()
}
