package handlers

import (
	"spendwise-backend/internal/database"
	"spendwise-backend/internal/models"

	"github.com/gofiber/fiber/v2"
)

func GetDashboardStats(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	// Query Parameters
	groupID := c.QueryInt("group_id", 0)
	memberID := c.QueryInt("member_id", 0)
	category := c.Query("category")
	startDate := c.Query("start_date") // YYYY-MM-DD
	endDate := c.Query("end_date")     // YYYY-MM-DD

	// Helper to find expenses
	var expenses []models.ExpenseRequest
	query := database.DB.Model(&models.ExpenseRequest{})

	if groupID > 0 {
		// Group View: Check if user is a member of the group
		var memberCount int64
		if err := database.DB.Model(&models.GroupMember{}).Where("group_id = ? AND user_id = ?", groupID, userID).Count(&memberCount).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error checking group membership"})
		}
		if memberCount == 0 {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "You are not a member of this group"})
		}
		query = query.Where("group_id = ?", groupID)

		// Optional Member Filter (only if in group view)
		if memberID > 0 {
			query = query.Where("requester_id = ?", memberID)
		}
	} else {
		// Personal View (Default): Only show my expenses
		query = query.Where("requester_id = ?", userID)
	}

	// Common Filters
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if startDate != "" && endDate != "" {
		query = query.Where("created_at BETWEEN ? AND ?", startDate+" 00:00:00", endDate+" 23:59:59")
	}

	// Execute Query
	if err := query.Find(&expenses).Error; err != nil {
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
