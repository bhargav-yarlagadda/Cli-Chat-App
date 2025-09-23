package commands

import (
	"bufio"
	"chat-client/utils"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// Chat starts a chat session with a given user
func Chat(args []string) {
	// --- 1. Get JWT token ---
	jwtToken := os.Getenv("JWT_TOKEN")
	if jwtToken == "" {
		fmt.Println("Please login first to obtain JWT token.")
		return
	}

	// --- 2. Get target username ---
	var username string
	helpRegex := regexp.MustCompile(`^--help$|^-h$`)
	for _, arg := range args {
		if helpRegex.MatchString(arg) {
			fmt.Println("Usage: chat [--username:<username>]")
			return
		}
	}

	if len(args) >= 1 && strings.HasPrefix(args[0], "--username:") {
		username = strings.TrimPrefix(args[0], "--username:")
	} else {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter username: ")
		u, _ := reader.ReadString('\n')
		username = strings.TrimSpace(u)
	}

	// --- 3. Load current user's private key ---
	currentUser := os.Getenv("CURRENT_USER")
	if currentUser == "" {
		fmt.Println("Please login first to obtain your username.")
		return
	}

	keyFileName := fmt.Sprintf("keys/%s_private.pem", currentUser)
	privKeyData, err := os.ReadFile(keyFileName)
	if err != nil {
		fmt.Printf("Could not find private key file at %s\n", keyFileName)
		return
	}

	if !strings.Contains(string(privKeyData), "-----BEGIN RSA PRIVATE KEY-----") {
		fmt.Printf("Invalid private key format in %s\n", keyFileName)
		return
	}
	privKey, err := parsePrivateKey(string(privKeyData))
	if err != nil {
		fmt.Printf("Error parsing private key: %v\n", err)
		return
	}

	// --- 4. Get receiver's public key from server ---
	userInfo, exists := utils.GetUser(username, jwtToken)
	if !exists {
		fmt.Println("User not found:", username)
		return
	}
	receiverPubKey, err := parsePublicKey(userInfo.PublicKey)
	if err != nil {
		fmt.Printf("Error parsing %s's public key: %v\n", username, err)
		return
	}

	// --- 5. Connect to WebSocket server ---
	wsURL := "ws://localhost:8080/chat"
	client, err := utils.NewWSClient(jwtToken, wsURL)
	if err != nil {
		fmt.Println("Failed to connect to chat server:", err)
		return
	}
	defer client.Close()

	fmt.Printf("\nStarting chat with %s...\n", username)
	fmt.Println("Type your message and press Enter to send. Type 'exit' to quit.")
	fmt.Println("----------------------------------------")

	// --- 6. Receive messages from server ---
	go client.ReceiveMessages(func(sender, content string) {
		if sender != username {
			return
		}
		encryptedBytes, err := base64.StdEncoding.DecodeString(content)
		if err != nil {
			fmt.Printf("\nError decoding message: %v\n", err)
			return
		}

		decrypted := decryptMessage(privKey, encryptedBytes)
		if decrypted == nil {
			fmt.Printf("\nFailed to decrypt message from %s\n", sender)
			return
		}

		// Pretty print with timestamp and indentation for multiline messages
		// Move to line start to avoid leaving the prompt mid-line
		fmt.Print("\r")
		text := string(decrypted)
		lines := strings.Split(text, "\n")
		ts := time.Now().Format("15:04")
		if len(lines) > 0 {
			fmt.Printf("\n[%s] %s: %s\n", ts, sender, strings.TrimRight(lines[0], "\r"))
			for i := 1; i < len(lines); i++ {
				fmt.Printf("%s\n", strings.TrimRight(lines[i], "\r"))
			}
		} else {
			fmt.Printf("\n[%s] %s:\n", ts, sender)
		}
		// Restore prompt
		fmt.Print("You: ")
	})

	// --- 7. Handle user input ---
	reader := bufio.NewReader(os.Stdin)
	// Initial prompt
	fmt.Print("You: ")
	for {
		msg, _ := reader.ReadString('\n')
		msg = strings.TrimSpace(msg)

		if msg == "exit" {
			fmt.Println("Exiting chat...")
			return
		}

		// Encrypt message
		encrypted := encryptMessage(receiverPubKey, []byte(msg))
		if encrypted == nil {
			fmt.Println("Failed to encrypt message. Please try again.")
			continue
		}

		// Send over WebSocket
		if err := client.SendMessage(username, encrypted); err != nil {
			fmt.Printf("Failed to send message: %v\n", err)
			continue
		}
		fmt.Println("✓ Message sent successfully")
		// Redisplay prompt
		fmt.Print("You: ")
	}
}

// ------------------- RSA helpers -------------------

func parsePrivateKey(privPEM string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privPEM))
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, fmt.Errorf("invalid private key PEM")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}
	return key, nil
}

func parsePublicKey(pubPEM string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pubPEM))
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("invalid public key PEM")
	}
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}
	pubKey, ok := pubInterface.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("expected RSA public key, got different type")
	}
	return pubKey, nil
}

const maxMessageLength = 245 // PKCS#1 v1.5 padding

func encryptMessage(pubKey *rsa.PublicKey, msg []byte) []byte {
	if len(msg) <= maxMessageLength {
		ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, pubKey, msg)
		if err != nil {
			fmt.Printf("Encryption error: %v\n", err)
			return nil
		}
		result := make([]byte, 2+len(ciphertext))
		result[0] = byte(len(ciphertext) >> 8)
		result[1] = byte(len(ciphertext))
		copy(result[2:], ciphertext)
		return result
	}

	var encrypted []byte
	for i := 0; i < len(msg); i += maxMessageLength {
		end := i + maxMessageLength
		if end > len(msg) {
			end = len(msg)
		}
		chunk := msg[i:end]
		ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, pubKey, chunk)
		if err != nil {
			fmt.Printf("Chunk encryption error: %v\n", err)
			return nil
		}
		chunkHeader := []byte{byte(len(ciphertext) >> 8), byte(len(ciphertext))}
		encrypted = append(encrypted, chunkHeader...)
		encrypted = append(encrypted, ciphertext...)
	}
	return encrypted
}

func decryptMessage(privKey *rsa.PrivateKey, ciphertext []byte) []byte {
	if len(ciphertext) < 2 {
		fmt.Println("Invalid ciphertext: too short")
		return nil
	}

	var decrypted []byte
	i := 0
	for i < len(ciphertext) {
		if i+2 > len(ciphertext) {
			fmt.Println("Invalid ciphertext: truncated chunk header")
			return nil
		}
		size := int(ciphertext[i])<<8 | int(ciphertext[i+1])
		i += 2
		if size <= 0 || size > 256 || i+size > len(ciphertext) {
			fmt.Println("Invalid chunk size in ciphertext")
			return nil
		}
		chunk := ciphertext[i : i+size]
		plain, err := rsa.DecryptPKCS1v15(rand.Reader, privKey, chunk)
		if err != nil {
			fmt.Printf("Chunk decryption error: %v\n", err)
			return nil
		}
		decrypted = append(decrypted, plain...)
		i += size
	}
	return decrypted
}
