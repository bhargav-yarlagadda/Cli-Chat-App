package utils

import (
	"encoding/json"
	"fmt"

	"github.com/go-resty/resty/v2"
)

type UserInfo struct {
	ID        uint   `json:"id"`
	Username  string `json:"username"`
	PublicKey string `json:"public_key"`
	CreatedAt string `json:"created_at"`
}

type PendingRequest struct {
	RequestID      uint   `json:"request_id"` // match server response format
	SenderID       uint   `json:"sender_id"`
	SenderUsername string `json:"sender_username"`
	Status         string `json:"status"` // optional
}

var BaseURL = "http://localhost:8080" // replace with your server URL

var Requests []PendingRequest

// GetUser calls /auth/user-info?username=<username>
// Returns UserInfo struct and true if user exists, else nil and false
func GetUser(username string, jwtToken string) (*UserInfo, bool) {
	client := resty.New()
	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+jwtToken).
		SetQueryParam("username", username).
		Get(BaseURL + "/auth/user-info")

	if err != nil {
		fmt.Println("Request error:", err)
		return nil, false
	}

	if resp.IsSuccess() {
		var result struct {
			User UserInfo `json:"user"`
		}

		if err := json.Unmarshal(resp.Body(), &result); err != nil {
			fmt.Println("Failed to parse response:", err)
			return nil, false
		}

		return &result.User, true
	}

	fmt.Println("User not found or error:", resp.String())
	return nil, false
}
