package handlers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSignup(t *testing.T) {
	setupTestDB()
	app := setupApp()
	app.Post("/auth/signup", Signup)

	t.Run("Success", func(t *testing.T) {
		payload := map[string]string{
			"email":     "test@example.com",
			"password":  "password123",
			"full_name": "Test User",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/auth/signup", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("Duplicate Email", func(t *testing.T) {
		// Create first user
		payload := map[string]string{
			"email":     "duplicate@example.com",
			"password":  "password123",
			"full_name": "First User",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/auth/signup", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		app.Test(req)

		// Try creating same user again
		req2 := httptest.NewRequest("POST", "/auth/signup", bytes.NewReader(body))
		req2.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req2)

		assert.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
	})
}

func TestLogin(t *testing.T) {
	setupTestDB()
	app := setupApp()
	app.Post("/auth/signup", Signup)
	app.Post("/auth/login", Login)

	// Create a user first
	signupPayload := map[string]string{
		"email":     "login@example.com",
		"password":  "password123",
		"full_name": "Login User",
	}
	signupBody, _ := json.Marshal(signupPayload)
	app.Test(httptest.NewRequest("POST", "/auth/signup", bytes.NewReader(signupBody)))

	t.Run("Success", func(t *testing.T) {
		payload := map[string]string{
			"email":    "login@example.com",
			"password": "password123",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("Invalid Password", func(t *testing.T) {
		payload := map[string]string{
			"email":    "login@example.com",
			"password": "wrongpassword",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode)
	})
}
