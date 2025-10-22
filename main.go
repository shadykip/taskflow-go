package main

import (
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "TaskFlow API v1",
		})
	})

	r.GET("/health", healthCheck)

	r.Run(":8080")
}

var startTime = time.Now()

func healthCheck(c *gin.Context) {
	uptime := time.Since(startTime).Truncate(time.Second)
	c.JSON(200, gin.H{
		"status": "ok",
		"uptime": uptime.String(),
	})
}
