// Package main starts the Snippy API HTTP server.
package main

import (
	"log"
	"os"
	"time"

	"github.com/jheysaaz/snippy-backend/app/auth"
	"github.com/jheysaaz/snippy-backend/app/database"
	"github.com/jheysaaz/snippy-backend/app/handlers"
	"github.com/jheysaaz/snippy-backend/app/middleware"
	_ "github.com/jheysaaz/snippy-backend/docs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Snippy API
// @version 1.0
// @description Code snippets management API with authentication
// @host localhost:8080
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter "Bearer {token}"
func main() {
	// Initialize database
	if err := database.Init(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Initialize prepared statements for better query performance
	if err := database.InitPreparedStatements(); err != nil {
		log.Printf("Warning: prepared statements not initialized: %v", err)
		// Continue without prepared statements (fallback to regular queries)
	}

	// Start data retention cleanup job (runs every 24 hours)
	go startDataRetentionCleanup()

	// Start token cleanup job (optional background task)
	// go models.StartTokenCleanupJob()

	// Ensure cleanup on exit
	defer func() {
		if database.DB != nil {
			if err := database.DB.Close(); err != nil {
				log.Printf("error closing database: %v", err)
			}
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

	// Swagger docs
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API routes
	api := r.Group("/api/v1")
	{
		// Authentication routes (with strict rate limiting)
		authRoutes := api.Group("/auth")
		authRoutes.Use(middleware.StrictRateLimitMiddleware(strictLimiter))
		{
			authRoutes.POST("/register", handlers.CreateUser)
			authRoutes.POST("/login", handlers.Login)
			authRoutes.GET("/availability", handlers.CheckAvailability)
			authRoutes.POST("/refresh", handlers.RefreshAccessToken)
			authRoutes.POST("/logout", handlers.Logout)
			authRoutes.POST("/logout-all", handlers.LogoutAll)
		}

		// Protected auth routes (require authentication)
		protectedAuth := api.Group("/auth")
		protectedAuth.Use(auth.Middleware())
		{
			// Sessions endpoints restricted to tester/premium/admin users
			protectedAuth.GET("/sessions", middleware.SessionsAccess, handlers.GetSessions)
			protectedAuth.POST("/sessions/:sessionId", middleware.SessionsAccess, handlers.LogoutSession)
		}

		// Public role routes
		api.GET("/roles", handlers.GetAllRoles)

		// Protected routes (require authentication)
		protected := api.Group("")
		protected.Use(auth.Middleware())
		{
			// User routes
			users := protected.Group("/users")
			{
				users.GET("/", handlers.GetUsers)
				users.GET("/profile", handlers.GetCurrentUser)
				users.PUT("/profile", handlers.UpdateCurrentUser)
				users.GET("/me/roles", handlers.GetMyRoles)
				users.GET("/:id", handlers.GetUser)
				users.PUT("/:id", handlers.UpdateUser)
				users.DELETE("/:id", handlers.DeleteUser)
			}

			// Snippet routes
			snippets := protected.Group("/snippets")
			{
				snippets.GET("/", handlers.GetCurrentUserSnippets)
				snippets.GET("/sync", handlers.SyncSnippets)
				snippets.POST("/", handlers.CreateSnippet)
				snippets.GET("/:id", handlers.GetSnippet)
				snippets.PUT("/:id", handlers.UpdateSnippet)
				snippets.DELETE("/:id", handlers.DeleteSnippet)
				snippets.GET("/:id/history", handlers.GetSnippetHistory)
				snippets.POST("/:id/restore/:versionNumber", handlers.RestoreSnippetVersion)
			}

			// Admin-only routes
			admin := protected.Group("/admin")
			admin.Use(middleware.AdminOnly)
			{
				// Role management
				admin.GET("/users/:userId/roles", handlers.GetUserRoles)
				admin.POST("/users/:userId/roles", handlers.AssignUserRole)
				admin.DELETE("/users/:userId/roles/:roleName", handlers.RevokeUserRole)
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
		log.Printf("Failed to start server: %v", err)
	}
}

// startDataRetentionCleanup runs the data retention cleanup job every 24 hours
func startDataRetentionCleanup() {
	// Run cleanup immediately on startup
	policy := database.DefaultRetentionPolicy()
	if err := database.CleanupOldData(policy); err != nil {
		log.Printf("Initial data cleanup failed: %v", err)
	}

	// Schedule cleanup to run every 24 hours
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("Running scheduled data retention cleanup...")
		if err := database.CleanupOldData(policy); err != nil {
			log.Printf("Scheduled data cleanup failed: %v", err)
		}
	}
}
