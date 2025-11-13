package service

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/phitonias/streak/internal/config"
	"github.com/phitonias/streak/internal/database"
	"github.com/phitonias/streak/internal/models"
	"github.com/phitonias/streak/internal/utils"
	"gorm.io/gorm"
)

type AuthService struct {
	cfg *config.Config
}

func NewAuthService(cfg *config.Config) *AuthService {
	return &AuthService{cfg: cfg}
}

type RegisterInput struct {
	Username        string
	Email           string
	Password        string
	Role            models.UserRole
	DisplayName     string
	SportCategories []string
}

type AuthResponse struct {
	AccessToken  string      `json:"accessToken"`
	RefreshToken string      `json:"refreshToken"`
	User         interface{} `json:"user"`
}

func (s *AuthService) Register(input RegisterInput) (*AuthResponse, error) {
	// Check if user already exists
	var existingUser models.User
	result := database.DB.Where("username = ? OR email = ?", input.Username, input.Email).First(&existingUser)
	if result.Error == nil {
		return nil, errors.New("username or email already exists")
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("database error: %w", result.Error)
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Set display name if not provided
	displayName := input.DisplayName
	if displayName == "" {
		displayName = input.Username
	}

	// Create user
	user := models.User{
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: hashedPassword,
		Role:         input.Role,
		DisplayName:  &displayName,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// If athlete, create athlete profile
	if input.Role == models.RoleAthlete {
		streamKey := generateStreamKey()
		rtmpURL := fmt.Sprintf("%s/%s", s.cfg.RTMP.ServerURL, streamKey)

		athleteProfile := models.AthleteProfile{
			UserID:          user.ID,
			SportCategories: input.SportCategories,
			StreamKey:       streamKey,
			RTMPURL:         &rtmpURL,
		}

		if err := database.DB.Create(&athleteProfile).Error; err != nil {
			return nil, fmt.Errorf("failed to create athlete profile: %w", err)
		}

		user.AthleteProfile = &athleteProfile
	}

	// Generate tokens
	tokens, err := utils.GenerateTokenPair(
		user.ID,
		user.Username,
		user.Role,
		s.cfg.JWT.Secret,
		s.cfg.JWT.RefreshSecret,
		s.cfg.JWT.Expiry,
		s.cfg.JWT.RefreshExpiry,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return &AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         sanitizeUser(&user),
	}, nil
}

func (s *AuthService) Login(username, password string) (*AuthResponse, error) {
	// Find user
	var user models.User
	result := database.DB.Preload("AthleteProfile").Where("username = ?", username).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid username or password")
		}
		return nil, fmt.Errorf("database error: %w", result.Error)
	}

	// Check if banned
	if user.IsBanned {
		return nil, errors.New("account has been banned")
	}

	// Verify password
	if !utils.CheckPassword(password, user.PasswordHash) {
		return nil, errors.New("invalid username or password")
	}

	// Generate tokens
	tokens, err := utils.GenerateTokenPair(
		user.ID,
		user.Username,
		user.Role,
		s.cfg.JWT.Secret,
		s.cfg.JWT.RefreshSecret,
		s.cfg.JWT.Expiry,
		s.cfg.JWT.RefreshExpiry,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return &AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         sanitizeUser(&user),
	}, nil
}

func (s *AuthService) RefreshToken(refreshToken string) (string, error) {
	// Validate refresh token
	claims, err := utils.ValidateToken(refreshToken, s.cfg.JWT.RefreshSecret)
	if err != nil {
		return "", errors.New("invalid refresh token")
	}

	// Verify user still exists and is not banned
	var user models.User
	if err := database.DB.Where("id = ? AND is_banned = false", claims.UserID).First(&user).Error; err != nil {
		return "", errors.New("invalid refresh token")
	}

	// Generate new access token
	tokens, err := utils.GenerateTokenPair(
		user.ID,
		user.Username,
		user.Role,
		s.cfg.JWT.Secret,
		s.cfg.JWT.RefreshSecret,
		s.cfg.JWT.Expiry,
		s.cfg.JWT.RefreshExpiry,
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return tokens.AccessToken, nil
}

func (s *AuthService) GetMe(userID uuid.UUID) (interface{}, error) {
	var user models.User
	result := database.DB.Preload("AthleteProfile").Where("id = ?", userID).First(&user)
	if result.Error != nil {
		return nil, fmt.Errorf("user not found: %w", result.Error)
	}

	return sanitizeUser(&user), nil
}

// Helper functions
func generateStreamKey() string {
	return fmt.Sprintf("%s_%d", uuid.New().String()[:8], uuid.New().ID())
}

func sanitizeUser(user *models.User) map[string]interface{} {
	result := map[string]interface{}{
		"id":           user.ID,
		"username":     user.Username,
		"email":        user.Email,
		"role":         user.Role,
		"display_name": user.DisplayName,
		"bio":          user.Bio,
		"avatar_url":   user.AvatarURL,
		"banner_url":   user.BannerURL,
		"created_at":   user.CreatedAt,
		"is_verified":  user.IsVerified,
	}

	if user.AthleteProfile != nil {
		result["sport_categories"] = user.AthleteProfile.SportCategories
		result["stream_key"] = user.AthleteProfile.StreamKey
		result["rtmp_url"] = user.AthleteProfile.RTMPURL
		result["follower_count"] = user.AthleteProfile.FollowerCount
		result["total_view_count"] = user.AthleteProfile.TotalViewCount
		result["is_live"] = user.AthleteProfile.IsLive
	}

	return result
}
