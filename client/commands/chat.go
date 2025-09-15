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

	// --- 3. Get current user keys ---
	privKeyStr := os.Getenv("PRIVATE_KEY")
	pubKeyStr := os.Getenv("PUBLIC_KEY")

	if privKeyStr == "" || pubKeyStr == "" {
		reader := bufio.NewReader(os.Stdin)

		if privKeyStr == "" {
			fmt.Print("Enter your PRIVATE key (PEM format): ")
			p, _ := reader.ReadString('\n')
			privKeyStr = strings.TrimSpace(p)
			os.Setenv("PRIVATE_KEY", privKeyStr)
		}

		if pubKeyStr == "" {
			fmt.Print("Enter your PUBLIC key (PEM format): ")
			p, _ := reader.ReadString('\n')
			pubKeyStr = strings.TrimSpace(p)
			os.Setenv("PUBLIC_KEY", pubKeyStr)
		}
	}

	// --- 4. Get receiver's public key from server ---
	userInfo, exists := utils.GetUser(username, jwtToken)
	if !exists {
		fmt.Println("User not found:", username)
		return
	}
	receiverPubKeyStr := userInfo.PublicKey

	// --- 5. Parse RSA keys ---
	privKey := parsePrivateKey(privKeyStr)
	receiverPubKey := parsePublicKey(receiverPubKeyStr)

	// --- Ready to encrypt/decrypt messages ---
	fmt.Printf("Starting chat with %s...\n", username)

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("You: ")
		msg, _ := reader.ReadString('\n')
		msg = strings.TrimSpace(msg)

		// Encrypt message for receiver
		encrypted := encryptMessage(receiverPubKey, []byte(msg))
		fmt.Println("Encrypted message:", encrypted)

		// Decrypt message (for demonstration, using your own private key)
		decrypted := decryptMessage(privKey, encrypted)
		fmt.Println("Decrypted message (for self-check):", string(decrypted))
	}
}

// ------------------- RSA helpers -------------------

func parsePrivateKey(privPEM string) *rsa.PrivateKey {
	block, _ := pem.Decode([]byte(privPEM))
	if block == nil {
		fmt.Println("Invalid private key PEM")
		os.Exit(1)
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		fmt.Println("Failed to parse private key:", err)
		os.Exit(1)
	}
	return key
}

func parsePublicKey(pubPEM string) *rsa.PublicKey {
	block, _ := pem.Decode([]byte(pubPEM))
	if block == nil {
		fmt.Println("Invalid public key PEM")
		os.Exit(1)
	}
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		fmt.Println("Failed to parse public key:", err)
		os.Exit(1)
	}
	pubKey, ok := pubInterface.(*rsa.PublicKey)
	if !ok {
		fmt.Println("Invalid type for public key")
		os.Exit(1)
	}
	return pubKey
}

func encryptMessage(pubKey *rsa.PublicKey, msg []byte) []byte {
	ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, pubKey, msg)
	if err != nil {
		fmt.Println("Failed to encrypt message:", err)
		return nil
	}
	return ciphertext
}

func decryptMessage(privKey *rsa.PrivateKey, ciphertext []byte) []byte {
	plain, err := rsa.DecryptPKCS1v15(rand.Reader, privKey, ciphertext)
	if err != nil {
		fmt.Println("Failed to decrypt message:", err)
		return nil
	}
	return plain
}
