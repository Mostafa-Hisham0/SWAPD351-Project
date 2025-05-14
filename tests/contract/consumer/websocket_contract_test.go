package consumer_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/pact-foundation/pact-go/dsl"
	"github.com/stretchr/testify/assert"
)

func TestWebSocketContract(t *testing.T) {
	// Create Pact client
	pact := &dsl.Pact{
		Consumer: "RTCS-Client",
		Provider: "RTCS-Server",
	}

	// Start Pact server
	pact.Setup(true)

	// Clean up after test
	defer pact.Teardown()

	t.Run("Connect and Send Message", func(t *testing.T) {
		// Define the expected request and response
		pact.
			AddInteraction().
			Given("User is authenticated").
			UponReceiving("A WebSocket connection request").
			WithRequest(dsl.Request{
				Method: "GET",
				Path:   dsl.String("/ws"),
				Headers: dsl.MapMatcher{
					"Upgrade":               dsl.String("websocket"),
					"Connection":            dsl.String("Upgrade"),
					"Sec-WebSocket-Version": dsl.String("13"),
					"Authorization":         dsl.String("Bearer valid-token"),
				},
			}).
			WillRespondWith(dsl.Response{
				Status: 101,
				Headers: dsl.MapMatcher{
					"Upgrade":    dsl.String("websocket"),
					"Connection": dsl.String("Upgrade"),
				},
			})

		// Execute the test
		err := pact.Verify(func() error {
			// Create WebSocket connection
			wsURL := fmt.Sprintf("ws://localhost:%d/ws", pact.Server.Port)
			header := http.Header{}
			header.Add("Authorization", "Bearer valid-token")

			conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
			if err != nil {
				return err
			}
			defer conn.Close()

			// Send a message
			message := map[string]interface{}{
				"type":    "message",
				"chat_id": "123",
				"content": "Hello, World!",
			}
			err = conn.WriteJSON(message)
			if err != nil {
				return err
			}

			// Read response
			var response map[string]interface{}
			err = conn.ReadJSON(&response)
			if err != nil {
				return err
			}

			// Assert response
			assert.Equal(t, "message", response["type"])
			assert.Equal(t, "123", response["chat_id"])
			assert.Equal(t, "Hello, World!", response["content"])
			return nil
		})

		assert.NoError(t, err)
	})

	t.Run("Connect with Invalid Token", func(t *testing.T) {
		// Define the expected request and response
		pact.
			AddInteraction().
			Given("User is not authenticated").
			UponReceiving("A WebSocket connection request with invalid token").
			WithRequest(dsl.Request{
				Method: "GET",
				Path:   dsl.String("/ws"),
				Headers: dsl.MapMatcher{
					"Upgrade":               dsl.String("websocket"),
					"Connection":            dsl.String("Upgrade"),
					"Sec-WebSocket-Version": dsl.String("13"),
					"Authorization":         dsl.String("Bearer invalid-token"),
				},
			}).
			WillRespondWith(dsl.Response{
				Status: 401,
				Headers: dsl.MapMatcher{
					"Content-Type": dsl.String("application/json"),
				},
				Body: map[string]interface{}{
					"error": dsl.String("Invalid token"),
				},
			})

		// Execute the test
		err := pact.Verify(func() error {
			// Create WebSocket connection
			wsURL := fmt.Sprintf("ws://localhost:%d/ws", pact.Server.Port)
			header := http.Header{}
			header.Add("Authorization", "Bearer invalid-token")

			_, _, err := websocket.DefaultDialer.Dial(wsURL, header)
			assert.Error(t, err)
			return nil
		})

		assert.NoError(t, err)
	})

	t.Run("Send Message to Non-existent Chat", func(t *testing.T) {
		// Define the expected request and response
		pact.
			AddInteraction().
			Given("User is authenticated but chat does not exist").
			UponReceiving("A WebSocket connection request").
			WithRequest(dsl.Request{
				Method: "GET",
				Path:   dsl.String("/ws"),
				Headers: dsl.MapMatcher{
					"Upgrade":               dsl.String("websocket"),
					"Connection":            dsl.String("Upgrade"),
					"Sec-WebSocket-Version": dsl.String("13"),
					"Authorization":         dsl.String("Bearer valid-token"),
				},
			}).
			WillRespondWith(dsl.Response{
				Status: 101,
				Headers: dsl.MapMatcher{
					"Upgrade":    dsl.String("websocket"),
					"Connection": dsl.String("Upgrade"),
				},
			})

		// Execute the test
		err := pact.Verify(func() error {
			// Create WebSocket connection
			wsURL := fmt.Sprintf("ws://localhost:%d/ws", pact.Server.Port)
			header := http.Header{}
			header.Add("Authorization", "Bearer valid-token")

			conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
			if err != nil {
				return err
			}
			defer conn.Close()

			// Send a message to non-existent chat
			message := map[string]interface{}{
				"type":    "message",
				"chat_id": "non-existent",
				"content": "Hello, World!",
			}
			err = conn.WriteJSON(message)
			if err != nil {
				return err
			}

			// Read response
			var response map[string]interface{}
			err = conn.ReadJSON(&response)
			if err != nil {
				return err
			}

			// Assert response
			assert.Equal(t, "error", response["type"])
			assert.Equal(t, "Chat not found", response["message"])
			return nil
		})

		assert.NoError(t, err)
	})
}
