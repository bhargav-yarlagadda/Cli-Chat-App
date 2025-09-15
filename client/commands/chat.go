package commands

import (
	"bufio"
	"chat-client/utils"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Chat starts a chat session
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

	// --- 3. Get current user's private key from file ---
	currentUser := os.Getenv("CURRENT_USER")
	if currentUser == "" {
		fmt.Println("Please login first to obtain your username.")
		return
	}

	// Load private key from file
	keyFileName := fmt.Sprintf("keys/%s_private.pem", currentUser)
	privKeyData, err := os.ReadFile(keyFileName)
	if err != nil {
		fmt.Printf("Could not find private key file at %s\n", keyFileName)
		fmt.Println("If you haven't registered yet, please use the register command first.")
		fmt.Println("If you have registered, make sure your private key file is in the correct location.")
		fmt.Printf("Current user: %s\n", currentUser)
		return
	}

	// Validate PEM format
	if !strings.Contains(string(privKeyData), "-----BEGIN RSA PRIVATE KEY-----") {
		fmt.Printf("Invalid private key format in %s\n", keyFileName)
		fmt.Println("The key file should be in PEM format starting with -----BEGIN RSA PRIVATE KEY-----")
		return
	}
	privKeyStr := string(privKeyData)

	// --- 4. Get receiver's public key from server ---
	userInfo, exists := utils.GetUser(username, jwtToken)
	if !exists {
		fmt.Println("User not found:", username)
		return
	}
	receiverPubKeyStr := userInfo.PublicKey

	// --- 5. Parse RSA keys ---
	fmt.Printf("Loading private key for user %s...\n", currentUser)
	privKey, err := parsePrivateKey(privKeyStr)
	if err != nil {
		fmt.Printf("Error parsing your private key from %s: %v\n", keyFileName, err)
		fmt.Println("Please ensure your private key file is in the correct PEM format.")
		return
	}

	fmt.Printf("Loading public key for user %s...\n", username)
	receiverPubKey, err := parsePublicKey(receiverPubKeyStr)
	if err != nil {
		fmt.Printf("Error parsing %s's public key: %v\n", username, err)
		fmt.Println("The user's public key may be invalid or corrupted.")
		return
	}

	// Verify that we can encrypt and decrypt with these keys
	fmt.Println("Verifying encryption keys...")
	testMessage := []byte("test")
	encrypted := encryptMessage(receiverPubKey, testMessage)
	if encrypted == nil {
		fmt.Printf("Failed to encrypt test message. Key verification failed.\n")
		fmt.Printf("Current user: %s, Target user: %s\n", currentUser, username)
		return
	}

	decrypted := decryptMessage(privKey, encrypted)
	if decrypted == nil {
		fmt.Println("Failed to decrypt test message. Key verification failed.")
		fmt.Printf("Please ensure you're using the correct private key for user %s\n", currentUser)
		return
	}

	if string(decrypted) != string(testMessage) {
		fmt.Println("Key verification failed. Message encryption may not work.")
		fmt.Printf("Expected test message: %q, Got: %q\n", testMessage, decrypted)
		return
	}

	fmt.Println("✓ Encryption keys verified successfully")

	// --- Ready to encrypt/decrypt messages ---
	fmt.Printf("\nStarting chat with %s...\n", username)
	fmt.Println("Type your message and press Enter to send. Type 'exit' to quit.")
	fmt.Println("----------------------------------------")

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\nYou: ")
		msg, _ := reader.ReadString('\n')
		msg = strings.TrimSpace(msg)

		if msg == "exit" {
			fmt.Println("Exiting chat...")
			return
		}

		// Encrypt message for receiver
		encrypted := encryptMessage(receiverPubKey, []byte(msg))
		if encrypted == nil {
			fmt.Println("Failed to encrypt message. Please try again.")
			continue
		}

		// Decrypt message (for demonstration, using receiver's public key)
		decrypted := decryptMessage(privKey, encrypted)
		if decrypted == nil {
			fmt.Println("Failed to decrypt message for verification. Message might not be delivered correctly.")
			continue
		}

		if string(decrypted) == msg {
			fmt.Println("✓ Message encrypted successfully")
		} else {
			fmt.Println("⚠ Message encryption verification failed")
		}
	}
}

// ------------------- RSA helpers -------------------

func parsePrivateKey(privPEM string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privPEM))
	if block == nil {
		return nil, fmt.Errorf("invalid private key PEM")
	}
	if block.Type != "RSA PRIVATE KEY" {
		return nil, fmt.Errorf("expected RSA PRIVATE KEY, got %s", block.Type)
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}
	return key, nil
}

func parsePublicKey(pubPEM string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pubPEM))
	if block == nil {
		return nil, fmt.Errorf("invalid public key PEM")
	}
	if block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("expected PUBLIC KEY, got %s", block.Type)
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

// For RSA-2048, the maximum size of data that can be encrypted is the key size (2048 bits = 256 bytes)
// minus padding (11 bytes for PKCS#1 v1.5)
const maxMessageLength = 245 // 256 - 11 bytes for PKCS#1 v1.5 padding

func encryptMessage(pubKey *rsa.PublicKey, msg []byte) []byte {
	msgLen := len(msg)
	if msgLen == 0 {
		return nil
	}

	// For small messages, encrypt directly
	if msgLen <= maxMessageLength {
		ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, pubKey, msg)
		if err != nil {
			fmt.Printf("Encryption error: %v\n", err)
			return nil
		}
		// Add length prefix for consistency with chunked messages
		result := make([]byte, 2+len(ciphertext))
		result[0] = byte(len(ciphertext) >> 8)
		result[1] = byte(len(ciphertext))
		copy(result[2:], ciphertext)
		return result
	}

	// For larger messages, split into chunks
	var encrypted []byte
	for i := 0; i < msgLen; i += maxMessageLength {
		end := i + maxMessageLength
		if end > msgLen {
			end = msgLen
		}

		chunk := msg[i:end]
		ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, pubKey, chunk)
		if err != nil {
			fmt.Printf("Chunk encryption error: %v\n", err)
			return nil
		}

		// Store chunk length and data
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
		// Check if we have enough bytes for the chunk header
		if i+2 > len(ciphertext) {
			fmt.Println("Invalid ciphertext: truncated chunk header")
			return nil
		}

		// Get chunk size from 2-byte header
		size := int(ciphertext[i])<<8 | int(ciphertext[i+1])
		i += 2

		// Validate chunk size
		if size <= 0 || size > 256 || i+size > len(ciphertext) {
			fmt.Printf("Invalid chunk size: %d (total length: %d, current position: %d)\n",
				size, len(ciphertext), i)
			return nil
		}

		// Extract and decrypt chunk
		chunk := ciphertext[i : i+size]
		plain, err := rsa.DecryptPKCS1v15(rand.Reader, privKey, chunk)
		if err != nil {
			fmt.Printf("Chunk decryption error at position %d (chunk size: %d): %v\n", i, size, err)
			return nil
		}

		decrypted = append(decrypted, plain...)
		i += size
	}

	return decrypted
}
