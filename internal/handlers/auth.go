package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"spendwise-backend/internal/database"
	"spendwise-backend/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func Signup(c *fiber.Ctx) error {
	type SignupRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		FullName string `json:"full_name"`
	}

	var req SignupRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not hash password"})
	}

	user := models.User{
		Email:        req.Email,
		PasswordHash: string(hash),
		FullName:     req.FullName,
	}

	if result := database.DB.Create(&user); result.Error != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Could not create user. Email might be taken."})
	}

	return c.JSON(user)
}

func Login(c *fiber.Ctx) error {
	type LoginRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	var user models.User
	if result := database.DB.Where("email = ?", req.Email).First(&user); result.Error != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid credentials"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid credentials"})
	}

	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(time.Hour * 72).Unix(),
	})

	t, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not generate token"})
	}

	return c.JSON(fiber.Map{
		"token": t,
		"user":  user,
	})
}

func GetMe(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var user models.User
	if result := database.DB.First(&user, userID); result.Error != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	return c.JSON(user)
}

func UpdateProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	type UpdateProfileRequest struct {
		FullName string `json:"full_name"`
		Phone    string `json:"phone"`
	}

	var req UpdateProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	var user models.User
	if result := database.DB.First(&user, userID); result.Error != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	user.FullName = req.FullName
	user.Phone = req.Phone

	if err := database.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not update profile"})
	}

	return c.JSON(user)
}

func ChangePassword(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	type ChangePasswordRequest struct {
		CurrentPassword string `json:"currentPassword"` // Match frontend key
		NewPassword     string `json:"newPassword"`
	}

	var req ChangePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	var user models.User
	if result := database.DB.First(&user, userID); result.Error != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	// Verify current password (optional but recommended for security)
	// Note: Frontend might not send current password if it's a simple reset flow,
	// but for "Change Password" it usually does.
	// If the frontend sends it, we should check it.
	if req.CurrentPassword != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid current password"})
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not hash password"})
	}

	user.PasswordHash = string(hash)

	if err := database.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not update password"})
	}

	return c.JSON(fiber.Map{"message": "Password updated successfully"})
}

func UploadAvatar(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	file, err := c.FormFile("avatar")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No file uploaded"})
	}

	uploadDir := "./uploads/avatars"
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.MkdirAll(uploadDir, 0755)
	}

	filename := fmt.Sprintf("%d_%d_%s", userID, time.Now().Unix(), file.Filename)
	filepath := filepath.Join(uploadDir, filename)

	if err := c.SaveFile(file, filepath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not save file"})
	}

	var user models.User
	if result := database.DB.First(&user, userID); result.Error != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	// Construct public URL (assuming static file serving is set up)
	// Note: In production, this should be a full URL or relative path handled by frontend
	user.AvatarURL = "/uploads/avatars/" + filename

	if err := database.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not update avatar URL"})
	}

	return c.JSON(fiber.Map{"avatar_url": user.AvatarURL})
}
