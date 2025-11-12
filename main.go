package main

import (
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

var db *sql.DB

func main() {
	// Get PostgreSQL connection string from environment variable
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/snippy?sslmode=disable"
	}

	// Connect to PostgreSQL
	var err error
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to PostgreSQL:", err)
	}

	// Configure connection pool for performance
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(0)               // Reuse connections indefinitely
	db.SetConnMaxIdleTime(5 * time.Minute) // Close idle connections after 5 minutes

	// Ping database to verify connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping PostgreSQL:", err)
	}

	log.Println("Successfully connected to PostgreSQL")

	// Initialize database schema
	if err := initDatabase(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Start background cleanup job for expired refresh tokens
	go startTokenCleanupJob()

	// Ensure cleanup on exit
	defer func() {
		closeErr := db.Close()
		if closeErr != nil {
			log.Println("Error closing database connection:", closeErr)
		}
	}()

	// Setup Gin router
	router := gin.Default()

	// Set max request body size to prevent DoS (10MB limit)
	router.MaxMultipartMemory = 10 << 20 // 10 MB

	// CORS middleware for Chrome extension
	router.Use(corsMiddleware())

	// Rate limiting
	// General rate limit: 100 requests per minute per IP
	generalLimiter := NewRateLimiter(100, 100)
	// Strict rate limit for auth endpoints: 5 requests per minute per IP
	strictLimiter := NewRateLimiter(5, 5)

	// API routes
	api := router.Group("/api/v1")
	api.Use(RateLimitMiddleware(generalLimiter)) // Apply general rate limit to all API routes
	{
		// Health check (public)
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		// Authentication routes (public with strict rate limiting)
		auth := api.Group("/auth")
		auth.Use(StrictRateLimitMiddleware(strictLimiter))
		{
			auth.POST("/login", login)
			auth.POST("/refresh", refreshAccessToken) // Exchange refresh token for new access token
			auth.POST("/logout", logout)              // Revoke single refresh token
		}

		// Protected auth routes
		authProtected := api.Group("/auth")
		authProtected.Use(AuthMiddleware())
		{
			authProtected.POST("/logout-all", logoutAll) // Revoke all tokens for user
		}

		// Public user routes (strict rate limiting for registration)
		api.POST("/users", StrictRateLimitMiddleware(strictLimiter), createUser)

		// Protected user routes (require authentication)
		users := api.Group("/users")
		users.Use(AuthMiddleware())
		{
			users.GET("", getUsers)
			users.GET("/:id", getUser)
			users.GET("/username/:username", getUserByUsername)
			users.PUT("/:id", updateUser)    // Can only update own profile
			users.DELETE("/:id", deleteUser) // Can only delete own account
			users.GET("/:id/snippets", getUserSnippets)
		}

		// Snippet routes with optional auth (can view without auth, need auth to create/update/delete)
		snippets := api.Group("/snippets")
		{
			snippets.GET("", getSnippets)    // Public - anyone can view
			snippets.GET("/:id", getSnippet) // Public - anyone can view

			// Protected routes (require authentication)
			snippets.POST("", AuthMiddleware(), createSnippet)
			snippets.PUT("/:id", AuthMiddleware(), updateSnippet)
			snippets.DELETE("/:id", AuthMiddleware(), deleteSnippet)
		}
	}

	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Check if SSL/TLS certificates are available
	sslCert := os.Getenv("SSL_CERT_FILE")
	sslKey := os.Getenv("SSL_KEY_FILE")

	if sslCert != "" && sslKey != "" {
		// Run with HTTPS
		log.Printf("Server starting with HTTPS on port %s", port)
		log.Printf("Using SSL cert: %s", sslCert)
		log.Printf("Using SSL key: %s", sslKey)

		if err := router.RunTLS(":"+port, sslCert, sslKey); err != nil {
			log.Printf("Failed to start HTTPS server: %v", err)
			return
		}
	} else {
		// Run with HTTP
		log.Printf("Server starting with HTTP on port %s", port)
		log.Println("To enable HTTPS, set SSL_CERT_FILE and SSL_KEY_FILE environment variables")

		if err := router.Run(":" + port); err != nil {
			log.Printf("Failed to start HTTP server: %v", err)
			return
		}
	}
}

// CORS middleware to allow Chrome extension requests
func corsMiddleware() gin.HandlerFunc {
	// Get allowed origins from environment variable, default to * for development
	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "*" // Default for development
		log.Println("WARNING: CORS set to allow all origins (*). Set CORS_ALLOWED_ORIGINS in production!")
	}

	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", allowedOrigins)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
