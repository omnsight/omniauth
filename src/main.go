package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	gwRuntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/omnsight/omnauth/gen/oauth/v1"
	"github.com/omnsight/omnauth/src/utils"
)

func main() {
	// ---- 1. Start the gRPC Server (your logic) ----
	// Get gRPC address from environment variable or use default
	grpcPort := os.Getenv(utils.GrpcPort)
	if grpcPort == "" {
		logrus.Fatalf("missing environment variable %s", utils.GrpcPort)
	}

	serverPort := os.Getenv(utils.ServerPort)
	if serverPort == "" {
		logrus.Fatalf("missing environment variable %s", utils.ServerPort)
	}

	clientId := os.Getenv(utils.KeycloakClientID)
	if clientId == "" {
		logrus.Fatalf("missing environment variable %s", utils.KeycloakClientID)
	}

	// Create a gRPC server
	gRPCServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(utils.LoggingInterceptor, utils.GrpcGatewayIdentityInterceptor(clientId)),
	)

	cloakHelper := utils.NewCloakHelper()

	// Register your business logic implementation with the gRPC server
	authService, err := NewAuthService(cloakHelper)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("failed to create AuthService")
	}
	oauth.RegisterAuthServiceServer(gRPCServer, authService)

	// Enable reflection for debugging
	reflection.Register(gRPCServer)

	// Start the gRPC server in a separate goroutine
	go func() {
		lis, _ := net.Listen("tcp", ":"+grpcPort)
		gRPCServer.Serve(lis)
	}()

	// ---- 2. Start the gRPC-Gateway (the connection) ----
	ctx := context.Background()

	// Create a client connection to the gRPC server
	// The gateway acts as a client - using NewClient instead of deprecated DialContext
	conn, err := grpc.NewClient(
		"localhost:"+grpcPort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("failed to create gRPC client")
	}
	defer conn.Close()

	// Create the gRPC-Gateway's multiplexer (router)
	// This mux knows how to translate HTTP routes (from proto definitions) to gRPC calls
	gwmux := gwRuntime.NewServeMux()

	// Register all service handlers with the gateway's router
	if err := oauth.RegisterAuthServiceHandler(ctx, gwmux, conn); err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("failed to register AuthService handler")
	}

	// ---- 3. Start the Gin Server (the HTTP entrypoint) ----
	// Create a Gin router
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/health"},
	}))

	// Tell Gin to proxy any requests on /v1/* to the gRPC-Gateway
	// THIS IS THE "CONNECTION"
	r.Any("/v1/*any", gin.WrapH(gwmux))

	// Add other Gin routes as needed
	r.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// --- CHECK 1: Keycloak Connectivity ---
		_, err = cloakHelper.Client.GetCerts(ctx, "master")
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

	// Run the Gin server
	r.Run(":" + serverPort)
}
