package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"spendwise-backend/internal/database"
	"spendwise-backend/internal/models"

	"github.com/gofiber/fiber/v2"
)

func CreateExpense(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	type CreateExpenseRequest struct {
		GroupID        uint    `json:"group_id"`
		Title          string  `json:"title"`
		Category       string  `json:"category"`
		Amount         float64 `json:"amount"`
		Description    string  `json:"description"`
		TargetUserID   *uint   `json:"target_user_id"`
		IsDirectRecord bool    `json:"is_direct_record"`
	}

	var req CreateExpenseRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Verify membership
	var member models.GroupMember
	if err := database.DB.Where("group_id = ? AND user_id = ?", req.GroupID, userID).First(&member).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Not a member of this group"})
	}

	status := "pending"
	var approvedBy *uint
	var approvedAt *time.Time

	if req.IsDirectRecord {
		status = "approved"
		approvedBy = &userID
		now := time.Now()
		approvedAt = &now
	}

	expense := models.ExpenseRequest{
		GroupID:      req.GroupID,
		RequesterID:  userID,
		Title:        req.Title,
		Category:     req.Category,
		Amount:       req.Amount,
		Description:  req.Description,
		Status:       status,
		TargetUserID: req.TargetUserID,
		ApprovedBy:   approvedBy,
		ApprovedAt:   approvedAt,
	}

	if err := database.DB.Create(&expense).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not create expense"})
	}

	return c.JSON(expense)
}

func ListExpenses(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	// Optional: Filter by status and group
	status := c.Query("status")
	groupID := c.Query("group_id")

	query := database.DB.Preload("Requester").Preload("TargetUser").Where("requester_id = ?", userID)

	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	if groupID != "" && groupID != "all" {
		query = query.Where("group_id = ?", groupID)
	}

	expenses := make([]models.ExpenseRequest, 0)
	if err := query.Order("created_at desc").Find(&expenses).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not fetch expenses"})
	}

	return c.JSON(expenses)
}

func GetExpense(c *fiber.Ctx) error {
	id := c.Params("id")
	var expense models.ExpenseRequest
	// Preload Attachments and ApprovalSlips
	if err := database.DB.Preload("Requester").Preload("Attachments").Preload("ApprovalSlips").First(&expense, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Expense not found"})
	}
	return c.JSON(expense)
}

func UploadAttachment(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	expenseIDStr := c.FormValue("expense_id")
	expenseID, err := strconv.Atoi(expenseIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid expense_id"})
	}

	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No file uploaded"})
	}

	// Ensure upload directory exists
	uploadDir := "./uploads"
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.Mkdir(uploadDir, 0755)
	}

	// Generate unique filename
	filename := fmt.Sprintf("%d_%d_%s", userID, time.Now().Unix(), file.Filename)
	filepath := filepath.Join(uploadDir, filename)

	if err := c.SaveFile(file, filepath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not save file"})
	}

	// Save record
	attachment := models.ExpenseAttachment{
		ExpenseID:  uint(expenseID),
		FileName:   file.Filename,
		FilePath:   "/uploads/" + filename,
		FileSize:   file.Size,
		FileType:   file.Header.Get("Content-Type"),
		UploadedBy: userID,
		UploadedAt: time.Now(),
	}

	if err := database.DB.Create(&attachment).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not save attachment record"})
	}

	return c.JSON(attachment)
}
