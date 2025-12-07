package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"spendwise-backend/internal/database"
)

// HealthCheck returns the health status of the application
func HealthCheck(c *fiber.Ctx) error {
	// Basic health response
	status := fiber.Map{
		"status":    "healthy",
		"service":   "spendwise-backend",
		"timestamp": time.Now().Unix(),
	}

	// Check database connection
	if database.DB != nil {
		sqlDB, err := database.DB.DB()
		if err != nil {
			status["database"] = "error"
			status["database_error"] = err.Error()
			return c.Status(fiber.StatusServiceUnavailable).JSON(status)
		}

		if err := sqlDB.Ping(); err != nil {
			status["database"] = "disconnected"
			status["database_error"] = err.Error()
			return c.Status(fiber.StatusServiceUnavailable).JSON(status)
		}

		status["database"] = "connected"
	} else {
		status["database"] = "not_initialized"
	}

	return c.JSON(status)
}

// SimpleHealthCheck returns a minimal health check response
func SimpleHealthCheck(c *fiber.Ctx) error {
	return c.SendString("OK")
}
