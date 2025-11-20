package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/omnsight/omniscent-library/src/clients"
	"github.com/omnsight/omniscent-library/src/middleware"
)

func main() {
	cloakHelper := clients.NewCloakHelper()

	r := gin.Default()

	r.Use(middleware.AuthMiddleware(cloakHelper.ClientID))

	r.GET("/users/:id", func(c *gin.Context) {
		callerID := c.GetString("userID")
		callerRoles := c.GetStringSlice("userRoles")

		targetUserID := c.Param("id")

		log.Printf("[Audit] Caller %s with roles %v is requesting public data of user %s", callerID, callerRoles, targetUserID)

		publicData, err := cloakHelper.GetPublicUserData(c.Request.Context(), targetUserID)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			} else {
				log.Printf("Error fetching user: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error fetching user data"})
			}
			return
		}

		c.JSON(http.StatusOK, publicData)
	})

	log.Println("Server running on :8081")
	if err := r.Run(":8081"); err != nil {
		log.Fatal(err)
	}
}
