package middleware

import (
	"chat-server/db"
	"fmt"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

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

		// Parse and validate the token
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.ErrUnauthorized
			}
			return []byte(secret), nil
		})
		if err != nil {
			fmt.Printf("Token parsing error: %v\n", err)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
		}
		if !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Token is not valid"})
		}

		// Extract claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token claims"})
		}

		// Debug print claims
		fmt.Printf("Token claims: %+v\n", claims)

		// Get user ID from claims
		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID in token"})
		}
		userID := uint(userIDFloat)

		// Fetch user from DB
		var user db.User
		if err := db.DB_Conn.First(&user, userID).Error; err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User not found"})
		}

		// Store user information in context for handlers
		c.Locals("user", claims)   // Store claims in the same way as other handlers expect
		c.Locals("userID", userID) // Also store parsed userID for convenience
		c.Locals("userDB", &user)  // Store full user record

		return c.Next()
	}
}
