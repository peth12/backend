package handlers

import (
	"spendwise-backend/internal/database"
	"spendwise-backend/internal/models"

	"github.com/gofiber/fiber/v2"
)

// ... existing handlers ...

func GetGroupInfoByInvite(c *fiber.Ctx) error {
	inviteCode := c.Params("code")
	// Optional: Get user ID if logged in to check membership, but this endpoint might be public or protected.
	// The frontend calls it with auth token, so we can check membership.
	userID := c.Locals("user_id").(uint)

	var group models.ExpenseGroup
	if err := database.DB.Where("invite_code = ?", inviteCode).First(&group).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Group not found"})
	}

	var member models.GroupMember
	isMember := false
	if err := database.DB.Where("group_id = ? AND user_id = ?", group.ID, userID).First(&member).Error; err == nil {
		isMember = true
	}

	return c.JSON(fiber.Map{
		"group":     group,
		"is_member": isMember,
	})
}
