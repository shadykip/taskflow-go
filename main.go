package main

import (
	"time"

	"github.com/gin-gonic/gin"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// User model
type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Email     string    `json:"email" gorm:"uniqueIndex;not null"`
	Password  string    `json:"-" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
}

var db *gorm.DB

func main() {
	// connect to db
	dsn := "host=localhost user=dev password=linspace dbname=taskflow port=5432 sslmode=disable"
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("❌ Failed to connect to database: " + err.Error())
	}
	err = db.AutoMigrate(&User{})
	if err != nil {
		panic("❌ Failed to migrate database: " + err.Error())
	}

	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "TaskFlow API v1",
		})
	})

	r.GET("/health", healthCheck)
	r.GET("/users", getUsers)

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

func getUsers(c *gin.Context) {
	var users []User

	result := db.Find(&users)
	if result.Error != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch users"})
		return
	}
	// Return only safe fields (password is omitted by JSON tag)
	c.JSON(200, users)
}
