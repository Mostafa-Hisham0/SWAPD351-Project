package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"rtcs/internal/cache"
	"rtcs/internal/config"
	"rtcs/internal/middleware"
	"rtcs/internal/repository"
	"rtcs/internal/service"
	"rtcs/internal/transport"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile)
}

func main() {
	log.Printf("Starting server...")

	// Initialize configuration
	cfg := config.Get()
	log.Printf("Configuration loaded")

	// Connect to PostgreSQL
	db, err := connectDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	// Set DB connection pool limits
	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.SetMaxOpenConns(50)
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetConnMaxLifetime(time.Hour)
	}
	log.Printf("Connected to database")

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	chatRepo := repository.NewChatRepository(db)
	log.Printf("Repositories initialized")

	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisURL,
	})
	messageCache := cache.NewMessageCache(rdb)
	log.Printf("Connected to Redis")

	// Initialize services
	authService := service.NewAuthService(userRepo)
	messageService := service.NewMessageService(messageRepo, messageCache)
	chatService := service.NewChatService(chatRepo)
	log.Printf("Services initialized")

	// Initialize handlers
	authHandler := transport.NewAuthHandler(authService)
	messageHandler := transport.NewMessageHandler(messageService)
	chatHandler := transport.NewChatHandler(chatService)

	// Create router
	router := mux.NewRouter()

	// WebSocket endpoint (register before middleware)
	wsHandler := transport.NewWebSocketHandler()
	router.HandleFunc("/ws", wsHandler.HandleWebSocket)
	log.Printf("WebSocket endpoint added")

	// Add middleware first
	router.Use(middleware.CORS)
	router.Use(middleware.Logging)
	router.Use(middleware.Recover)
	router.Use(middleware.Metrics)

	// Health check endpoint
	router.HandleFunc("/health", transport.HealthCheck).Methods("GET")

	// Metrics endpoint
	router.Handle("/metrics", promhttp.Handler())

	// Auth routes
	authRouter := router.PathPrefix("/auth").Subrouter()
	authRouter.HandleFunc("/register", authHandler.Register).Methods("POST")
	authRouter.HandleFunc("/login", authHandler.Login).Methods("POST")

	// Chat routes (protected)
	chatRouter := router.PathPrefix("/chats").Subrouter()
	chatRouter.Use(middleware.Auth)
	chatRouter.HandleFunc("", chatHandler.CreateChat).Methods("POST")
	chatRouter.HandleFunc("", chatHandler.ListChats).Methods("GET")
	chatRouter.HandleFunc("/{chatId}", chatHandler.GetChat).Methods("GET")
	chatRouter.HandleFunc("/{chatId}/join", chatHandler.JoinChat).Methods("POST")
	chatRouter.HandleFunc("/{chatId}/leave", chatHandler.LeaveChat).Methods("POST")

	// Message routes (protected)
	messageRouter := router.PathPrefix("/messages").Subrouter()
	messageRouter.Use(middleware.Auth)
	messageRouter.HandleFunc("", messageHandler.Send).Methods("POST")
	messageRouter.HandleFunc("/{messageId}", messageHandler.DeleteMessage).Methods("DELETE")
	messageRouter.HandleFunc("/chat/{chatId}", messageHandler.GetChatHistory).Methods("GET")

	// Serve static files from the public directory (must be last)
	staticRouter := router.PathPrefix("/").Subrouter()
	staticRouter.PathPrefix("/").Handler(http.FileServer(http.Dir("public")))

	// Log all registered routes
	log.Printf("=== Registered Routes ===")
	err = router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err != nil {
			pathTemplate = "<no template>"
		}
		methods, err := route.GetMethods()
		if err != nil {
			methods = []string{"ANY"}
		}
		log.Printf("Route: %s [%s]", pathTemplate, methods)
		return nil
	})
	if err != nil {
		log.Printf("Error walking routes: %v", err)
	}
	log.Printf("======================")

	// Create server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Start server
	go func() {
		log.Printf("Server is running on port 8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Shutdown server gracefully
	log.Println("Server is shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}

func connectDB(url string) (*gorm.DB, error) {
	log.Printf("Connecting to database...")
	db, err := gorm.Open(postgres.Open(url), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	log.Printf("Connected to database")

	// Skip auto-migration since we're using manual migrations
	// if err := db.AutoMigrate(&model.User{}, &model.Message{}, &model.Chat{}, &model.ChatUser{}); err != nil {
	//     return nil, fmt.Errorf("failed to migrate database: %w", err)
	// }
	// log.Printf("Auto-migrations completed")

	return db, nil
}
