package main

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/omnsight/omniscent-library/src/clients"
	"github.com/omnsight/omniscent-library/src/constants"
	"github.com/omnsight/omniscent-library/src/middleware"
)

// getUserHandler handles the GET /users/:id endpoint
func getUserHandler(cloakHelper *clients.CloakHelper) gin.HandlerFunc {
	return func(c *gin.Context) {
		callerID := c.GetString(middleware.UserIDKey)
		callerRoles := c.GetStringSlice(middleware.UserRolesKey)

		targetUserID := c.Param("id")

		logrus.Infof("[%s, %v] requests to get public data of user %s", callerID, callerRoles, targetUserID)

		publicData, err := cloakHelper.GetPublicUserData(c.Request.Context(), targetUserID)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			} else {
				logrus.WithError(err).Error("Error fetching user")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error fetching user data"})
			}
			return
		}

		c.JSON(http.StatusOK, publicData)
	}
}

func main() {
	serverPort := os.Getenv(constants.ServerPort)
	if serverPort == "" {
		logrus.Fatalf("missing environment variable %s", constants.ServerPort)
	}

	cloakHelper := clients.NewCloakHelper()
	r := gin.Default()

	// Add other Gin routes as needed
	r.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// --- CHECK 1: Keycloak Connectivity ---
		_, err := cloakHelper.Client.GetCerts(ctx, "omni")
		if err != nil {
			logrus.WithError(err).Error("Keycloak client is unreachable")
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"reason": "identity_provider_unreachable",
			})
			return
		}

		// --- SUCCESS ---
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"services": gin.H{
				"database":     "connected",
				"query_engine": "operational",
			},
		})
	})

	api := r.Group("/")
	api.Use(middleware.AuthMiddleware(cloakHelper.ClientID))
	api.GET("/users/:id", getUserHandler(cloakHelper))

	logrus.Infof("Server running on: %s", serverPort)
	if err := r.Run(":" + serverPort); err != nil {
		logrus.WithError(err).Fatal("Error starting server")
	}
}
