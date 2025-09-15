// package handlers

// import (
// 	"chat-server/db"
// 	"encoding/json"
// 	"log"
// 	"sync"

// 	"github.com/gofiber/fiber/v2"
// 	"github.com/gofiber/websocket/v2"
// 	"github.com/golang-jwt/jwt/v5"
// )

// // Clients: thread-safe map of userID -> []*websocket.Conn
// var Clients sync.Map // key: uint (userID), value: []*websocket.Conn

// // IncomingMessage represents a message sent by client
// type IncomingMessage struct {
// 	ReceiverID uint   `json:"receiver_id"` // The other participant
// 	Content    string `json:"content"`     // already encrypted
// }

// // HandleWebSocketServer sets up WebSocket routes
// func HandleWebSocketServer(router fiber.Router) {
// 	// Middleware to upgrade HTTP to WebSocket
// 	router.Use(func(c *fiber.Ctx) error {
// 		if websocket.IsWebSocketUpgrade(c) {
// 			return c.Next()
// 		}
// 		return fiber.ErrUpgradeRequired
// 	})

// 	router.Get("/", websocket.New(func(conn *websocket.Conn) {
// 		claims := conn.Locals("user").(jwt.MapClaims)
// 		userID := uint(claims["user_id"].(float64))

// 		// Add connection to user's slice
// 		//  we are storeing the current users connection in a slice
// 		// this is because there could be a case
// 		// where same user logged in thought different devices
// 		// if we dont store them in a slice
// 		// when one conn is closed alll others are closed tooo
// 		// also there could mistakes in message handling too
// 		conns, _ := Clients.LoadOrStore(userID, []*websocket.Conn{})
// 		connSlice := conns.([]*websocket.Conn)
// 		connSlice = append(connSlice, conn)
// 		Clients.Store(userID, connSlice)

// 		defer func() {
// 			// Remove this connection from slice
// 			conns, ok := Clients.Load(userID)
// 			if ok {
// 				connSlice := conns.([]*websocket.Conn)
// 				newSlice := []*websocket.Conn{}
// 				for _, c := range connSlice {
// 					if c != conn {
// 						newSlice = append(newSlice, c)
// 					}
// 				}
// 				if len(newSlice) == 0 {
// 					Clients.Delete(userID)
// 				} else {
// 					Clients.Store(userID, newSlice)
// 				}
// 			}
// 			conn.Close()
// 			log.Printf("User %d disconnected\n", userID)
// 		}()

// 		log.Printf("User %d connected via WebSocket\n", userID)

// 		for {
// 			_, msg, err := conn.ReadMessage()
// 			if err != nil {
// 				log.Println("read error:", err)
// 				break
// 			}

// 			go handleIncomingMessage(userID, msg)
// 		}
// 	}))
// }

// // handleIncomingMessage processes each message
// func handleIncomingMessage(senderID uint, raw []byte) {
// 	var incoming IncomingMessage
// 	if err := json.Unmarshal(raw, &incoming); err != nil {
// 		log.Println("invalid message format:", err)
// 		return
// 	}

// 	// Save message in DB (Delivered=false initially)
// 	message := db.Message{
// 		SenderID:   senderID,
// 		ReceiverID: incoming.ReceiverID,
// 		Content:    incoming.Content,
// 		Delivered:  false,
// 	}
// 	db.DB_Conn.Create(&message)

// 	// Fetch undelivered messages from receiver → sender (current user)
// 	// receiver id is the opposite party 
// 	// also check weather the current uses has any pending undelived messages from the opposite partyyyy
// 	var undelivered []db.Message
// 	db.DB_Conn.Where("sender_id = ? AND receiver_id = ? AND delivered = ?", incoming.ReceiverID, senderID, false).
// 		Order("created_at asc").Find(&undelivered)

// 	// Send all undelivered messages to current user (senderID)
// 	if conns, ok := Clients.Load(senderID); ok {
// 		for _, c := range conns.([]*websocket.Conn) {
// 			for _, msg := range undelivered {
// 				out, _ := json.Marshal(msg)
// 				c.WriteMessage(websocket.TextMessage, out)
// 				db.DB_Conn.Model(&msg).Update("delivered", true)
// 			}
// 		}
// 	}

