// File: ./internal/webapp/auth.go
package webapp

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/fitz123/mcduck-wallet/internal/logger"
)

type AuthService struct {
	BotToken string
}

func NewAuthService(botToken string) *AuthService {
	return &AuthService{BotToken: botToken}
}

func (as *AuthService) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Debug("AuthMiddleware: Starting authentication process")

		initData := r.Header.Get("X-Telegram-Init-Data")
		logger.Debug("AuthMiddleware: Received initData", "initData", initData)

		if initData == "" {
			logger.Warn("AuthMiddleware: No initData received")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if !as.validateInitData(initData) {
			logger.Warn("AuthMiddleware: Invalid initData")
			http.Error(w, "Invalid init data", http.StatusUnauthorized)
			return
		}

		userID, err := as.getUserIDFromInitData(initData)
		if err != nil {
			logger.Error("AuthMiddleware: Error getting user ID", "error", err)
			http.Error(w, "Invalid user data", http.StatusUnauthorized)
			return
		}

		logger.Info("AuthMiddleware: Authenticated user", "userID", userID)

		// Set user ID in context
		ctx := context.WithValue(r.Context(), "userID", userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (as *AuthService) validateInitData(initData string) bool {
	values, err := url.ParseQuery(initData)
	if err != nil {
		return false
	}

	dataCheckString := as.getDataCheckString(values)
	secret := as.getHMACSecret()
	hash := as.getHash(dataCheckString, secret)

	return hash == values.Get("hash")
}

func (as *AuthService) getDataCheckString(values url.Values) string {
	dataToCheck := make([]string, 0, len(values))
	for k, v := range values {
		if k == "hash" {
			continue
		}
		dataToCheck = append(dataToCheck, fmt.Sprintf("%s=%s", k, v[0]))
	}
	sort.Strings(dataToCheck)
	return strings.Join(dataToCheck, "\n")
}

func (as *AuthService) getHMACSecret() []byte {
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(as.BotToken))
	return secret.Sum(nil)
}

func (as *AuthService) getHash(data string, secret []byte) string {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func (as *AuthService) getUserIDFromInitData(initData string) (int64, error) {
	values, _ := url.ParseQuery(initData)
	userDataStr := values.Get("user")
	var userData map[string]interface{}
	err := json.Unmarshal([]byte(userDataStr), &userData)
	if err != nil {
		return 0, err
	}
	userID, ok := userData["id"].(float64)
	if !ok {
		return 0, fmt.Errorf("user ID not found")
	}
	return int64(userID), nil
}

func GetUserIDFromContext(ctx context.Context) int64 {
	userID, _ := ctx.Value("userID").(int64)
	return userID
}
