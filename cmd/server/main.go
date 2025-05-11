package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"rtcs/internal/config"
	"rtcs/internal/middleware"
	"rtcs/internal/repository"
	"rtcs/internal/service"
	"rtcs/internal/transport"
)

func main() {
	// Load configuration
	cfg := config.Get()

	// Connect to database
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Connect to Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: strings.TrimPrefix(cfg.RedisURL, "redis://"),
	})

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	chatRepo := repository.NewChatRepository(db)

	// Initialize services
	authService := service.NewAuthService(userRepo)
	chatService := service.NewChatService(chatRepo)
	statusService := service.NewStatusService(redisClient)
	profileService := service.NewProfileService(userRepo)

	// Initialize handlers
	authHandler := transport.NewAuthHandler(authService)
	chatHandler := transport.NewChatHandler(chatService)
	profileHandler := transport.NewProfileHandler(profileService)

	// Load OAuth config
	oauthCfg, err := config.LoadOAuthConfig()
	if err != nil {
		log.Printf("Warning: OAuth config not loaded: %v", err)
	}
	oauthHandler := transport.NewOAuthHandler(oauthCfg, authService)

	// Initialize WebSocket handler
	wsHandler := transport.NewWebSocketHandler(statusService, profileService)

	// Set up router
	router := mux.NewRouter()

	// Public routes
	router.HandleFunc("/health", transport.HealthCheck).Methods("GET")
	router.HandleFunc("/api/auth/register", authHandler.Register).Methods("POST")
	router.HandleFunc("/api/auth/login", authHandler.Login).Methods("POST")
	router.HandleFunc("/auth/google", oauthHandler.GoogleLogin).Methods("GET")
	router.HandleFunc("/auth/google/callback", oauthHandler.GoogleCallback).Methods("GET")

	// Protected routes
	apiRouter := router.PathPrefix("/api").Subrouter()
	apiRouter.Use(middleware.AuthMiddleware())

	// Chat routes
	apiRouter.HandleFunc("/chats", chatHandler.CreateChat).Methods("POST")
	apiRouter.HandleFunc("/chats", chatHandler.ListChats).Methods("GET")
	apiRouter.HandleFunc("/chats/{chatId}", chatHandler.GetChat).Methods("GET")
	apiRouter.HandleFunc("/chats/{chatId}/join", chatHandler.JoinChat).Methods("POST")
	apiRouter.HandleFunc("/chats/{chatId}/leave", chatHandler.LeaveChat).Methods("POST")

	// Profile routes
	apiRouter.HandleFunc("/profile", profileHandler.GetMyProfile).Methods("GET")
	apiRouter.HandleFunc("/profile", profileHandler.UpdateProfile).Methods("PUT")
	apiRouter.HandleFunc("/users/{userId}/profile", profileHandler.GetProfile).Methods("GET")

	// WebSocket route
	router.HandleFunc("/ws", wsHandler.HandleWebSocket)

	// Get certificate paths from environment variables or use defaults
	certPath := os.Getenv("TLS_CERT_PATH")
	if certPath == "" {
		certPath = "certs/server.crt"
	}

	keyPath := os.Getenv("TLS_KEY_PATH")
	if keyPath == "" {
		keyPath = "certs/server.key"
	}

	// Create an HTTP server with TLS configuration
	httpsPort := os.Getenv("HTTPS_PORT")
	if httpsPort == "" {
		httpsPort = "8443"
	}

	server := &http.Server{
		Addr:    ":" + httpsPort,
		Handler: router,
	}

	// Start HTTP to HTTPS redirect server
	go startHTTPRedirectServer()

	// Start the HTTPS server
	go func() {
		log.Printf("Starting HTTPS server on %s", server.Addr)
		if err := server.ListenAndServeTLS(certPath, keyPath); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not start HTTPS server: %v", err)
		}
	}()

	// Set up graceful shutdown
	gracefulShutdown(server)
}

// HTTP to HTTPS redirect server
func startHTTPRedirectServer() {
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	redirectHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		// If the host includes a port, strip it
		if i := strings.IndexByte(host, ':'); i >= 0 {
			host = host[:i]
		}

		httpsPort := os.Getenv("HTTPS_PORT")
		if httpsPort == "" {
			httpsPort = "8443"
		}

		// Construct the HTTPS URL
		httpsURL := "https://" + host + ":" + httpsPort + r.URL.String()
		http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
	})

	redirectServer := &http.Server{
		Addr:    ":" + httpPort,
		Handler: redirectHandler,
	}

	log.Printf("Starting HTTP redirect server on %s", redirectServer.Addr)
	if err := redirectServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("HTTP redirect server error: %v", err)
	}
}

// Graceful shutdown function
func gracefulShutdown(server *http.Server) {
	// Create channel to listen for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Block until interrupt signal is received
	<-stop

	log.Println("Shutting down server...")

	// Create a deadline for server shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server gracefully stopped")
}
