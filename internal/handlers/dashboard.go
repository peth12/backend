package handlers

import (
	"spendwise-backend/internal/database"
	"spendwise-backend/internal/models"

	"github.com/gofiber/fiber/v2"
)

func GetDashboardStats(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	// Fetch all expenses for the user
	var expenses []models.ExpenseRequest
	if err := database.DB.Where("requester_id = ?", userID).Find(&expenses).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not fetch stats"})
	}

	totalExpenses := len(expenses)
	var pendingCount, approvedCount, rejectedCount int
	var totalAmount float64
	categoryMap := make(map[string]struct {
		Count  int
		Amount float64
	})
	monthlyMap := make(map[string]float64)

	for _, e := range expenses {
		totalAmount += e.Amount

		switch e.Status {
		case "pending":
			pendingCount++
		case "approved":
			approvedCount++
		case "rejected":
			rejectedCount++
		}

		// Category Data
		cat := categoryMap[e.Category]
		cat.Count++
		cat.Amount += e.Amount
		categoryMap[e.Category] = cat

		// Monthly Data
		month := e.CreatedAt.Format("Jan")
		monthlyMap[month] += e.Amount
	}

	// Format for frontend
	var categoryData []map[string]interface{}
	for k, v := range categoryMap {
		categoryData = append(categoryData, map[string]interface{}{
			"name":   k,
			"value":  v.Count,
			"amount": v.Amount,
		})
	}

	var monthlyData []map[string]interface{}
	// Note: This simple map iteration doesn't guarantee order.
	// For production, we should sort by month index.
	for k, v := range monthlyMap {
		monthlyData = append(monthlyData, map[string]interface{}{
			"month":  k,
			"amount": v,
		})
	}

	return c.JSON(fiber.Map{
		"totalExpenses": totalExpenses,
		"pendingCount":  pendingCount,
		"approvedCount": approvedCount,
		"rejectedCount": rejectedCount,
		"totalAmount":   totalAmount,
		"categoryData":  categoryData,
		"monthlyData":   monthlyData,
	})
}
