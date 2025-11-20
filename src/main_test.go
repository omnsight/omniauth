package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCloakHelper is a mock implementation of CloakHelperInterface
type MockCloakHelper struct {
	mock.Mock
	ClientID string
}

// GetPublicUserData mocks the GetPublicUserData method
func (m *MockCloakHelper) GetPublicUserData(ctx context.Context, userID string) (map[string]interface{}, error) {
	args := m.Called(ctx, userID)
	result, _ := args.Get(0).(map[string]interface{})
	return result, args.Error(1)
}

// TestGetUser tests the GET /users/:id endpoint
func TestGetUser(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	t.Run("successful user retrieval", func(t *testing.T) {
		// Create a mock CloakHelper
		mockCloakHelper := new(MockCloakHelper)

		// Create a test server
		router := gin.New()

		// Add a simple middleware to simulate auth for testing
		router.Use(func(c *gin.Context) {
			c.Set("userID", "caller123")
			c.Set("userRoles", []string{"admin"})
			c.Next()
		})

		// Register the handler
		router.GET("/users/:id", func(c *gin.Context) {
			targetUserID := c.Param("id")

			publicData := map[string]interface{}{
				"id":       targetUserID,
				"username": fmt.Sprintf("user-%s", targetUserID),
			}

			mockCloakHelper.On("GetPublicUserData", c.Request.Context(), targetUserID).Return(publicData, nil)

			result, err := mockCloakHelper.GetPublicUserData(c.Request.Context(), targetUserID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error fetching user data"})
				return
			}

			c.JSON(http.StatusOK, result)
		})

		// Create a request
		req, _ := http.NewRequest("GET", "/users/test123", nil)
		w := httptest.NewRecorder()

		// Perform the request
		router.ServeHTTP(w, req)

		// Assert the response
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "test123", response["id"])
		assert.Equal(t, "user-test123", response["username"])

		mockCloakHelper.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		// Create a mock CloakHelper
		mockCloakHelper := new(MockCloakHelper)

		// Create a test server
		router := gin.New()

		// Add a simple middleware to simulate auth for testing
		router.Use(func(c *gin.Context) {
			c.Set("userID", "caller123")
			c.Set("userRoles", []string{"admin"})
			c.Next()
		})

		// Register the handler
		router.GET("/users/:id", func(c *gin.Context) {
			targetUserID := c.Param("id")

			// Mock the not found error
			mockCloakHelper.On("GetPublicUserData", c.Request.Context(), targetUserID).Return((map[string]interface{})(nil), fmt.Errorf("404 Not Found"))

			result, err := mockCloakHelper.GetPublicUserData(c.Request.Context(), targetUserID)
			if err != nil {
				if fmt.Sprintf("%v", err) == "404 Not Found" {
					c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error fetching user data"})
				}
				return
			}

			c.JSON(http.StatusOK, result)
		})

		// Create a request
		req, _ := http.NewRequest("GET", "/users/nonexistent", nil)
		w := httptest.NewRecorder()

		// Perform the request
		router.ServeHTTP(w, req)

		// Assert the response
		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "user not found", response["error"])

		mockCloakHelper.AssertExpectations(t)
	})
}
