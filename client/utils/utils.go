package utils

type PendingRequest struct {
	RequestID      uint   `json:"id"`             // id from server is request_id
	SenderID       uint   `json:"sender_id"`
	SenderUsername string `json:"sender_username"`
	Status         string `json:"status"`         // optional
}
var Requests []PendingRequest


const BaseURL = "http://localhost:8080"
