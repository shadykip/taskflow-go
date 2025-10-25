package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestRootEndpoint(t *testing.T) {
	// Switch Gin to test mode
	gin.SetMode(gin.TestMode)

	// Setup
	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "TaskFlow API v1"})
	})

	// Perform request
	req, _ := http.NewRequest("GET", "/", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	// Assert
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "TaskFlow API v1")
}

func TestHealthEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/health", healthCheck)

	req, _ := http.NewRequest("GET", "/health", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	body := resp.Body.String()
	assert.Contains(t, body, `"status":"ok"`)
	assert.Contains(t, body, `"uptime"`)
}

func TestDatabaseConnection(t *testing.T) {
	// Reuse the same DSN as in main.go (in real life, use test DB — but for Day 3, this is fine)
	dsn := "host=localhost user=dev password=linspace dbname=taskflow port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	assert.NoError(t, err)

	// Ping the database
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, sqlDB.Ping())
}
func TestRegisterUser_ValidInput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/register", registerUser)

	jsonBody := `{"email":"user4@example.com","password":"mypassword"}`
	req, _ := http.NewRequest("POST", "/register", strings.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusCreated, resp.Code)
	assert.Contains(t, resp.Body.String(), "user4@example.com")
}

func TestRegisterUser_InvalidEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/register", registerUser)

	jsonBody := `{"email":"invalid","password":"123"}`
	req, _ := http.NewRequest("POST", "/register", strings.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "Invalid email")
}
func TestLogin_ValidCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/login", loginUser)

	// First, register a user (for test)
	db.Exec("INSERT INTO users (email, password, created_at) VALUES (?, ?, ?)",
		"login@test.com",
		"$2a$10$Xe4wnrV2VQYc8qJZxKZ5.eS6u1U1X6y7Y8Z9a0b1c2d3e4f5g6h7i8j9k", // pre-hashed "password123"
		time.Now())

	jsonBody := `{"email":"login@test.com","password":"password123"}`
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "token")
}

func TestProtectedRoute_WithoutToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/users", authMiddleware(), getUsers)

	req, _ := http.NewRequest("GET", "/users", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
}
