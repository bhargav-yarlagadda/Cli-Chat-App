package commands

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"chat-client/utils"

	"github.com/go-resty/resty/v2"
)

func AddUser(args []string) {
	// Check if user is logged in
	if token := os.Getenv("JWT_TOKEN"); token != "" {
		JWTToken = token
	} else {
		fmt.Println("You must login first using the login command.")
		return
	}

	// Check for --help
	helpRegex := regexp.MustCompile(`^--help$|^-h$`)
	for _, arg := range args {
		if helpRegex.MatchString(arg) {
			fmt.Println("Usage: add-user [--username:<username>]")
			fmt.Println("If no username is provided, you will be prompted interactively.")
			return
		}
	}

	var username string

	// Parse CLI args
	if len(args) >= 1 && strings.HasPrefix(args[0], "--username:") {
		username = strings.TrimPrefix(args[0], "--username:")
	} else {
		// Interactive prompt
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter username to connect: ")
		u, _ := reader.ReadString('\n')
		username = strings.TrimSpace(u)
	}

	// Make request to server
	client := resty.New()
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", "Bearer "+JWTToken).
		SetBody(map[string]string{
			"username": username,
		}).
		Post(utils.BaseURL + "/connections/connect") // fixed route

	if err != nil {
		log.Fatal("Connection request failed:", err)
	}

	if resp.StatusCode() != 200 {
		fmt.Println("Failed to send connection request:", resp.String())
		return
	}

	fmt.Println("Connection request sent successfully!")
}
