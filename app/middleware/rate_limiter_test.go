package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func TestNewRateLimiter(t *testing.T) {
	tests := []struct {
		name    string
		rate    rate.Limit
		burst   int
		wantErr bool
	}{
		{"valid limiter", rate.Limit(10), 20, false},
		{"low rate", rate.Limit(1), 1, false},
		{"high rate", rate.Limit(1000), 1000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := NewRateLimiter(tt.rate, tt.burst)
			if limiter == nil {
				t.Error("NewRateLimiter() returned nil")
			}
		})
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		rate           rate.Limit
		burst          int
		requests       int
		expectRejected bool
	}{
		{"within limit", rate.Limit(10), 10, 5, false},
		{"at limit", rate.Limit(2), 2, 2, false},
		{"exceed limit", rate.Limit(1), 1, 3, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := NewRateLimiter(tt.rate, tt.burst)
			router := gin.New()
			router.Use(RateLimitMiddleware(limiter))
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "ok")
			})

			rejectedCount := 0
			for i := 0; i < tt.requests; i++ {
				w := httptest.NewRecorder()
				req, _ := http.NewRequestWithContext(context.Background(), "GET", "/test", nil)
				req.RemoteAddr = "192.168.1.1:12345"
				router.ServeHTTP(w, req)

				if w.Code == http.StatusTooManyRequests {
					rejectedCount++
				}
			}

			if tt.expectRejected && rejectedCount == 0 {
				t.Error("Expected requests to be rejected but none were")
			}
			if !tt.expectRejected && rejectedCount > 0 {
				t.Errorf("Expected no rejections but got %d", rejectedCount)
			}
		})
	}
}

func TestStrictRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewRateLimiter(rate.Limit(2), 2)
	router := gin.New()
	router.Use(StrictRateLimitMiddleware(limiter))
	router.POST("/login", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// First 2 requests should succeed
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequestWithContext(context.Background(), "POST", "/login", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d failed with status %d", i+1, w.Code)
		}
	}

	// 3rd request should be rate limited
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "/login", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", w.Code)
	}
}

func TestRateLimiterDifferentIPs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewRateLimiter(rate.Limit(1), 1)
	router := gin.New()
	router.Use(RateLimitMiddleware(limiter))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Different IPs should have independent limits
	ips := []string{"192.168.1.1:12345", "192.168.1.2:12345", "192.168.1.3:12345"}

	for _, ip := range ips {
		w := httptest.NewRecorder()
		req, _ := http.NewRequestWithContext(context.Background(), "GET", "/test", nil)
		req.RemoteAddr = ip
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request from %s failed with status %d", ip, w.Code)
		}
	}
}

func TestVisitorCleanup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewRateLimiter(rate.Limit(10), 10)
	router := gin.New()
	router.Use(RateLimitMiddleware(limiter))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Make a request to create a visitor
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	router.ServeHTTP(w, req)

	// Wait a bit and make another request to trigger cleanup logic
	time.Sleep(100 * time.Millisecond)

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.101:12345"
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w2.Code)
	}
}
