package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"spendwise-backend/internal/database"
	"spendwise-backend/internal/models"
	"spendwise-backend/internal/services/slipok"

	"github.com/gofiber/fiber/v2"
)

func ListApprovals(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	// Find groups where user is approver or admin
	var roles []models.UserRole
	if err := database.DB.Where("user_id = ? AND role IN ?", userID, []string{"approver", "admin", "requester"}).Find(&roles).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not fetch roles"})
	}

	if len(roles) == 0 {
		fmt.Println("No roles found for user")
		return c.JSON([]models.ExpenseRequest{})
	}

	var groupIDs []uint
	for _, r := range roles {
		groupIDs = append(groupIDs, r.GroupID)
	}

	expenses := make([]models.ExpenseRequest, 0)
	if err := database.DB.Preload("Requester").Where("group_id IN ? AND status = ?", groupIDs, "pending").Order("created_at desc").Find(&expenses).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not fetch pending approvals"})
	}

	return c.JSON(expenses)
}

func ApproveExpense(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	expenseID := c.Params("id")

	type ApproveRequest struct {
		Notes string `json:"notes"`
	}
	// Note: For file upload, we might need to handle form data differently,
	// but for simplicity assuming separate upload or just notes here if no file.
	// If file is present, it should be handled similar to UploadAttachment but for approval_slips.

	var req ApproveRequest
	// Try parsing body if JSON, but if multipart it might fail or be empty.
	c.BodyParser(&req)

	var expense models.ExpenseRequest
	if err := database.DB.First(&expense, expenseID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Expense not found"})
	}

	// Verify permission
	var role models.UserRole
	// Check if user is in the group (any role)
	if err := database.DB.Where("user_id = ? AND group_id = ?", userID, expense.GroupID).First(&role).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Not a member of this group"})
	}

	// Enforce Specific Approver if set
	if expense.TargetUserID != nil {
		if *expense.TargetUserID != userID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Only the assigned approver can approve this expense"})
		}
	} else {
		// If no specific approver, anyone in the group can approve.
		// Optional: Block requester from approving their own request?
		// The requirement didn't specify blocking self-approval, but it's good practice.
		// However, "everyone in the group" might literally mean everyone.
		// I will keep self-approval allowed for now if they are not the target, or maybe block it?
		// Let's block self-approval unless they assigned it to themselves (which is separate logic).
		if expense.RequesterID == userID {
			// For now, allow it to be safe with "everyone", or blocking it might confuse if they are testing alone.
			// User prompt: "everyone within the group can approve".
		}
	}

	// Handle Slip Upload if present
	file, err := c.FormFile("file")
	if err == nil {
		uploadDir := "./uploads"
		if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
			os.Mkdir(uploadDir, 0755)
		}
		filename := fmt.Sprintf("slip_%d_%d_%s", userID, time.Now().Unix(), file.Filename)
		filePath := filepath.Join(uploadDir, filename)

		if err := c.SaveFile(file, filePath); err == nil {
			slip := models.ApprovalSlip{
				ExpenseID:  expense.ID,
				FileName:   file.Filename,
				FilePath:   "/uploads/" + filename,
				FileSize:   file.Size,
				FileType:   file.Header.Get("Content-Type"),
				Notes:      c.FormValue("notes"),
				UploadedBy: userID,
				UploadedAt: time.Now(),
			}

			// Verify with SlipOK
			// We do this synchronously to ensure we capture the result before responding
			// In a production high-load env, this might be better as a background job,
			// but for this use case, immediate feedback is valuable.
			resp, err := slipok.VerifySlip(filePath)
			if err == nil && resp.Success && resp.Data.Success {
				slip.IsVerified = true

				// Marshal relevant data to JSON string
				dataBytes, _ := json.Marshal(resp.Data)
				slip.SlipOKData = string(dataBytes)
			} else if err != nil {
				fmt.Printf("SlipOK Verification Failed: %v\n", err)
			}

			database.DB.Create(&slip)
		}
	}

	// Update Status
	now := time.Now()
	expense.Status = "approved"
	expense.ApprovedBy = &userID
	expense.ApprovedAt = &now

	// Deduct from Approver's Wallet
	var approver models.User
	if err := database.DB.First(&approver, userID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not find approver"})
	}

	approver.WalletBalance -= expense.Amount

	tx := database.DB.Begin()
	if err := tx.Save(&expense).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not approve expense"})
	}

	if err := tx.Save(&approver).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not update wallet balance"})
	}

	// Create Debit Transaction
	transaction := models.WalletTransaction{
		UserID:      userID,
		Amount:      expense.Amount,
		Type:        "debit",
		Description: fmt.Sprintf("Approved expense: %s", expense.Title),
		ReferenceID: expense.ID,
		CreatedAt:   time.Now(),
	}

	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not create transaction record"})
	}

	tx.Commit()

	return c.JSON(expense)
}

func RejectExpense(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	expenseID := c.Params("id")

	type RejectRequest struct {
		Reason string `json:"reason"`
	}

	var req RejectRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	var expense models.ExpenseRequest
	if err := database.DB.First(&expense, expenseID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Expense not found"})
	}

	// Verify permission
	var role models.UserRole
	// Check if user is in the group (any role)
	if err := database.DB.Where("user_id = ? AND group_id = ?", userID, expense.GroupID).First(&role).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Not a member of this group"})
	}

	// Enforce Specific Approver if set
	if expense.TargetUserID != nil {
		if *expense.TargetUserID != userID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Only the assigned approver can reject this expense"})
		}
	}

	// Update Status
	now := time.Now()
	expense.Status = "rejected"
	expense.ApprovedBy = &userID
	expense.ApprovedAt = &now
	expense.RejectionReason = req.Reason

	if err := database.DB.Save(&expense).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not reject expense"})
	}

	return c.JSON(expense)
}
