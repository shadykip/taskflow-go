package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

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

var jwtSecret = []byte("taskflow-secret-key-change-in-prod")

func generateToken(userID uint) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(), // 24h expiry
	})
	return token.SignedString(jwtSecret)
}

// Simulate async email sending
func sendWelcomeEmail(email string) {
	// todo: connect to SMTP, send HTML email, handle retries
	// For now: just log + simulate delay
	time.Sleep(500 * time.Millisecond) // simulate network delay
	fmt.Printf("[EMAIL SENT] Welcome email to: %s\n", email)
}

// Auth Middleware
func authMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			ctx.AbortWithStatusJSON(401, gin.H{"error": "Missing Auth Header"})
			return
		}
		// Expect: "Bearer <token>"
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			ctx.AbortWithStatusJSON(401, gin.H{"error": "Authorization header must be Bearer <token>"})
			return
		}
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			ctx.AbortWithStatusJSON(401, gin.H{"error": "Invalid or expired token"})
			return
		}
		// Extract user_id from token
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if userID, ok := claims["user_id"].(float64); ok {
				// Store user ID in context for handlers
				ctx.Set("user_id", uint(userID))
				ctx.Next()
				return
			}
		}

		ctx.AbortWithStatusJSON(401, gin.H{"error": "Invalid token claims"})
	}
}

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
	r.POST("/register", registerUser)
	r.POST("/login", loginUser)

	protected := r.Group("/")
	protected.Use(authMiddleware())
	{
		protected.GET("/users", getUsers)
		protected.GET("/me", getMe)
	}

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
		Email    string `json:"email" binding : "required"`
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
	go sendWelcomeEmail(user.Email)
	c.JSON(201, gin.H{
		"id":         user.ID,
		"email":      user.Email,
		"created_at": user.CreatedAt,
	})
}

func loginUser(c *gin.Context) {
	var input struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "Invalid JSON"})
		return
	}
	// find user by email
	var user User
	if err := db.Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(401, gin.H{"error": "Invalid email or password"})
		return
	}
	// check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		c.JSON(401, gin.H{"error": "Invalid email or password"})
		return
	}

	// Generate token
	token, err := generateToken(user.ID)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to generate token"})
		return
	}
	c.JSON(200, gin.H{
		"token": token,
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
		},
	})
}

func getMe(c *gin.Context) {
	//Get user_id from context set by authMiddleware
	userID, exists := c.Get("user_id")
	if !exists {
		c.AbortWithStatusJSON(500, gin.H{"error": "User ID not found in context"})
		return
	}
	var user User
	if err := db.First(&user, userID).Error; err != nil {
		c.AbortWithStatusJSON(404, gin.H{"error": "User not found"})
		return
	}
	c.JSON(200, gin.H{
		"id":         user.ID,
		"email":      user.Email,
		"created_at": user.CreatedAt,
	})
}
