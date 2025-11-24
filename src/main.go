package main

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/omnsight/omniscent-library/src/clients"
	"github.com/omnsight/omniscent-library/src/constants"
	"github.com/omnsight/omniscent-library/src/middleware"
)

// getUserHandler handles the GET /users/:id endpoint
func getUserHandler(cloakHelper *clients.CloakHelper) gin.HandlerFunc {
	return func(c *gin.Context) {
		callerID := c.GetString("userID")
		callerRoles := c.GetStringSlice("userRoles")

		targetUserID := c.Param("id")

		logrus.Infof("[Audit] Caller %s with roles %v is requesting public data of user %s", callerID, callerRoles, targetUserID)

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
	cloakHelper := clients.NewCloakHelper()

	r := gin.Default()

	r.Use(middleware.AuthMiddleware(cloakHelper.ClientID))

	r.GET("/users/:id", getUserHandler(cloakHelper))

	serverPort := os.Getenv(constants.ServerPort)
	if serverPort == "" {
		logrus.Fatalf("missing environment variable %s", constants.ServerPort)
	}

	logrus.Infof("Server running on: %s", serverPort)
	if err := r.Run(":" + serverPort); err != nil {
		logrus.WithError(err).Fatal("Error starting server")
	}
}
