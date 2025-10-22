package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
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
