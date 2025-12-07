package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"time"

	"spendwise-backend/internal/database"
	"spendwise-backend/internal/models"

	"github.com/gofiber/fiber/v2"
)

func generateInviteCode() string {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return "default"
	}
	return hex.EncodeToString(bytes)
}

func CreateGroup(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	type CreateGroupRequest struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	var req CreateGroupRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	group := models.ExpenseGroup{
		Name:        req.Name,
		Description: req.Description,
		InviteCode:  generateInviteCode(),
		CreatedBy:   userID,
	}

	tx := database.DB.Begin()

	if err := tx.Create(&group).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not create group"})
	}

	// Add creator as member
	member := models.GroupMember{
		GroupID:  group.ID,
		UserID:   userID,
		JoinedAt: time.Now(),
	}
	if err := tx.Create(&member).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not add member"})
	}

	// Add creator as admin
	role := models.UserRole{
		GroupID: group.ID,
		UserID:  userID,
		Role:    "admin",
	}
	if err := tx.Create(&role).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not add role"})
	}

	tx.Commit()

	return c.JSON(group)
}

func ListGroups(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var members []models.GroupMember
	if err := database.DB.Preload("Group").Where("user_id = ?", userID).Find(&members).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not fetch groups"})
	}

	groups := make([]map[string]interface{}, 0)
	for _, m := range members {
		// Get member count
		var count int64
		database.DB.Model(&models.GroupMember{}).Where("group_id = ?", m.GroupID).Count(&count)

		groups = append(groups, map[string]interface{}{
			"id":           m.Group.ID,
			"name":         m.Group.Name,
			"description":  m.Group.Description,
			"invite_code":  m.Group.InviteCode,
			"created_by":   m.Group.CreatedBy,
			"created_at":   m.Group.CreatedAt,
			"member_count": count,
		})
	}

	return c.JSON(groups)
}

func JoinGroup(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	type JoinRequest struct {
		InviteCode string `json:"invite_code"`
	}

	var req JoinRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	var group models.ExpenseGroup
	if err := database.DB.Where("invite_code = ?", req.InviteCode).First(&group).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Group not found"})
	}

	// Check if already member
	var existingMember models.GroupMember
	if err := database.DB.Where("group_id = ? AND user_id = ?", group.ID, userID).First(&existingMember).Error; err == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Already a member"})
	}

	tx := database.DB.Begin()

	member := models.GroupMember{
		GroupID:  group.ID,
		UserID:   userID,
		JoinedAt: time.Now(),
	}
	if err := tx.Create(&member).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not join group"})
	}

	// Default role: requester
	role := models.UserRole{
		GroupID: group.ID,
		UserID:  userID,
		Role:    "requester",
	}
	if err := tx.Create(&role).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not assign role"})
	}

	tx.Commit()

	return c.JSON(fiber.Map{"message": "Joined group successfully", "group": group})
}

func GetGroup(c *fiber.Ctx) error {
	// TODO: Implement get single group details if needed
	return c.SendStatus(fiber.StatusNotImplemented)
}

func GetGroupMembers(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	groupID := c.Params("id")

	// Verify membership
	var member models.GroupMember
	if err := database.DB.Where("group_id = ? AND user_id = ?", groupID, userID).First(&member).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Not a member of this group"})
	}

	var members []models.GroupMember
	if err := database.DB.Preload("User").Where("group_id = ?", groupID).Find(&members).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not fetch group members"})
	}

	var result []map[string]interface{}
	for _, m := range members {
		result = append(result, map[string]interface{}{
			"id":         m.User.ID,
			"full_name":  m.User.FullName,
			"email":      m.User.Email,
			"avatar_url": m.User.AvatarURL,
			"joined_at":  m.JoinedAt,
		})
	}

	return c.JSON(result)
}
func UpdateGroup(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	groupID := c.Params("id")

	// Verify Admin/Owner Permission
	var role models.UserRole
	if err := database.DB.Where("group_id = ? AND user_id = ? AND role = 'admin'", groupID, userID).First(&role).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Only admins can update group details"})
	}

	type UpdateGroupRequest struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	var req UpdateGroupRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	var group models.ExpenseGroup
	if err := database.DB.First(&group, groupID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Group not found"})
	}

	group.Name = req.Name
	group.Description = req.Description

	if err := database.DB.Save(&group).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not update group"})
	}

	return c.JSON(group)
}

func RemoveMember(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	groupID := c.Params("id")
	targetUserID := c.Params("userId") // ID of user to remove

	// Verify Admin/Owner Permission
	var adminRole models.UserRole
	if err := database.DB.Where("group_id = ? AND user_id = ? AND role = 'admin'", groupID, userID).First(&adminRole).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Only admins can remove members"})
	}

	// Prevent removing self (use LeaveGroup for that)
	if strconv.Itoa(int(userID)) == targetUserID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot kick yourself"})
	}

	tx := database.DB.Begin()

	// Remove from GroupMember
	if err := tx.Where("group_id = ? AND user_id = ?", groupID, targetUserID).Delete(&models.GroupMember{}).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not remove member"})
	}

	// Remove from UserRole
	if err := tx.Where("group_id = ? AND user_id = ?", groupID, targetUserID).Delete(&models.UserRole{}).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not remove member role"})
	}

	tx.Commit()

	return c.JSON(fiber.Map{"message": "Member removed successfully"})
}
