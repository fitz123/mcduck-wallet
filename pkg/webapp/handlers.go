// File: webapp/handlers.go

package webapp

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/fitz123/mcduck-wallet/pkg/database"
	"github.com/fitz123/mcduck-wallet/pkg/handlers"
)

// SetupRoutes sets up the HTTP routes for the WebApp
func SetupRoutes() {
	http.HandleFunc("/balance", AuthMiddleware(getBalance))
	http.HandleFunc("/transfer", AuthMiddleware(transferMoney))
	http.HandleFunc("/history", AuthMiddleware(getTransactionHistory))
	http.HandleFunc("/", serveHTML)
}

func serveHTML(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "webapp/index.html")
}

func getBalance(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r)
	user, err := handlers.GetOrCreateUser(userID, "")
	if err != nil {
		http.Error(w, "Error fetching balance", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "Your current balance is $%.2f", user.Balance)
}

func transferMoney(w http.ResponseWriter, r *http.Request) {
	log.Println("Transfer money handler called")

	userID := GetUserIDFromContext(r)
	log.Printf("User ID from context: %d", userID)

	if userID == 0 {
		log.Println("User ID is 0, possibly not authenticated")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	toUsername := r.FormValue("to_username")
	amount, err := strconv.ParseFloat(r.FormValue("amount"), 64)
	if err != nil {
		log.Printf("Invalid amount: %v", err)
		http.Error(w, "Invalid amount", http.StatusBadRequest)
		return
	}

	log.Printf("Transfer request: From user %d to %s, amount: %.2f", userID, toUsername, amount)

	var toUser database.User
	if err := database.DB.Where("username = ?", toUsername).First(&toUser).Error; err != nil {
		log.Printf("Recipient not found: %v", err)
		http.Error(w, "Recipient not found", http.StatusNotFound)
		return
	}

	log.Printf("Recipient found: ID %d", toUser.TelegramID)

	err = handlers.TransferMoney(userID, toUser.TelegramID, amount)
	if err != nil {
		log.Printf("Transfer failed: %v", err)
		http.Error(w, fmt.Sprintf("Transfer failed: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	log.Printf("Transfer successful: $%.2f from %d to %d", amount, userID, toUser.TelegramID)
	fmt.Fprintf(w, "Successfully transferred $%.2f to %s", amount, toUsername)
}

func getTransactionHistory(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r)
	transactions, err := handlers.GetTransactionHistory(userID)
	if err != nil {
		http.Error(w, "Error fetching transaction history", http.StatusInternalServerError)
		return
	}

	var historyHTML string
	for _, t := range transactions {
		var description string
		if t.Type == "transfer" {
			if t.Amount < 0 {
				description = fmt.Sprintf("Sent $%.2f", -t.Amount)
			} else {
				description = fmt.Sprintf("Received $%.2f", t.Amount)
			}
		} else {
			description = fmt.Sprintf("Deposited $%.2f", t.Amount)
		}
		historyHTML += fmt.Sprintf("<p>%s - %s</p>", t.Timestamp.Format("2006-01-02 15:04:05"), description)
	}

	fmt.Fprint(w, historyHTML)
}
