package main

// main.go wires up the HTTP server: static files, the WebSocket endpoint, the
// admin dashboard, and shared managers.

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/trytobebee/snake_go/pkg/game"
)

var detailedLogs = flag.Bool("detailed-logs", false, "Enable detailed session logging to database")

// addr is the HTTP listen address. Precedence: -addr flag > PORT env > :8501.
// Default is :8501 (not :8080) to avoid clashing with common local services.
var addr = flag.String("addr", defaultAddr(), "HTTP listen address, e.g. :8501")

func defaultAddr() string {
	if p := os.Getenv("PORT"); p != "" {
		return ":" + p
	}
	return ":8501"
}

// Shared managers.
var (
	lbManager   = game.NewLeaderboardManager()
	userManager = game.NewUserManager()
)

// upgrader upgrades HTTP connections to WebSocket, restricting cross-origin
// requests via checkOrigin.
var upgrader = websocket.Upgrader{CheckOrigin: checkOrigin}

// checkOrigin allows requests with no Origin header (native clients), requests
// whose Origin is in the ALLOWED_ORIGINS allowlist, or — when no allowlist is
// configured — same-origin requests only.
func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // non-browser client
	}

	if allowed := os.Getenv("ALLOWED_ORIGINS"); allowed != "" {
		for _, a := range strings.Split(allowed, ",") {
			if strings.TrimSpace(a) == origin {
				return true
			}
		}
		return false
	}

	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return u.Host == r.Host // same-origin
}

func main() {
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found, relying on system environment variables")
	}

	game.InitDB()

	// Static files.
	http.Handle("/", http.FileServer(http.Dir("web/static")))

	// Single source of truth: serve the proto schema to the frontend.
	http.HandleFunc("/proto/snake.proto", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "pkg/proto/snake.proto")
	})

	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/admin/feedback", adminFeedbackHandler)

	log.Printf("🚀 Snake Game Web Server starting on http://localhost%s\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