// 	// If receiver online → deliver immediately
// 	if conns, ok := Clients.Load(incoming.ReceiverID); ok {
// 		for _, c := range conns.([]*websocket.Conn) {
// 			out, _ := json.Marshal(message)
// 			if err := c.WriteMessage(websocket.TextMessage, out); err != nil {
// 				log.Println("send error:", err)
// 				continue
// 			}
// 			db.DB_Conn.Model(&message).Update("delivered", true)
// 		}
// 	}
// }
package handlers

import (
	"chat-server/db"
	"encoding/json"
	"log"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// Clients stores userID -> []*websocket.Conn
var Clients sync.Map

// IncomingMessage represents a message sent by a client
type IncomingMessage struct {
	ReceiverUsername string `json:"receiver_username"` // Receiver username
	Content          string `json:"content"`           // Encrypted message
}

// HandleWebSocketServer sets up the WebSocket endpoint
func HandleWebSocketServer(router fiber.Router) {
	router.Use(func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	router.Get("/", websocket.New(func(conn *websocket.Conn) {
		// --- 1. Identify user from Locals (JWT claims must be stored here) ---
		claims := conn.Locals("user").(map[string]interface{})
		senderID := uint(claims["user_id"].(float64))

		// --- 2. Add connection to Clients map ---
		conns, _ := Clients.LoadOrStore(senderID, []*websocket.Conn{})
		connSlice := conns.([]*websocket.Conn)
		connSlice = append(connSlice, conn)
		Clients.Store(senderID, connSlice)

		defer func() {
			// Remove connection on disconnect
			conns, ok := Clients.Load(senderID)
			if ok {
				connSlice := conns.([]*websocket.Conn)
				newSlice := []*websocket.Conn{}
				for _, c := range connSlice {
					if c != conn {
						newSlice = append(newSlice, c)
					}
				}
				if len(newSlice) == 0 {
					Clients.Delete(senderID)
				} else {
					Clients.Store(senderID, newSlice)
				}
			}
			conn.Close()
			log.Printf("User %d disconnected\n", senderID)
		}()

		log.Printf("User %d connected via WebSocket\n", senderID)

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Println("read error:", err)
				break
			}

			go handleIncomingMessage(senderID, msg)
		}
	}))
}

// handleIncomingMessage validates connection and delivers messages
func handleIncomingMessage(senderID uint, raw []byte) {
	var incoming IncomingMessage
	if err := json.Unmarshal(raw, &incoming); err != nil {
		log.Println("invalid message format:", err)
		return
	}

	// --- 1. Fetch receiver from DB ---
	var receiver db.User
	if err := db.DB_Conn.Where("username = ?", incoming.ReceiverUsername).First(&receiver).Error; err != nil {
		log.Println("Receiver not found:", incoming.ReceiverUsername)
		return
	}

	// --- 2. Validate connection ---
	var conn db.Connection
	if err := db.DB_Conn.Where(
		"(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)",
		senderID, receiver.ID, receiver.ID, senderID,
	).First(&conn).Error; err != nil {
		log.Printf("No connection between %d and %d\n", senderID, receiver.ID)
		return
	}

	if conn.Status != "accepted" {
		log.Printf("Connection not accepted between %d and %d\n", senderID, receiver.ID)
		return
	}

	// --- 3. Save message in DB ---
	message := db.Message{
		SenderID:   senderID,
		ReceiverID: receiver.ID,
		Content:    incoming.Content,
		Delivered:  false,
	}
	db.DB_Conn.Create(&message)

	// --- 4. Deliver undelivered messages to sender (if any) ---
	var undelivered []db.Message
	db.DB_Conn.Where("sender_id = ? AND receiver_id = ? AND delivered = ?", receiver.ID, senderID, false).
		Order("created_at asc").Find(&undelivered)

	if conns, ok := Clients.Load(senderID); ok {
		for _, c := range conns.([]*websocket.Conn) {
			for _, msg := range undelivered {
				out, _ := json.Marshal(msg)
				c.WriteMessage(websocket.TextMessage, out)
				db.DB_Conn.Model(&msg).Update("delivered", true)
			}
		}
	}

	// --- 5. Deliver message to receiver if online ---
	if conns, ok := Clients.Load(receiver.ID); ok {
		for _, c := range conns.([]*websocket.Conn) {
			out, _ := json.Marshal(message)
			if err := c.WriteMessage(websocket.TextMessage, out); err != nil {
				log.Println("send error:", err)
				continue
			}
			db.DB_Conn.Model(&message).Update("delivered", true)
		}
	}
}
