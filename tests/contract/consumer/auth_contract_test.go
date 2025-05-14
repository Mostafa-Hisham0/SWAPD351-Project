package consumer_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/pact-foundation/pact-go/dsl"
	"github.com/stretchr/testify/assert"
)

func TestAuthContract(t *testing.T) {
	// Create Pact client
	pact := &dsl.Pact{
		Consumer: "RTCS-Client",
		Provider: "RTCS-Server",
	}

	// Start Pact server
	pact.Setup(true)

	// Clean up after test
	defer pact.Teardown()

	t.Run("Register User", func(t *testing.T) {
		// Define the expected request and response
		pact.
			AddInteraction().
			Given("User does not exist").
			UponReceiving("A request to register a new user").
			WithRequest(dsl.Request{
				Method: "POST",
				Path:   dsl.String("/auth/register"),
				Headers: dsl.MapMatcher{
					"Content-Type": dsl.String("application/json"),
				},
				Body: map[string]interface{}{
					"email":    "test@example.com",
					"password": "password123",
				},
			}).
			WillRespondWith(dsl.Response{
				Status: 201,
				Headers: dsl.MapMatcher{
					"Content-Type": dsl.String("application/json"),
				},
				Body: map[string]interface{}{
					"message": dsl.String("User registered successfully"),
				},
			})

		// Execute the test
		err := pact.Verify(func() error {
			// Create request body
			reqBody := map[string]string{
				"email":    "test@example.com",
				"password": "password123",
			}
			jsonBody, err := json.Marshal(reqBody)
			if err != nil {
				return err
			}

			// Create request
			req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/auth/register", pact.Server.Port), bytes.NewBuffer(jsonBody))
			if err != nil {
				return err
			}

			req.Header.Set("Content-Type", "application/json")

			// Send request
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			// Assert response
			assert.Equal(t, 201, resp.StatusCode)

			var response map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&response)
			if err != nil {
				return err
			}

			assert.Equal(t, "User registered successfully", response["message"])
			return nil
		})

		assert.NoError(t, err)
	})

	t.Run("Login User", func(t *testing.T) {
		// Define the expected request and response
		pact.
			AddInteraction().
			Given("User exists").
			UponReceiving("A request to login").
			WithRequest(dsl.Request{
				Method: "POST",
				Path:   dsl.String("/auth/login"),
				Headers: dsl.MapMatcher{
					"Content-Type": dsl.String("application/json"),
				},
				Body: map[string]interface{}{
					"email":    "test@example.com",
					"password": "password123",
				},
			}).
			WillRespondWith(dsl.Response{
				Status: 200,
				Headers: dsl.MapMatcher{
					"Content-Type": dsl.String("application/json"),
				},
				Body: map[string]interface{}{
					"token": dsl.String("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."),
				},
			})

		// Execute the test
		err := pact.Verify(func() error {
			// Create request body
			reqBody := map[string]string{
				"email":    "test@example.com",
				"password": "password123",
			}
			jsonBody, err := json.Marshal(reqBody)
			if err != nil {
				return err
			}

			// Create request
			req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/auth/login", pact.Server.Port), bytes.NewBuffer(jsonBody))
			if err != nil {
				return err
			}

			req.Header.Set("Content-Type", "application/json")

			// Send request
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			// Assert response
			assert.Equal(t, 200, resp.StatusCode)

			var response map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&response)
			if err != nil {
				return err
			}

			assert.NotEmpty(t, response["token"])
			return nil
		})

		assert.NoError(t, err)
	})

	t.Run("Login Invalid Credentials", func(t *testing.T) {
		// Define the expected request and response
		pact.
			AddInteraction().
			Given("User exists").
			UponReceiving("A request to login with invalid credentials").
			WithRequest(dsl.Request{
				Method: "POST",
				Path:   dsl.String("/auth/login"),
				Headers: dsl.MapMatcher{
					"Content-Type": dsl.String("application/json"),
				},
				Body: map[string]interface{}{
					"email":    "test@example.com",
					"password": "wrongpassword",
				},
			}).
			WillRespondWith(dsl.Response{
				Status: 401,
				Headers: dsl.MapMatcher{
					"Content-Type": dsl.String("application/json"),
				},
				Body: map[string]interface{}{
					"error": dsl.String("Invalid credentials"),
				},
			})

		// Execute the test
		err := pact.Verify(func() error {
			// Create request body
			reqBody := map[string]string{
				"email":    "test@example.com",
				"password": "wrongpassword",
			}
			jsonBody, err := json.Marshal(reqBody)
			if err != nil {
				return err
			}

			// Create request
			req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/auth/login", pact.Server.Port), bytes.NewBuffer(jsonBody))
			if err != nil {
				return err
			}

			req.Header.Set("Content-Type", "application/json")

			// Send request
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			// Assert response
			assert.Equal(t, 401, resp.StatusCode)

			var response map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&response)
			if err != nil {
				return err
			}

			assert.Equal(t, "Invalid credentials", response["error"])
			return nil
		})

		assert.NoError(t, err)
	})
}
