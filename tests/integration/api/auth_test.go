package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"rtcs/internal/middleware"
	"rtcs/internal/model"
	"rtcs/internal/repository"
	"rtcs/internal/service"
	"rtcs/internal/transport"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type testServer struct {
	router  *mux.Router
	db      *gorm.DB
	cleanup func()
}

func setupTestServer(t *testing.T) *testServer {
	// Start PostgreSQL container
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "postgres:15",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "test",
		},
		WaitingFor: wait.ForAll(
			wait.ForLog("database system is ready to accept connections"),
			wait.ForListeningPort("5432/tcp"),
		),
	}

	postgresContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	// Get container host and port
	host, err := postgresContainer.Host(ctx)
	require.NoError(t, err)
	port, err := postgresContainer.MappedPort(ctx, "5432")
	require.NoError(t, err)

	// Connect to database
	dsn := "host=" + host + " port=" + port.Port() + " user=test password=test dbname=test sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	// Run migrations
	err = db.AutoMigrate(&model.User{}, &model.Message{}, &model.Chat{}, &model.ChatUser{})
	require.NoError(t, err)

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	chatRepo := repository.NewChatRepository(db)

	// Initialize services
	authService := service.NewAuthService(userRepo)
	messageService := service.NewMessageService(messageRepo, nil)
	chatService := service.NewChatService(chatRepo)

	// Initialize handlers
	authHandler := transport.NewAuthHandler(authService)
	messageHandler := transport.NewMessageHandler(messageService)
	chatHandler := transport.NewChatHandler(chatService)

	// Create router
	router := mux.NewRouter()

	// Add middleware
	router.Use(middleware.CORS)
	router.Use(middleware.Logging)
	router.Use(middleware.Recover)
	router.Use(middleware.Metrics)

	// Register routes
	authRouter := router.PathPrefix("/auth").Subrouter()
	authRouter.HandleFunc("/register", authHandler.Register).Methods("POST")
	authRouter.HandleFunc("/login", authHandler.Login).Methods("POST")

	chatRouter := router.PathPrefix("/chats").Subrouter()
	chatRouter.Use(middleware.Auth)
	chatRouter.HandleFunc("", chatHandler.CreateChat).Methods("POST")
	chatRouter.HandleFunc("", chatHandler.ListChats).Methods("GET")
	chatRouter.HandleFunc("/{chatId}", chatHandler.GetChat).Methods("GET")
	chatRouter.HandleFunc("/{chatId}/join", chatHandler.JoinChat).Methods("POST")
	chatRouter.HandleFunc("/{chatId}/leave", chatHandler.LeaveChat).Methods("POST")

	messageRouter := router.PathPrefix("/messages").Subrouter()
	messageRouter.Use(middleware.Auth)
	messageRouter.HandleFunc("", messageHandler.Send).Methods("POST")
	messageRouter.HandleFunc("/{messageId}", messageHandler.DeleteMessage).Methods("DELETE")
	messageRouter.HandleFunc("/chat/{chatId}", messageHandler.GetChatHistory).Methods("GET")

	// Create cleanup function
	cleanup := func() {
		sqlDB, err := db.DB()
		if err == nil {
			sqlDB.Close()
		}
		postgresContainer.Terminate(ctx)
	}

	return &testServer{
		router:  router,
		db:      db,
		cleanup: cleanup,
	}
}

func TestRegister(t *testing.T) {
	server := setupTestServer(t)
	defer server.cleanup()

	tests := []struct {
		name           string
		payload        map[string]string
		expectedStatus int
	}{
		{
			name: "successful registration",
			payload: map[string]string{
				"email":    "test@example.com",
				"password": "password123",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "invalid email",
			payload: map[string]string{
				"email":    "invalid-email",
				"password": "password123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "short password",
			payload: map[string]string{
				"email":    "test@example.com",
				"password": "short",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			payload, err := json.Marshal(tt.payload)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(payload))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			rr := httptest.NewRecorder()

			// Send request
			server.router.ServeHTTP(rr, req)

			// Assert response
			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedStatus == http.StatusCreated {
				var response map[string]interface{}
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response["message"])
			}
		})
	}
}

func TestLogin(t *testing.T) {
	server := setupTestServer(t)
	defer server.cleanup()

	// Register a test user first
	registerPayload := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	}
	payload, err := json.Marshal(registerPayload)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusCreated, rr.Code)

	tests := []struct {
		name           string
		payload        map[string]string
		expectedStatus int
	}{
		{
			name: "successful login",
			payload: map[string]string{
				"email":    "test@example.com",
				"password": "password123",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "wrong password",
			payload: map[string]string{
				"email":    "test@example.com",
				"password": "wrongpassword",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "user not found",
			payload: map[string]string{
				"email":    "nonexistent@example.com",
				"password": "password123",
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			payload, err := json.Marshal(tt.payload)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(payload))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			rr := httptest.NewRecorder()

			// Send request
			server.router.ServeHTTP(rr, req)

			// Assert response
			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response["token"])
			}
		})
	}
}
