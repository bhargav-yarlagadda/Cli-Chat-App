package commands

import (
	"bufio"
	"chat-client/utils"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"

	"regexp"
	"strings"

	"github.com/go-resty/resty/v2"
)

// var baseURL = "http://localhost:8080/auth"

// GenerateKeys generates a 2048-bit RSA key pair and returns PEM strings
func GenerateKeys() (privateKeyPEM string, publicKeyPEM string, err error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}

	privDER := x509.MarshalPKCS1PrivateKey(privateKey)
	privBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privDER,
	}
	privateKeyPEM = string(pem.EncodeToMemory(privBlock))

	pubDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", err
	}
	pubBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubDER,
	}
	publicKeyPEM = string(pem.EncodeToMemory(pubBlock))

	return
}

// Register handles CLI registration
func Register(args []string) {
	// Check if --help or -h is present
	helpRegex := regexp.MustCompile(`^--help$|^-h$`)
	for _, arg := range args {
		if helpRegex.MatchString(arg) {
			fmt.Println("Usage: register [--username:<username>] [--password:<password>]")
			fmt.Println("If no username/password is provided, you will be prompted interactively.")
			fmt.Println("A public/private key pair will be generated. The PRIVATE key will be displayed, store it safely!")
			return
		}
	}

	var username, password string

	// Parse CLI args if provided
	if len(args) >= 2 &&
		strings.HasPrefix(args[0], "--username:") &&
		strings.HasPrefix(args[1], "--password:") {

		username = strings.TrimPrefix(args[0], "--username:")
		password = strings.TrimPrefix(args[1], "--password:")
	} else {
		reader := bufio.NewReader(os.Stdin)

		// Interactive input
		fmt.Print("Enter username: ")
		usernameInput, _ := reader.ReadString('\n')
		username = strings.TrimSpace(usernameInput)

		fmt.Print("Enter password: ")
		passwordInput, _ := reader.ReadString('\n')
		password = strings.TrimSpace(passwordInput)
	}

	if strings.Contains(username, " ") || strings.Contains(password, " ") {
		fmt.Println("❌ Username and password cannot contain spaces.")
		return
	}

	// Generate RSA key pair
	privateKey, publicKey, err := GenerateKeys()
	if err != nil {
		log.Fatal("Failed to generate keys:", err)
	}

	// Send public key to server
	client := resty.New()
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]string{
			"username":   username,
			"password":   password,
			"public_key": publicKey,
		}).
		Post(utils.BaseURL + "/auth/register")

	if err != nil {
		log.Fatal("Request failed:", err)
	}

	// Check HTTP status
	if resp.IsSuccess() {
		// Create keys directory if it doesn't exist
		if err := os.MkdirAll("keys", 0700); err != nil {
			log.Fatal("Failed to create keys directory:", err)
		}

		// Save private key to file
		keyFileName := fmt.Sprintf("keys/%s_private.pem", username)
		if err := os.WriteFile(keyFileName, []byte(privateKey), 0600); err != nil {
			log.Fatal("Failed to save private key:", err)
		}

		fmt.Println("\n✅ Registration successful!")
		fmt.Println("----- IMPORTANT -----")
		fmt.Printf("Your private key has been saved to: %s\n", keyFileName)
		fmt.Println("Keep this file safe and secure. If lost, you will not be able to decrypt messages!")
		fmt.Println("Recommended: Back up this file in a secure location.")
	} else {
		fmt.Printf("❌ Registration failed: %s\n", resp.String())
	}
}
