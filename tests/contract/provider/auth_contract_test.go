package provider_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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

func TestAuthProvider(t *testing.T) {
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
	err = db.AutoMigrate(&model.User{})
	assert.NoError(t, err)

	// Initialize repository and service
	userRepo := repository.NewUserRepository(db)
	authService := service.NewAuthService(userRepo)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/auth/register":
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			var req struct {
				Username string `json:"username"`
				Password string `json:"password"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}

			_, err := authService.Register(r.Context(), req.Username, req.Password)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{
				"message": "User registered successfully",
			})

		case "/auth/login":
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			var req struct {
				Username string `json:"username"`
				Password string `json:"password"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}

			token, err := authService.Login(r.Context(), req.Username, req.Password)
			if err != nil {
				http.Error(w, "Invalid credentials", http.StatusUnauthorized)
				return
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{
				"token": token,
			})
		}
	}))
	defer server.Close()

	// Define state handlers
	stateHandlers := types.StateHandlers{
		"User does not exist": func() error {
			// Delete test user if exists
			return db.Where("username = ?", "test@example.com").Delete(&model.User{}).Error
		},
		"User exists": func() error {
			// Create test user
			_, err := authService.Register(ctx, "test@example.com", "password123")
			return err
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
