package provider_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/pact-foundation/pact-go/dsl"
	"github.com/pact-foundation/pact-go/types"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	gormPostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"

	"rtcs/internal/model"
	"rtcs/internal/repository"
	"rtcs/internal/service"
)

func TestWebSocketProvider(t *testing.T) {
	// Create Pact client
	pact := &dsl.Pact{
		Consumer: "RTCS-Client",
		Provider: "RTCS-Server",
	}

	// Start Pact server
	pact.Setup(true)

	// Clean up after test
	defer pact.Teardown()

	// Start PostgreSQL container
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "postgres:14-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "test",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp"),
	}

	postgresC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	assert.NoError(t, err)
	defer postgresC.Terminate(ctx)

	// Get PostgreSQL connection details
	host, err := postgresC.Host(ctx)
	assert.NoError(t, err)
	port, err := postgresC.MappedPort(ctx, "5432")
	assert.NoError(t, err)

	// Connect to PostgreSQL
	dsn := "postgres://test:test@" + host + ":" + port.Port() + "/test?sslmode=disable"
	db, err := gorm.Open(gormPostgres.Open(dsn), &gorm.Config{})
	assert.NoError(t, err)

	// Run migrations
	err = db.AutoMigrate(&model.User{}, &model.Chat{}, &model.ChatUser{}, &model.Message{})
	assert.NoError(t, err)

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	messageRepo := repository.NewMessageRepository(db)

	// Initialize services
	authService := service.NewAuthService(userRepo)
	messageService := service.NewMessageService(messageRepo, nil)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Upgrade HTTP connection to WebSocket
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}

		// Get token from query parameter
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "Missing token", http.StatusUnauthorized)
			return
		}

		// Validate token
		userID, err := authService.ValidateToken(r.Context(), token)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Upgrade connection
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, "Failed to upgrade connection", http.StatusInternalServerError)
			return
		}
		defer conn.Close()

		// Handle WebSocket messages
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				break
			}

			// Parse message
			var msg struct {
				Type    string          `json:"type"`
				Payload json.RawMessage `json:"payload"`
			}
			if err := json.Unmarshal(message, &msg); err != nil {
				continue
			}

			// Handle message based on type
			switch msg.Type {
			case "message":
				var payload struct {
					ChatID  string `json:"chatId"`
					Content string `json:"content"`
				}
				if err := json.Unmarshal(msg.Payload, &payload); err != nil {
					continue
				}

				// Create message
				_, err := messageService.SendMessage(r.Context(), payload.ChatID, userID.String(), payload.Content)
				if err != nil {
					continue
				}

				// Send acknowledgment
				conn.WriteJSON(map[string]string{
					"type":    "message_ack",
					"payload": "Message sent successfully",
				})
			}
		}
	}))
	defer server.Close()

	// Define state handlers
	stateHandlers := types.StateHandlers{
		"User is authenticated": func() error {
			// Create test user and get token
			_, err := authService.Register(ctx, "test@example.com", "password123")
			return err
		},
		"User is not authenticated": func() error {
			// Delete test user if exists
			return db.Where("username = ?", "test@example.com").Delete(&model.User{}).Error
		},
		"Chat does not exist": func() error {
			// Delete test chat if exists
			return db.Where("name = ?", "Test Chat").Delete(&model.Chat{}).Error
		},
	}

	// Verify provider against consumer contracts
	_, err = pact.VerifyProvider(t, types.VerifyRequest{
		Provider:        "RTCS-Server",
		ProviderBaseURL: server.URL,
		StateHandlers:   stateHandlers,
	})

	assert.NoError(t, err)
}
