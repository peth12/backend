package main

import (
	"log"
	"os"

	"spendwise-backend/internal/database"
	"spendwise-backend/internal/models"
	"spendwise-backend/internal/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Connect to Database
	database.Connect()

	// Run Migrations
	models.Migrate(database.DB)

	// Initialize Fiber
	app := fiber.New()

	// Middleware
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	// Routes
	routes.SetupRoutes(app)

	// Static files for uploads
	// Static files for uploads
	app.Static("/uploads", "./uploads")

	// Serve Frontend Static Files
	// Assuming the frontend is built to ../dist relative to the backend execution context
	app.Static("/", "../dist")

	// Handle SPA routing: Accessing non-API routes should return index.html
	app.Get("*", func(c *fiber.Ctx) error {
		// If it's an API route that wasn't matched, return 404 (already handled by routes above if strictly ordered, but let's be careful)
		path := c.Path()
		if len(path) >= 4 && path[:4] == "/api" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Endpoint not found"})
		}
		// Otherwise send index.html
		return c.SendFile("../dist/index.html")
	})

	// Start Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	log.Fatal(app.Listen(":" + port))
}
