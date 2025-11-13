package main

import (
	"log"
	"os"

	"github.com/jheysaaz/snippy-backend/app/auth"
	"github.com/jheysaaz/snippy-backend/app/database"
	"github.com/jheysaaz/snippy-backend/app/handlers"
	"github.com/jheysaaz/snippy-backend/app/middleware"

	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize database
	if err := database.Init(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Start token cleanup job (optional background task)
	// go models.StartTokenCleanupJob()

	// Ensure cleanup on exit
	defer func() {
		if database.DB != nil {
			database.DB.Close()
		}
	}()

	log.Println("Starting Snippy API server...")

	// Set Gin mode based on environment
	env := os.Getenv("GIN_MODE")
	if env == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize Gin router
	r := gin.New()

	// Add middleware
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// CORS middleware with environment-specific origins
	corsOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if corsOrigins == "" {
		corsOrigins = "http://localhost:3000" // Default for development
	}
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", corsOrigins)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Rate limiting middleware
	generalLimiter := middleware.NewRateLimiter(100, 100)
	r.Use(middleware.RateLimitMiddleware(generalLimiter))
	strictLimiter := middleware.NewRateLimiter(5, 5)

	// Health endpoint
	r.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API routes
	api := r.Group("/api/v1")
	{
		// Authentication routes (with strict rate limiting)
		authRoutes := api.Group("/auth")
		authRoutes.Use(middleware.StrictRateLimitMiddleware(strictLimiter))
		{
			authRoutes.POST("/register", handlers.CreateUser)
			authRoutes.POST("/login", handlers.Login)
			authRoutes.POST("/refresh", handlers.RefreshAccessToken)
			authRoutes.POST("/logout", handlers.Logout)
		}

		// Protected routes (require authentication)
		protected := api.Group("")
		protected.Use(auth.AuthMiddleware())
		{
			// User routes
			users := protected.Group("/users")
			{
				users.GET("/", handlers.GetUsers)
				users.GET("/profile", handlers.GetCurrentUser)
				users.PUT("/profile", handlers.UpdateCurrentUser)
				users.GET("/:id", handlers.GetUser)
				users.PUT("/:id", handlers.UpdateUser)
				users.DELETE("/:id", handlers.DeleteUser)
			}

			// Snippet routes
			snippets := protected.Group("/snippets")
			{
				snippets.GET("/", handlers.GetUserSnippets)
				snippets.POST("/", handlers.CreateSnippet)
				snippets.GET("/:id", handlers.GetSnippet)
				snippets.PUT("/:id", handlers.UpdateSnippet)
				snippets.DELETE("/:id", handlers.DeleteSnippet)
			}
		}
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server running on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
