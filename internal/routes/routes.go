package routes

import (
	"spendwise-backend/internal/handlers"
	"spendwise-backend/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App) {
	api := app.Group("/api")

	// Health check endpoint (no authentication required)
	api.Get("/health", handlers.HealthCheck)

	// Auth
	auth := api.Group("/auth")
	auth.Post("/signup", handlers.Signup)
	auth.Post("/login", handlers.Login)
	auth.Get("/me", middleware.Protected(), handlers.GetMe)
	auth.Put("/profile", middleware.Protected(), handlers.UpdateProfile)
	auth.Post("/change-password", middleware.Protected(), handlers.ChangePassword)
	auth.Post("/avatar", middleware.Protected(), handlers.UploadAvatar)

	// Groups
	groups := api.Group("/groups", middleware.Protected())
	groups.Get("/invite/:code", handlers.GetGroupInfoByInvite)
	groups.Post("/", handlers.CreateGroup)
	groups.Get("/", handlers.ListGroups)
	groups.Post("/join", handlers.JoinGroup)
	groups.Get("/:id", handlers.GetGroup)
	groups.Get("/:id/members", handlers.GetGroupMembers)

	// Expenses
	expenses := api.Group("/expenses", middleware.Protected())
	expenses.Post("/", handlers.CreateExpense)
	expenses.Get("/", handlers.ListExpenses)
	expenses.Get("/:id", handlers.GetExpense)

	// Storage
	storage := api.Group("/storage", middleware.Protected())
	storage.Post("/upload", handlers.UploadAttachment)

	// Approvals
	approvals := api.Group("/approvals", middleware.Protected())
	approvals.Get("/", handlers.ListApprovals)
	approvals.Post("/:id/approve", handlers.ApproveExpense)
	approvals.Post("/:id/reject", handlers.RejectExpense)

	// Dashboard
	dashboard := api.Group("/dashboard", middleware.Protected())
	dashboard.Get("/stats", handlers.GetDashboardStats)
}
