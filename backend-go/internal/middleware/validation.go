package middleware

import (
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/phitonias/streak/internal/utils"
)

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,50}$`)
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	passwordRegex = regexp.MustCompile(`^(?=.*[a-z])(?=.*[A-Z])(?=.*\d).{8,}$`)
)

type RegisterRequest struct {
	Username        string   `json:"username" binding:"required"`
	Email           string   `json:"email" binding:"required"`
	Password        string   `json:"password" binding:"required"`
	Role            string   `json:"role" binding:"required"`
	DisplayName     string   `json:"displayName"`
	SportCategories []string `json:"sportCategories"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type CreateStreamRequest struct {
	Title         string   `json:"title" binding:"required,min=1,max=200"`
	Description   string   `json:"description"`
	SportCategory string   `json:"sportCategory" binding:"required"`
	Tags          []string `json:"tags"`
}

type UpdateStreamRequest struct {
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	SportCategory string   `json:"sportCategory"`
	Tags          []string `json:"tags"`
	ThumbnailURL  string   `json:"thumbnailUrl"`
}

type UpdateProfileRequest struct {
	DisplayName     string   `json:"displayName"`
	Bio             string   `json:"bio"`
	AvatarURL       string   `json:"avatarUrl"`
	BannerURL       string   `json:"bannerUrl"`
	SportCategories []string `json:"sportCategories"`
}

func ValidateRegister(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, 400, "Invalid request body")
		c.Abort()
		return
	}

	// Validate username
	if !usernameRegex.MatchString(req.Username) {
		utils.ErrorResponse(c, 400, "Username must be 3-50 characters and contain only letters, numbers, and underscores")
		c.Abort()
		return
	}

	// Validate email
	if !emailRegex.MatchString(req.Email) {
		utils.ErrorResponse(c, 400, "Invalid email address")
		c.Abort()
		return
	}

	// Validate password
	if !passwordRegex.MatchString(req.Password) {
		utils.ErrorResponse(c, 400, "Password must be at least 8 characters with uppercase, lowercase, and number")
		c.Abort()
		return
	}

	// Validate role
	if req.Role != "athlete" && req.Role != "follower" {
		utils.ErrorResponse(c, 400, "Role must be athlete or follower")
		c.Abort()
		return
	}

	c.Set("validatedRequest", req)
	c.Next()
}
