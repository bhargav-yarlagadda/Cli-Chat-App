package handlers

import (
	"chat-server/db"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"

	"golang.org/x/crypto/bcrypt"
)

// ---------------- User Model ----------------
// The User struct represents the "users" table in the database.
// GORM automatically maps the struct to a table named "users" (snake_case plural of struct name).
// Fields with gorm tags define column properties in the database.
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`                 // Primary key column, auto-increment
	Username  string    `gorm:"uniqueIndex;not null" json:"username"` // Unique and required column
	Password  string    `gorm:"not null" json:"password"`             // Required column, will store hashed passwords
	PublicKey string    `gorm:"not null" json:"public_key"`           // Required column for storing public key
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`     // Automatically set when a new row is created
}

// ---------------- Register ----------------
// Handles user registration.
// Steps:
// 1. Parse request body into User struct.
// 2. Hash password before saving.
// 3. Insert user into "users" table using GORM's Create method.
// 4. Returns success or error response.
func register(c *fiber.Ctx) error {
	conn := db.DB_Conn
	if conn == nil {
		log.Fatal("Unable to connect to db")
	}

	body := new(User)
	if err := c.BodyParser(body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body " + err.Error()})
	}

	// Validate username format: only letters, numbers, underscores
	matched, err := regexp.MatchString(`^[a-zA-Z0-9_]+$`, body.Username)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Regex error " + err.Error()})
	}
	if !matched {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Username can only contain letters, numbers, and underscores"})
	}

	// Check uniqueness
	var existing db.User
	if err := conn.Where("username = ?", body.Username).First(&existing).Error; err == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Username already taken"})
	}

	// Hash the password before storing
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to hash password " + err.Error()})
	}
	body.Password = string(hashedPassword)

	// Save the user into the "users" table
	if err := conn.Create(&body).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to register user " + err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "User registered successfully"})
}

// ---------------- Login ----------------
// Handles user login.
// Steps:
// 1. Parse request body (username and password).
// 2. Search the "users" table using GORM Where + First to find the user.
//   - GORM automatically uses the struct type (User) to determine the table ("users").
//
// 3. Compare the hashed password with bcrypt.
// 4. Generate a JWT token if credentials are valid.
func login(c *fiber.Ctx) error {
	type LoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	req := new(LoginRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body " + err.Error()})
	}

	var user User
	matched, err := regexp.MatchString(`^[a-zA-Z0-9_]+$`, req.Username)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Regex error " + err.Error()})
	}
	if !matched {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Username can only contain letters, numbers, and underscores"})
	}
	// Search the "users" table for a user with matching username
	if err := db.DB_Conn.Where("username = ?", req.Username).First(&user).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid credentials " + err.Error()})
	}

	// Compare password hash
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid credentials " + err.Error()})
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"exp":      time.Now().Add(time.Hour * 72).Unix(), // Token expires in 3 days
	})

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "secret" // fallback (not recommended in production)
	}

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate token " + err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Login successful",
		"token":   tokenString,
	})
}

// ---------------- Validate ----------------
// Validates the JWT token sent in the Authorization header.
// Steps:
// 1. Extract token from header.
// 2. Parse and verify JWT using the secret.
// 3. Return valid/invalid response.


func validate(c *fiber.Ctx) error {

	tokenStr := c.Get("Authorization")
	if tokenStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing token"})
	}

	secret := os.Getenv("JWT_SECRET")
	fmt.Println(secret)
	if secret == "" {
		fmt.Printf("Missing Secret")
	}
	const prefix = "Bearer "
	if len(tokenStr) > len(prefix) && tokenStr[:len(prefix)] == prefix {
		tokenStr = tokenStr[len(prefix):]
	}

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fiber.ErrUnauthorized
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token " + err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Token is valid"})
}

func getUserByUsername(c *fiber.Ctx) error {
	username := c.Query("username")
	if username == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Please provide username"})
	}

	var user db.User
	// Query the database for the user
	if err := db.DB_Conn.Where("username = ?", username).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	// Return only safe fields
	return c.JSON(fiber.Map{
		"user": map[string]interface{}{
			"id":         user.ID,
			"username":   user.Username,
			"public_key": user.PublicKey,
			"created_at": user.CreatedAt,
		},
	})
}

func JWTMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		tokenStr := c.Get("Authorization")
		if tokenStr == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing token"})
		}

		const prefix = "Bearer "
		if len(tokenStr) > len(prefix) && tokenStr[:len(prefix)] == prefix {
			tokenStr = tokenStr[len(prefix):]
		}

		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			fmt.Println("Missing JWT_SECRET")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Server misconfigured"})
		}

		// Parse the token
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.ErrUnauthorized
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
		}

		// Store claims in Fiber Locals
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			c.Locals("user", claims)
		}

		// Call next handler
		return c.Next()
	}
}
// ---------------- Register Routes ----------------
// Maps endpoints to handlers
func HandleAuth(router fiber.Router) {
	router.Post("/login", login)
	router.Post("/register", register)
	router.Get("/validate", validate)
	router.Get("/user-info",JWTMiddleware(),getUserByUsername)
}
