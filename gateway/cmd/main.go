package main

import (
	"gateway/internal/auth"
	"gateway/internal/limiter"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})

	// Initialize rate limiters
	rateLimiter, err := limiter.NewRateLimitMiddleware(redisClient, "sliding", 100, "1m")
	if err != nil {
		log.Fatalf("Failed to create rate limiter: %v", err)
	}

	concurrencyLimiter := limiter.NewConcurrencyMiddleware(redisClient, 10, 30*time.Second)

	// Initialize Gin router
	r := gin.Default()

	// Apply global middleware
	r.Use(rateLimiter)
	r.Use(concurrencyLimiter)
	r.Use(auth.AuthMiddleware())

	// Protected routes example
	protected := r.Group("/api")
	{
		// Admin only routes
		admin := protected.Group("/admin")
		admin.Use(auth.RequireRoles("admin"))
		{
			admin.GET("/stats", func(c *gin.Context) {
				c.JSON(200, gin.H{"status": "admin stats"})
			})
		}

		// User routes
		user := protected.Group("/user")
		user.Use(auth.RequireRoles("user", "admin"))
		{
			user.GET("/profile", func(c *gin.Context) {
				userID, _ := c.Get(auth.CtxUserKey)
				c.JSON(200, gin.H{"user_id": userID})
			})
		}
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Gateway starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
