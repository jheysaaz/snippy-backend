package redirect

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

// Simple HTTP to HTTPS redirect server
// This can run alongside your main HTTPS server to redirect HTTP traffic

func redirectToHTTPS(w http.ResponseWriter, r *http.Request) {
	// Get the host (remove port if present)
	host := r.Host
	if host == "" {
		host = "localhost"
	}

	// Build HTTPS URL
	httpsURL := fmt.Sprintf("https://%s%s", host, r.RequestURI)

	// 301 permanent redirect to HTTPS
	http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
}

func main() {
	// Get HTTP port from environment (default 80)
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "80"
	}

	// Setup HTTP server that redirects to HTTPS
	http.HandleFunc("/", redirectToHTTPS)

	log.Printf("HTTP to HTTPS redirect server starting on port %s", httpPort)
	log.Printf("All HTTP requests will be redirected to HTTPS")

	if err := http.ListenAndServe(":"+httpPort, nil); err != nil {
		log.Fatal("Failed to start HTTP redirect server:", err)
	}
}
