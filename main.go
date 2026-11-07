package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v3"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	app := fiber.New()	

	app.Get("/", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "This app is running",
		})
	})

	log.Fatal(app.Listen(":" + port))
}
