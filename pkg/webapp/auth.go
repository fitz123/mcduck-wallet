// File: pkg/webapp/auth.go

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

	"github.com/fitz123/mcduck-wallet/pkg/logger"
)

var BotToken string

// InitAuth initializes the authentication module with the bot token
func InitAuth(botToken string) {
	BotToken = botToken
	logger.Info("Authentication module initialized")
}

// AuthMiddleware is a middleware that validates the Telegram WebApp init data
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		initData := r.Header.Get("X-Telegram-Init-Data")

		if initData == "" {
			logger.Warn("No initData received")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if !validateInitData(initData) {
			logger.Warn("Invalid initData")
			http.Error(w, "Invalid init data", http.StatusUnauthorized)
			return
		}

		userID, err := getUserIDFromInitData(initData)
		if err != nil {
			logger.Error("Error getting user ID", "error", err)
			http.Error(w, "Invalid user data", http.StatusUnauthorized)
			return
		}

		logger.Info("Authenticated user", "userID", userID)

		// Set user ID in context
		ctx := r.Context()
		ctx = context.WithValue(ctx, "userID", userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func validateInitData(initData string) bool {
	values, err := url.ParseQuery(initData)
	if err != nil {
		return false
	}

	dataCheckString := getDataCheckString(values)
	secret := getHMACSecret()
	hash := getHash(dataCheckString, secret)

	return hash == values.Get("hash")
}

func getDataCheckString(values url.Values) string {
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

func getHMACSecret() []byte {
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(BotToken))
	return secret.Sum(nil)
}

func getHash(data string, secret []byte) string {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func getUserIDFromInitData(initData string) (int64, error) {
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

// GetUserIDFromContext retrieves the user ID from the request context
func GetUserIDFromContext(r *http.Request) int64 {
	userID, _ := r.Context().Value("userID").(int64)
	logger.Debug("Retrieved user ID from context", "userID", userID)
	return userID
}
