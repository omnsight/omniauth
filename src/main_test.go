package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/Nerzal/gocloak/v13"
	"github.com/omnsight/omniscent-library/src/clients"
)

// TestGetUser tests the GET /users/:id endpoint
func TestGetUser(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Check if we're running in docker compose environment
	keycloakURL := os.Getenv("KEYCLOAK_URL")
	if keycloakURL == "" {
		t.Skip("Skipping integration test: KEYCLOAK_URL not set")
	}

	// Create a real CloakHelper
	cloakHelper := clients.NewCloakHelper()

	// Login to get access token
	jwt, err := cloakHelper.Client.LoginClient(context.Background(), cloakHelper.ClientID, cloakHelper.ClientSecret, cloakHelper.Realm)
	if err != nil {
		t.Fatalf("Failed to login client: %v", err)
	}

	// Create a test user for testing
	testUserName := "test123"
	user := gocloak.User{
		Username:      &testUserName,
		FirstName:     gocloak.StringP("Test"),
		LastName:      gocloak.StringP("User"),
		Email:         gocloak.StringP("test@example.com"),
		EmailVerified: gocloak.BoolP(true),
		Enabled:       gocloak.BoolP(true),
	}

	testUserId, err := cloakHelper.Client.CreateUser(context.Background(), jwt.AccessToken, cloakHelper.Realm, user)
	if err != nil {
		t.Logf("Warning: Could not create test user: %v", err)
	}

	t.Run("successful user retrieval", func(t *testing.T) {
		// Create a test server
		router := gin.New()

		// Add a simple middleware to simulate auth for testing
		router.Use(func(c *gin.Context) {
			c.Set("userID", "caller123")
			c.Set("userRoles", []string{"admin"})
			c.Next()
		})

		// Register the handler
		router.GET("/users/:id", getUserHandler(cloakHelper))

		// Create a request
		req, _ := http.NewRequest("GET", "/users/"+testUserId, nil)
		w := httptest.NewRecorder()

		// Perform the request
		router.ServeHTTP(w, req)

		assert.True(t, w.Code == http.StatusOK)
	})

	t.Run("user not found", func(t *testing.T) {
		// Create a test server
		router := gin.New()

		// Add a simple middleware to simulate auth for testing
		router.Use(func(c *gin.Context) {
			c.Set("userID", "caller123")
			c.Set("userRoles", []string{"admin"})
			c.Next()
		})

		// Register the handler
		router.GET("/users/:id", getUserHandler(cloakHelper))

		// Create a request for a non-existent user
		req, _ := http.NewRequest("GET", "/users/nonexistent", nil)
		w := httptest.NewRecorder()

		// Perform the request
		router.ServeHTTP(w, req)

		assert.True(t, w.Code == http.StatusNotFound)
	})
}
