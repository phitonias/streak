package api

import (
	"github.com/gin-gonic/gin"
	"github.com/phitonias/streak/internal/config"
	"github.com/phitonias/streak/internal/middleware"
	"github.com/phitonias/streak/internal/models"
	"github.com/phitonias/streak/internal/service"
	"github.com/phitonias/streak/internal/utils"
)

type AuthHandler struct {
	service *service.AuthService
	cfg     *config.Config
}

func NewAuthHandler(service *service.AuthService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{service: service, cfg: cfg}
}

func (h *AuthHandler) Register(c *gin.Context) {
	req, exists := c.Get("validatedRequest")
	if !exists {
		utils.ErrorResponse(c, 400, "Invalid request")
		return
	}

	registerReq := req.(middleware.RegisterRequest)

	input := service.RegisterInput{
		Username:        registerReq.Username,
		Email:           registerReq.Email,
		Password:        registerReq.Password,
		Role:            models.UserRole(registerReq.Role),
		DisplayName:     registerReq.DisplayName,
		SportCategories: registerReq.SportCategories,
	}

	result, err := h.service.Register(input)
	if err != nil {
		utils.ErrorResponse(c, 400, err.Error())
		return
	}

	utils.SuccessMessageResponse(c, 201, "User registered successfully", result)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req middleware.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, 400, "Invalid request body")
		return
	}

	result, err := h.service.Login(req.Username, req.Password)
	if err != nil {
		utils.ErrorResponse(c, 401, err.Error())
		return
	}

	utils.SuccessMessageResponse(c, 200, "Logged in successfully", result)
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, 400, "Refresh token is required")
		return
	}

	accessToken, err := h.service.RefreshToken(req.RefreshToken)
	if err != nil {
		utils.ErrorResponse(c, 401, err.Error())
		return
	}

	utils.SuccessResponse(c, 200, gin.H{"accessToken": accessToken})
}

func (h *AuthHandler) GetMe(c *gin.Context) {
	claims, ok := middleware.GetUserClaims(c)
	if !ok {
		utils.ErrorResponse(c, 401, "Not authenticated")
		return
	}

	user, err := h.service.GetMe(claims.UserID)
	if err != nil {
		utils.ErrorResponse(c, 404, "User not found")
		return
	}

	utils.SuccessResponse(c, 200, user)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	utils.SuccessMessageResponse(c, 200, "Logged out successfully", nil)
}
