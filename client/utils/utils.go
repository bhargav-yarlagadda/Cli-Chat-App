package utils

type PendingRequest struct {
	RequestID      uint   `json:"request_id"` // match server response format
	SenderID       uint   `json:"sender_id"`
	SenderUsername string `json:"sender_username"`
	Status         string `json:"status"` // optional
}

var Requests []PendingRequest

const BaseURL = "http://localhost:8080"
