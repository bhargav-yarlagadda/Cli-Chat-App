package main

import (
	"chat-server/db"
	"chat-server/handlers"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)


func main(){
	err := godotenv.Load()
	PORT := os.Getenv("PORT")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	err =db.ConnectToDB()
	if err != nil{
	log.Fatal("Error in loading env: ",err)
	}
	app := fiber.New() 
	app.Get("/",func(c *fiber.Ctx) error{
		return c.JSON(fiber.Map{"message":"Hello world"})
	})
	AuthRoutes := app.Group("/auth")
	handlers.HandleAuth(AuthRoutes)

	ConnectionRoutes := app.Group("/connections")
	ConnectionRoutes.Use(handlers.JWTMiddleware()) // to validate the jwt sent by user
	handlers.HandleConnections(ConnectionRoutes)
	err=app.Listen(PORT)
	if err != nil { 
		log.Fatal("Error in staring the server ",err)
	}

} 



