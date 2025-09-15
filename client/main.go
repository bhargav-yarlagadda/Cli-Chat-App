package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"chat-client/commands"

	"github.com/c-bata/go-prompt"
	"github.com/joho/godotenv"
)

func clearScreen() {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	default: // Linux, macOS
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}
func executor(input string) {
	godotenv.Load()
	input = strings.TrimSpace(input)
	if input == "" {
		return
	}

	args := strings.Split(input, " ")
	cmd := strings.ToLower(args[0])
	cmdArgs := args[1:]

	switch strings.ToLower(cmd) {
	case "register":
		commands.Register(cmdArgs)
	case "login":
		commands.Login(cmdArgs)
	case "add":
		commands.AddUser(cmdArgs)
	case "view-requests":
		commands.ViewPendingRequests()
	case "respond":
		commands.RespondToConnectionRequest(cmdArgs)
	case "chat":
		commands.Chat(cmdArgs)
	case "help":
		fmt.Println("\n=== Chat Application CLI Help ===")
		fmt.Println("\nAuthentication Commands:")
		fmt.Printf("%-20s : %s\n", "register", "Register a new user account")
		fmt.Printf("%-20s   %s\n", "", "Usage: register --username:yourname --password:yourpass")
		fmt.Printf("%-20s : %s\n", "login", "Login to your account")
		fmt.Printf("%-20s   %s\n", "", "Usage: login --username:yourname --password:yourpass")

		fmt.Println("\nConnection Management:")
		fmt.Printf("%-20s : %s\n", "add", "Send a connection request to another user")
		fmt.Printf("%-20s   %s\n", "", "Usage: add --username:targetuser")
		fmt.Printf("%-20s : %s\n", "view-requests", "View all pending connection requests")
		fmt.Printf("%-20s   %s\n", "", "Usage: view-requests")
		fmt.Printf("%-20s : %s\n", "respond", "Accept or reject a connection request")
		fmt.Printf("%-20s   %s\n", "", "Usage: respond --username:requester")

		fmt.Println("\nSystem Commands:")
		fmt.Printf("%-20s : %s\n", "clear", "Clear the terminal screen")
		fmt.Printf("%-20s : %s\n", "exit", "Logout and exit the application")
		fmt.Printf("%-20s : %s\n", "help", "Show this help message")

		fmt.Println("\nNote: Most commands require you to be logged in first.")

	case "clear":
		clearScreen()
	case "exit":
		os.Clearenv()
		os.Exit(0)
		return

	default:
		fmt.Println("Unknown command. Type 'help' for a list of commands.")
	}
}
func noCompleter(d prompt.Document) []prompt.Suggest {
	return []prompt.Suggest{} // return empty slice
}

func main() {
	fmt.Println("Welcome to Chat Application")
	fmt.Println("Type 'help' to see available commands")

	p := prompt.New(
		executor,
		noCompleter, // empty completer instead of nil
		prompt.OptionPrefix("> "),
		prompt.OptionTitle("CLI Chat App"),
		prompt.OptionHistory([]string{}),
	)
	p.Run()
}
