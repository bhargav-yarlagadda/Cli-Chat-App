package handlers

import (
	"chat-server/db"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// GetAllConnections returns all connections for logged-in user
func getAllConnections(c *fiber.Ctx) error {
	// Get claims from middleware
	claims := c.Locals("user")
	if claims == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "User not authorized. Please login.",
		})
	}

	userClaims := claims.(jwt.MapClaims)
	userID := uint(userClaims["user_id"].(float64)) // convert from float64

	// Fetch connections
	var connections []db.Connection
	if err := db.DB_Conn.
		Where("(sender_id = ? OR receiver_id = ?) AND status = ?", userID, userID, "accepted").
		Find(&connections).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(connections)
}

// getPendingCount returns the count of pending requests for logged-in user
func getPendingCount(c *fiber.Ctx) error {
	claims := c.Locals("user")
	if claims == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "User not authorized. Please login.",
		})
	}

	userClaims := claims.(jwt.MapClaims)
	userID := uint(userClaims["user_id"].(float64))

	var count int64
	if err := db.DB_Conn.
		Model(&db.Connection{}).
		Where("receiver_id = ? AND status = ?", userID, "pending").
		Count(&count).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"pending_count": count})
}

func sendConnectionRequest(c *fiber.Ctx) error {
	claims := c.Locals("user")
	if claims == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Unauthorized",
		})
	}
	userClaims := claims.(jwt.MapClaims)
	senderID := uint(userClaims["user_id"].(float64))

	body := struct {
		Username string `json:"username"`
	}{}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	var receiver db.User
	if err := db.DB_Conn.Where("username = ?", body.Username).First(&receiver).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}
	if receiver.ID == senderID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot send request to yourself"})
	}
	// Check if connection already exists
	var existing db.Connection
	err := db.DB_Conn.Where("(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)",
		senderID, receiver.ID, receiver.ID, senderID).First(&existing).Error
	if err == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Connection already exists"})
	}

	conn := db.Connection{
		SenderID:   senderID,
		ReceiverID: receiver.ID,
		Status:     "pending",
	}

	if err := db.DB_Conn.Create(&conn).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Connection request sent"})

}
func respondConnection(c *fiber.Ctx) error {
	claims := c.Locals("user")
	if claims == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Unauthorized",
		})
	}

	userClaims := claims.(jwt.MapClaims)
	userID := uint(userClaims["user_id"].(float64))

	// Parse body
	body := struct {
		RequestID uint   `json:"request_id"`
		Action    string `json:"action"` // "accept" or "reject"
	}{}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if body.Action != "accept" && body.Action != "reject" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Action must be 'accept' or 'reject'"})
	}

	// Find connection request
	var conn db.Connection
	if err := db.DB_Conn.
		Where("id = ? AND receiver_id = ?", body.RequestID, userID).
		First(&conn).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Connection request not found"})
	}

	// Only the receiver can respond
	if conn.ReceiverID != userID {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "You are not allowed to respond to this request"})
	}

	if body.Action == "accept" {
		// Accept: update status
		conn.Status = "accepted"
		if err := db.DB_Conn.Save(&conn).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
	} else {
		// Reject: delete the request
		if err := db.DB_Conn.Delete(&conn).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
	}

	return c.JSON(fiber.Map{"message": "Connection request " + body.Action})
}

// getPendingRequests returns all pending connection requests for logged-in user (as receiver)
// getPendingRequests returns all pending connection requests for logged-in user (as receiver)
func getPendingRequests(c *fiber.Ctx) error {
	claims := c.Locals("user")
	if claims == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "User not authorized. Please login.",
		})
	}

	userClaims := claims.(jwt.MapClaims)
	userID := uint(userClaims["user_id"].(float64))

	var pending []db.Connection
	if err := db.DB_Conn.
		Preload("Sender"). // preload the Sender relation
		Where("receiver_id = ? AND status = ?", userID, "pending").
		Find(&pending).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Build a custom response with only SenderID and SenderUsername
	type PendingResponse struct {
		RequestID      uint   `json:"request_id"`
		SenderID       uint   `json:"sender_id"`
		SenderUsername string `json:"sender_username"`
	}

	resp := make([]PendingResponse, len(pending))
	for i, p := range pending {
		resp[i] = PendingResponse{
			RequestID:      p.ID,
			SenderID:       p.SenderID,
			SenderUsername: p.Sender.Username,
		}
	}

	return c.JSON(resp)
}

func HandleConnections(app fiber.Router) {
	app.Get("/", getAllConnections)             // accepted connections
	app.Get("/pending", getPendingRequests)     // pending requests for receiver
	app.Post("/connect", sendConnectionRequest) // send request
	app.Post("/respond", respondConnection)     // accept/reject
	app.Get("/pending/count", getPendingCount)
}
