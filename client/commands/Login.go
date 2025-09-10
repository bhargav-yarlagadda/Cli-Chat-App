package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/go-resty/resty/v2"
	"chat-client/utils"
	"regexp"
)

var JWTToken string // store token for session

func Login(args []string) {
	if token := os.Getenv("JWT_TOKEN"); token != "" {
		JWTToken = token
		fmt.Println("You are already logged in! JWT token is active for this session.")
		return
	}

	// Check for --help
	helpRegex := regexp.MustCompile(`^--help$|^-h$`)
	for _, arg := range args {
		if helpRegex.MatchString(arg) {
			fmt.Println("Usage: login [--username:<username>] [--password:<password>]")
			fmt.Println("If no username/password is provided, you will be prompted interactively.")
			return
		}
	}

	var username, password string

	// Parse CLI args
	if len(args) >= 2 &&
		strings.HasPrefix(args[0], "--username:") &&
		strings.HasPrefix(args[1], "--password:") {

		username = strings.TrimPrefix(args[0], "--username:")
		password = strings.TrimPrefix(args[1], "--password:")
	} else {
		// Interactive prompt
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Enter username: ")
		u, _ := reader.ReadString('\n')
		username = strings.TrimSpace(u)

		fmt.Print("Enter password: ")
		p, _ := reader.ReadString('\n')
		password = strings.TrimSpace(p)
	}

	// Call backend
	client := resty.New()
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]string{
			"username": username,
			"password": password,
		}).
		Post(utils.BaseURL + "/auth/login")

	if err != nil {
		log.Fatal("Login request failed:", err)
	}

	if resp.StatusCode() != 200 {
		fmt.Println("Login failed:", resp.String())
		return
	}

	// Manually unmarshal JSON
	type LoginResp struct {
		Message string `json:"message"`
		Token   string `json:"token"`
	}

	var res LoginResp
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		log.Fatal("Failed to parse login response:", err)
	}

	JWTToken = res.Token // store for session
	os.Setenv("JWT_TOKEN", JWTToken) // optional: makes it available as env variable

	fmt.Println("Login successful! JWT stored for session.")
	pendingResp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(JWTToken).
		Get(utils.BaseURL + "/connections/pending/count")

	if err != nil {
		log.Println("Failed to fetch pending requests count:", err)
		return
	}

	if pendingResp.StatusCode() != 200 {
		log.Println("Could not get pending requests:", pendingResp.String())
		return
	}

	var pendingData struct {
		PendingCount int `json:"pending_count"`
	}

	if err := json.Unmarshal(pendingResp.Body(), &pendingData); err != nil {
		log.Println("Failed to parse pending count:", err)
		return
	}

	if pendingData.PendingCount > 0 {
		fmt.Printf("You have %d pending connection request(s)!\n please use `view-requests` command to view pending connection requests\n", pendingData.PendingCount)
	}
}
