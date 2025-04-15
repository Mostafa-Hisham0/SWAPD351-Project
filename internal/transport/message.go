package transport

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"rtcs/internal/service"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// SendMessageRequest represents the request body for sending a message
type SendMessageRequest struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

// MessageHandler handles message-related requests
type MessageHandler struct {
	messageService *service.MessageService
}

// NewMessageHandler creates a new message handler
func NewMessageHandler(messageService *service.MessageService) *MessageHandler {
	return &MessageHandler{
		messageService: messageService,
	}
}

// Send handles message sending
func (h *MessageHandler) Send(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received message send request at path: %s", r.URL.Path)
	log.Printf("Headers: %v", r.Header)

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	log.Printf("Decoded request: %+v", req)

	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		log.Printf("Error: user_id not found in context or wrong type")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	log.Printf("User ID from context: %s", userID.String())

	// Validate chat ID format but pass the string to service
	_, err := uuid.Parse(req.ChatID)
	if err != nil {
		log.Printf("Error parsing chat ID: %v", err)
		http.Error(w, "Invalid chat ID format", http.StatusBadRequest)
		return
	}

	message, err := h.messageService.SendMessage(r.Context(), req.ChatID, userID.String(), req.Text)
	if err != nil {
		log.Printf("Error sending message: %v", err)
		http.Error(w, "Failed to send message", http.StatusInternalServerError)
		return
	}
	log.Printf("Message sent successfully: %+v", message)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(message); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// GetChatHistory handles retrieving chat history
func (h *MessageHandler) GetChatHistory(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received get chat history request")

	vars := mux.Vars(r)
	chatID := vars["chatId"]
	if chatID == "" {
		log.Printf("Error: chatId is empty")
		http.Error(w, "Chat ID is required", http.StatusBadRequest)
		return
	}
	
	// Validate chat ID format
	_, err := uuid.Parse(chatID)
	if err != nil {
		log.Printf("Error parsing chat ID: %v", err)
		http.Error(w, "Invalid chat ID format", http.StatusBadRequest)
		return
	}
	
	log.Printf("Chat ID from URL: %s", chatID)

	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		log.Printf("Error: user_id not found in context or wrong type")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	log.Printf("User ID from context: %s", userID.String())

	limit := 50 // Default limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}
	log.Printf("Using limit: %d", limit)

	messages, err := h.messageService.GetChatHistory(r.Context(), chatID, limit)
	if err != nil {
		log.Printf("Error getting chat history: %v", err)
		http.Error(w, "Failed to get chat history", http.StatusInternalServerError)
		return
	}
	log.Printf("Retrieved %d messages", len(messages))

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(messages); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// DeleteMessage handles message deletion
func (h *MessageHandler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received delete message request")

	vars := mux.Vars(r)
	messageID := vars["messageId"]
	if messageID == "" {
		log.Printf("Error: messageId is empty")
		http.Error(w, "Message ID is required", http.StatusBadRequest)
		return
	}
	log.Printf("Message ID from URL: %s", messageID)

	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		log.Printf("Error: user_id not found in context or wrong type")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	log.Printf("User ID from context: %s", userID.String())

	if err := h.messageService.DeleteMessage(r.Context(), messageID, userID.String()); err != nil {
		log.Printf("Error deleting message: %v", err)
		http.Error(w, "Failed to delete message", http.StatusInternalServerError)
		return
	}
	log.Printf("Message deleted successfully")

	w.WriteHeader(http.StatusNoContent)
}
