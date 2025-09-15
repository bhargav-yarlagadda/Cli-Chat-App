package middleware

import (
	"chat-server/db"
	"fmt"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// ValidateConnection ensures JWT is valid and sender exists in DB
func ValidateConnection() fiber.Handler {
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

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.ErrUnauthorized
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token claims"})
		}

		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user_id in token"})
		}

		var sender db.User
		if err := db.DB_Conn.First(&sender, uint(userIDFloat)).Error; err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Sender not found"})
		}

		c.Locals("sender", &sender)
		return c.Next()
	}
}
