package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/go-resty/resty/v2"
	"chat-client/utils"
)

var mu sync.Mutex

func RespondToConnectionRequest(args []string) {
	// Ensure user is logged in
	if token := os.Getenv("JWT_TOKEN"); token != "" {
		JWTToken = token
	} else {
		fmt.Println("You must login first using the login command.")
		return
	}

	client := resty.New()
	reader := bufio.NewReader(os.Stdin)

	// Fetch pending requests if empty
	if utils.Requests == nil || len(utils.Requests) == 0 {
		resp, err := client.R().
			SetHeader("Content-Type", "application/json").
			SetAuthToken(JWTToken).
			Get(utils.BaseURL + "/connections/pending")
		if err != nil {
			log.Fatal("Failed to fetch pending requests:", err)
		}
		if resp.StatusCode() != 200 {
			fmt.Println("Failed to fetch pending requests:", resp.String())
			return
		}

		var rawRequests []struct {
			ID            uint   `json:"id"`
			SenderID      uint   `json:"sender_id"`
			SenderUsername string `json:"sender_username"`
			Status        string `json:"status"`
		}
		if err := json.Unmarshal(resp.Body(), &rawRequests); err != nil {
			log.Fatal("Failed to parse pending requests:", err)
		}

		utils.Requests = make([]utils.PendingRequest, len(rawRequests))
		for i, r := range rawRequests {
			utils.Requests[i] = utils.PendingRequest{
				RequestID:      r.ID,
				SenderID:       r.SenderID,
				SenderUsername: r.SenderUsername,
				Status:         r.Status,
			}
		}
	}

	if len(utils.Requests) == 0 {
		fmt.Println("No pending connection requests.")
		return
	}

	// Get target username
	targetUsername := ""
	if len(args) > 0 && strings.HasPrefix(args[0], "--username:") {
		targetUsername = strings.TrimPrefix(args[0], "--username:")
	}

	if targetUsername == "" {
		fmt.Println("Pending Connection Requests:")
		for _, req := range utils.Requests {
			fmt.Printf("Username: %s\n", req.SenderUsername)
		}
		fmt.Print("Enter the username you want to respond to: ")
		u, _ := reader.ReadString('\n')
		targetUsername = strings.TrimSpace(u)
	}

	// Find request ID
	var requestID uint
	found := false
	for _, req := range utils.Requests {
		if req.SenderUsername == targetUsername {
			requestID = req.RequestID
			found = true
			break
		}
	}

	if !found {
		fmt.Println("No pending request found from that username.")
		return
	}

	// Ask action
	fmt.Print("Enter action (accept/reject): ")
	action, _ := reader.ReadString('\n')
	action = strings.TrimSpace(strings.ToLower(action))
	if action != "accept" && action != "reject" {
		fmt.Println("Invalid action. Must be 'accept' or 'reject'.")
		return
	}

	// Send response
	body := map[string]interface{}{
		"request_id": requestID,
		"action":     action,
	}

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(JWTToken).
		SetBody(body).
		Post(utils.BaseURL + "/connections/respond")
	if err != nil {
		log.Fatal("Failed to respond to connection request:", err)
	}
	if resp.StatusCode() != 200 {
		fmt.Println("Error:", resp.String())
		return
	}

	fmt.Printf("Successfully %s the connection request from %s.\n", action, targetUsername)

	// Remove from cache
	go func() {
		mu.Lock()
		defer mu.Unlock()
		newRequests := make([]utils.PendingRequest, 0, len(utils.Requests))
		for _, req := range utils.Requests {
			if req.RequestID != requestID {
				newRequests = append(newRequests, req)
			}
		}
		utils.Requests = newRequests
	}()
}
