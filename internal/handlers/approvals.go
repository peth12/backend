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
	fmt.Printf("DEBUG: Approve - UserID: %d, GroupID: %d\n", userID, expense.GroupID)
	if err := database.DB.Where("user_id = ? AND group_id = ? AND role IN ?", userID, expense.GroupID, []string{"approver", "admin"}).First(&role).Error; err != nil {
		fmt.Printf("DEBUG: Approve - Role Check Failed: %v\n", err)
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Not authorized to approve"})
	}
	fmt.Printf("DEBUG: Approve - Role Found: %s\n", role.Role)

	// Enforce Specific Approver if set
	if expense.TargetUserID != nil {
		fmt.Printf("DEBUG: Approve - TargetUserID: %d, CurrentUserID: %d\n", *expense.TargetUserID, userID)
		if *expense.TargetUserID != userID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Only the assigned approver can approve this expense"})
		}
	} else {
		fmt.Println("DEBUG: Approve - TargetUserID is nil (Anyone)")
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

	if err := database.DB.Save(&expense).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not approve expense"})
	}

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
	if err := database.DB.Where("user_id = ? AND group_id = ? AND role IN ?", userID, expense.GroupID, []string{"approver", "admin"}).First(&role).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Not authorized to reject"})
	}

	// Enforce Specific Approver if set
	if expense.TargetUserID != nil {
		fmt.Printf("DEBUG: Reject - TargetUserID: %d, CurrentUserID: %d\n", *expense.TargetUserID, userID)
		if *expense.TargetUserID != userID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Only the assigned approver can reject this expense"})
		}
	} else {
		fmt.Println("DEBUG: Reject - TargetUserID is nil (Anyone)")
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
