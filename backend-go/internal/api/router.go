package api

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/phitonias/streak/internal/config"
	"github.com/phitonias/streak/internal/middleware"
	"github.com/phitonias/streak/internal/models"
	"github.com/phitonias/streak/internal/service"
)

func SetupRouter(cfg *config.Config) *gin.Engine {
	// Set Gin mode
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Middleware
	router.Use(middleware.RecoveryMiddleware())
	router.Use(middleware.ErrorHandler())

	// CORS
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = cfg.CORS.AllowOrigins
	corsConfig.AllowCredentials = true
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	router.Use(cors.New(corsConfig))

	// Services
	authService := service.NewAuthService(cfg)
	userService := service.NewUserService()
	streamService := service.NewStreamService()
	chatService := service.NewChatService()

	// Handlers
	authHandler := NewAuthHandler(authService, cfg)
	userHandler := NewUserHandler(userService)
	streamHandler := NewStreamHandler(streamService)

	// WebSocket Hub
	hub := NewHub(chatService, streamService)
	go hub.Run()

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "healthy",
			"timestamp": gin.H{},
		})
	})

	// Root
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"name":    "Streak API (Go)",
			"version": "1.0.0",
			"status":  "running",
		})
	})

	// API routes
	api := router.Group("/api")
	{
		// Auth routes
		auth := api.Group("/auth")
		{
			auth.POST("/register", middleware.ValidateRegister, authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.GET("/me", middleware.AuthMiddleware(cfg), authHandler.GetMe)
			auth.POST("/logout", middleware.AuthMiddleware(cfg), authHandler.Logout)
		}

		// User routes
		users := api.Group("/users")
		{
			users.GET("/:id", userHandler.GetUserByID)
			users.GET("/username/:username", userHandler.GetUserByUsername)
			users.PUT("/profile", middleware.AuthMiddleware(cfg), userHandler.UpdateProfile)
			users.POST("/:id/follow", middleware.AuthMiddleware(cfg), userHandler.FollowUser)
			users.DELETE("/:id/follow", middleware.AuthMiddleware(cfg), userHandler.UnfollowUser)
			users.GET("/:id/followers", userHandler.GetFollowers)
			users.GET("/:id/following", userHandler.GetFollowing)
			users.GET("/:id/is-following", middleware.AuthMiddleware(cfg), userHandler.CheckFollowing)
		}

		// Stream routes
		streams := api.Group("/streams")
		{
			// Public routes
			streams.GET("/live", streamHandler.GetLiveStreams)
			streams.GET("/:id", streamHandler.GetStream)
			streams.GET("/athlete/:athleteId", streamHandler.GetAthleteStreams)

			// Authenticated routes
			streams.GET("/following/live", middleware.AuthMiddleware(cfg), streamHandler.GetFollowingStreams)

			// Athlete-only routes
			streams.POST("/",
				middleware.AuthMiddleware(cfg),
				middleware.RequireRole(models.RoleAthlete),
				streamHandler.CreateStream)

			streams.PUT("/:id",
				middleware.AuthMiddleware(cfg),
				middleware.RequireRole(models.RoleAthlete),
				streamHandler.UpdateStream)

			streams.POST("/:id/start",
				middleware.AuthMiddleware(cfg),
				middleware.RequireRole(models.RoleAthlete),
				streamHandler.StartStream)

			streams.POST("/:id/stop",
				middleware.AuthMiddleware(cfg),
				middleware.RequireRole(models.RoleAthlete),
				streamHandler.StopStream)
		}

		// WebSocket route
		api.GET("/ws", HandleWebSocket(cfg, hub))
	}

	return router
}
