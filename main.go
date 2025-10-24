package main

import (
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

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

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func isValidEmail(email string) bool {
	return emailRegex.MatchString(strings.TrimSpace(email))
}

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
	r.POST("/register", registerUser)

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

func registerUser(c *gin.Context) {
	//parse input
	var input struct {
		Email    string `json:"Email" binding : "required"`
		Password string `json:"password" binding:"required,min=6`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "Invalid JSON or missing fields"})
	}
	if !isValidEmail(input.Email) {
		c.JSON(400, gin.H{"error": "Invalid email format"})
		return
	}

	//Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to hash password"})
		return
	}

	user := User{
		Email:    input.Email,
		Password: string(hashedPassword),
	}
	result := db.Create(&user)
	if result.Error != nil {
		// Handle duplicate email (PostgreSQL unique constraint)
		if strings.Contains(result.Error.Error(), "duplicate key") {
			c.JSON(400, gin.H{"error": "Email already registered"})
			return
		}
		c.JSON(500, gin.H{"error": "Database error"})
		return
	}
	c.JSON(201, gin.H{
		"id":         user.ID,
		"email":      user.Email,
		"created_at": user.CreatedAt,
	})
}
