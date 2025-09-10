package commands

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/go-resty/resty/v2"
	"chat-client/utils"
)

func ViewPendingRequests() {
	// Ensure user is logged in
	if token := os.Getenv("JWT_TOKEN"); token != "" {
		JWTToken = token
	} else {
		fmt.Println("You must login first using the login command.")
		return
	}

	client := resty.New()

	// Fetch pending requests
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(JWTToken).
		Get(utils.BaseURL + "/connections/pending")
	if err != nil {
		log.Fatal("Request failed:", err)
	}

	if resp.StatusCode() != 200 {
		fmt.Println("Failed to fetch pending requests:", resp.String())
		return
	}

	// Use local struct to ensure proper unmarshaling
	var rawRequests []struct {
		ID            uint   `json:"id"`
		SenderID      uint   `json:"sender_id"`
		SenderUsername string `json:"sender_username"`
		Status        string `json:"status"`
	}

	if err := json.Unmarshal(resp.Body(), &rawRequests); err != nil {
		log.Fatal("Failed to parse response:", err)
	}

	// Convert to global Requests
	utils.Requests = make([]utils.PendingRequest, len(rawRequests))
	for i, r := range rawRequests {
		utils.Requests[i] = utils.PendingRequest{
			RequestID:      r.ID,
			SenderID:       r.SenderID,
			SenderUsername: r.SenderUsername,
			Status:         r.Status,
		}
	}

	if len(utils.Requests) == 0 {
		fmt.Println("No pending connection requests.")
		return
	}

	fmt.Println("Pending Connection Requests:")
	for _, req := range utils.Requests {
		fmt.Printf("Request ID: %d | From Username: %s\n", req.RequestID, req.SenderUsername)
	}
}
