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

	switch cmd {
	case "register":
		commands.Register(cmdArgs)
	case "login":
		commands.Login(cmdArgs)
	case "help":
	fmt.Println("\n=== Chat Application CLI Help ===")
	fmt.Printf("%-15s : %s\n", "register", "Register a new user (generates public/private key pair)")
	fmt.Printf("%-15s : %s\n", "login", "Login with your credentials and get a session JWT")
	fmt.Printf("%-15s : %s\n", "clear", "Clear the terminal screen")
	fmt.Printf("%-15s : %s\n", "exit", "Logout and clear current session JWT (optional command)")
	fmt.Printf("%-15s : %s\n", "help", "Show this help message\n")

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
