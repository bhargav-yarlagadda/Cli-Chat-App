package utils

import (
    "encoding/base64"
    "fmt"
    "github.com/gorilla/websocket"
    "errors"
    "net"
    "net/http"
    "strings"
    "time"
)

// WSClient represents a WebSocket connection
type WSClient struct {
	Conn *websocket.Conn
}

// NewWSClient connects to the WebSocket server with JWT in headers
func NewWSClient(jwtToken, wsURL string) (*WSClient, error) {
	// Add Authorization header
	header := http.Header{}
	header.Add("Authorization", "Bearer "+jwtToken)

	// Dial WebSocket
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("failed to connect to websocket: %v, status: %s", err, resp.Status)
		}
		return nil, fmt.Errorf("failed to connect to websocket: %v", err)
	}

	return &WSClient{Conn: conn}, nil
}

// SendMessage sends an encrypted message to the server
func (c *WSClient) SendMessage(receiver string, encrypted []byte) error {
	msg := map[string]string{
		"receiver_username": receiver,
		"content":           base64.StdEncoding.EncodeToString(encrypted),
	}

	if err := c.Conn.WriteJSON(msg); err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}
	return nil
}

// ReceiveMessages listens for incoming messages and invokes the callback
func (c *WSClient) ReceiveMessages(handle func(sender, content string)) {
	for {
		var msg map[string]interface{}
		err := c.Conn.ReadJSON(&msg)
		if err != nil {
            // Suppress expected errors on normal shutdown
            if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) ||
                strings.Contains(err.Error(), "use of closed network connection") ||
                errors.Is(err, net.ErrClosed) {
                return
            }
            fmt.Println("Error reading message:", err)
			return
		}

		sender, _ := msg["sender_username"].(string)
		content, _ := msg["content"].(string)

		handle(sender, content)
	}
}

// Close closes the WebSocket connection
func (c *WSClient) Close() error {
    // Send close control frame for graceful shutdown
    _ = c.Conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
    return c.Conn.Close()
}
