package handlers

import (
	"spendwise-backend/internal/database"
	"spendwise-backend/internal/models"
	"time"

	"github.com/gofiber/fiber/v2"
)

func GetWallet(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var user models.User
	if err := database.DB.Select("wallet_balance").First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	return c.JSON(fiber.Map{
		"balance": user.WalletBalance,
	})
}

func TopupWallet(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	type TopupRequest struct {
		Amount float64 `json:"amount"`
	}

	var req TopupRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.Amount <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Amount must be positive"})
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	tx := database.DB.Begin()

	user.WalletBalance += req.Amount
	user.UpdatedAt = time.Now()

	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not topup wallet"})
	}

	transaction := models.WalletTransaction{
		UserID:      userID,
		Amount:      req.Amount,
		Type:        "credit",
		Description: "Wallet Topup",
		CreatedAt:   time.Now(),
	}

	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not create transaction record"})
	}

	tx.Commit()

	return c.JSON(fiber.Map{
		"message": "Wallet topped up successfully",
		"balance": user.WalletBalance,
	})
}

func GetWalletTransactions(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	txType := c.Query("type")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	db := database.DB.Where("user_id = ?", userID)

	if txType != "" && txType != "all" {
		db = db.Where("type = ?", txType)
	}
	if startDate != "" {
		db = db.Where("created_at >= ?", startDate)
	}
	if endDate != "" {
		// Add 1 day to include the end date fully if it's just a date string
		db = db.Where("created_at <= ?::date + interval '1 day'", endDate)
	}

	var transactions []models.WalletTransaction
	if err := db.Order("created_at desc").Find(&transactions).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not fetch transactions"})
	}

	return c.JSON(transactions)
}
