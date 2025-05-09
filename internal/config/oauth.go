package config

import (
	"bufio"
	"log"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type OAuthConfig struct {
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
}

func loadEnvFile() {
	file, err := os.Open(".env")
	if err != nil {
		log.Printf("Warning: .env file not found: %v", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			os.Setenv(key, value)
		}
	}
}

func LoadOAuthConfig() (*OAuthConfig, error) {
	// Try to load from .env file first
	loadEnvFile()

	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")

	log.Printf("Loading OAuth config - ClientID: %s, RedirectURL: %s", clientID, redirectURL)

	if clientID == "" || clientSecret == "" || redirectURL == "" {
		log.Printf("Warning: Missing OAuth configuration. Please check your environment variables.")
	}

	config := &OAuthConfig{
		GoogleClientID:     clientID,
		GoogleClientSecret: clientSecret,
		GoogleRedirectURL:  redirectURL,
	}
	return config, nil
}

func GetGoogleOAuthConfig(config *OAuthConfig) *oauth2.Config {
	log.Printf("Creating OAuth config with - ClientID: %s, RedirectURL: %s", config.GoogleClientID, config.GoogleRedirectURL)

	return &oauth2.Config{
		ClientID:     config.GoogleClientID,
		ClientSecret: config.GoogleClientSecret,
		RedirectURL:  config.GoogleRedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}
