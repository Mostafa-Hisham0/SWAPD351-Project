package http

import (
	"context"
	"net/http"

	"golang.org/x/oauth2"
	googleoauth2 "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"

	"rtcs/internal/config"
	"rtcs/internal/service"
)

type OAuthHandler struct {
	oauthConfig *oauth2.Config
	authService *service.AuthService
}

func NewOAuthHandler(cfg *config.OAuthConfig, authService *service.AuthService) *OAuthHandler {
	return &OAuthHandler{
		oauthConfig: config.GetGoogleOAuthConfig(cfg),
		authService: authService,
	}
}

func (h *OAuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	url := h.oauthConfig.AuthCodeURL("state")
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *OAuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	token, err := h.oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}

	client := h.oauthConfig.Client(context.Background(), token)
	service, err := googleoauth2.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		http.Error(w, "Failed to create OAuth2 service", http.StatusInternalServerError)
		return
	}

	userInfo, err := service.Userinfo.Get().Do()
	if err != nil {
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}

	// Create or get user from database
	user, err := h.authService.GetOrCreateGoogleUser(r.Context(), userInfo.Email, userInfo.Name, userInfo.Picture)
	if err != nil {
		http.Error(w, "Failed to create/get user", http.StatusInternalServerError)
		return
	}

	// Generate JWT token
	jwtToken, err := h.authService.GenerateToken(user.ID.String())
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Create HTML response that sets the token and redirects
	html := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>Authentication Successful</title>
		<script>
			// Store the token
			localStorage.setItem('token', '` + jwtToken + `');
			// Redirect to chat page
			window.location.href = '/test_websocket.html';
		</script>
	</head>
	<body>
		<p>Authentication successful! Redirecting...</p>
	</body>
	</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
