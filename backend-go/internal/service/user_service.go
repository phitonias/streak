package service

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/phitonias/streak/internal/database"
	"github.com/phitonias/streak/internal/models"
	"gorm.io/gorm"
)

type UserService struct{}

func NewUserService() *UserService {
	return &UserService{}
}

type UpdateProfileInput struct {
	DisplayName     *string
	Bio             *string
	AvatarURL       *string
	BannerURL       *string
	SportCategories []string
}

func (s *UserService) GetUserByID(userID uuid.UUID) (interface{}, error) {
	var user models.User
	result := database.DB.Preload("AthleteProfile").
		Where("id = ? AND is_banned = false", userID).
		First(&user)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, result.Error
	}

	return sanitizeUser(&user), nil
}

func (s *UserService) GetUserByUsername(username string) (interface{}, error) {
	var user models.User
	result := database.DB.Preload("AthleteProfile").
		Where("username = ? AND is_banned = false", username).
		First(&user)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, result.Error
	}

	return sanitizeUser(&user), nil
}

func (s *UserService) UpdateProfile(userID uuid.UUID, input UpdateProfileInput) (interface{}, error) {
	var user models.User
	if err := database.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, errors.New("user not found")
	}

	// Update user fields
	updates := make(map[string]interface{})
	if input.DisplayName != nil {
		updates["display_name"] = *input.DisplayName
	}
	if input.Bio != nil {
		updates["bio"] = *input.Bio
	}
	if input.AvatarURL != nil {
		updates["avatar_url"] = *input.AvatarURL
	}
	if input.BannerURL != nil {
		updates["banner_url"] = *input.BannerURL
	}

	if len(updates) > 0 {
		if err := database.DB.Model(&user).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	// Update athlete profile if sport categories provided
	if len(input.SportCategories) > 0 && user.Role == models.RoleAthlete {
		if err := database.DB.Model(&models.AthleteProfile{}).
			Where("user_id = ?", userID).
			Update("sport_categories", input.SportCategories).Error; err != nil {
			return nil, err
		}
	}

	return s.GetUserByID(userID)
}

func (s *UserService) FollowUser(followerID, athleteID uuid.UUID) error {
	// Check if athlete exists and is an athlete
	var athlete models.User
	if err := database.DB.Where("id = ?", athleteID).First(&athlete).Error; err != nil {
		return errors.New("athlete not found")
	}

	if athlete.Role != models.RoleAthlete {
		return errors.New("can only follow athletes")
	}

	if followerID == athleteID {
		return errors.New("cannot follow yourself")
	}

	// Check if already following
	var existingFollow models.Follow
	result := database.DB.Where("follower_id = ? AND athlete_id = ?", followerID, athleteID).First(&existingFollow)
	if result.Error == nil {
		return errors.New("already following this athlete")
	}

	// Create follow
	follow := models.Follow{
		FollowerID: followerID,
		AthleteID:  athleteID,
	}

	if err := database.DB.Create(&follow).Error; err != nil {
		return err
	}

	// Increment follower count (trigger handles this in DB, but we can also do it here)
	database.DB.Model(&models.AthleteProfile{}).
		Where("user_id = ?", athleteID).
		Update("follower_count", gorm.Expr("follower_count + 1"))

	return nil
}

func (s *UserService) UnfollowUser(followerID, athleteID uuid.UUID) error {
	result := database.DB.Where("follower_id = ? AND athlete_id = ?", followerID, athleteID).
		Delete(&models.Follow{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("not following this athlete")
	}

	// Decrement follower count
	database.DB.Model(&models.AthleteProfile{}).
		Where("user_id = ?", athleteID).
		Update("follower_count", gorm.Expr("follower_count - 1"))

	return nil
}

func (s *UserService) GetFollowers(athleteID uuid.UUID, page, limit int) ([]interface{}, *Pagination, error) {
	offset := (page - 1) * limit

	var follows []models.Follow
	if err := database.DB.Preload("Follower").
		Where("athlete_id = ?", athleteID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&follows).Error; err != nil {
		return nil, nil, err
	}

	// Get total count
	var total int64
	database.DB.Model(&models.Follow{}).Where("athlete_id = ?", athleteID).Count(&total)

	// Format response
	followers := make([]interface{}, len(follows))
	for i, follow := range follows {
		followers[i] = map[string]interface{}{
			"id":           follow.Follower.ID,
			"username":     follow.Follower.Username,
			"display_name": follow.Follower.DisplayName,
			"avatar_url":   follow.Follower.AvatarURL,
			"followed_at":  follow.CreatedAt,
		}
	}

	pagination := &Pagination{
		Page:       page,
		Limit:      limit,
		Total:      int(total),
		TotalPages: (int(total) + limit - 1) / limit,
	}

	return followers, pagination, nil
}

func (s *UserService) GetFollowing(followerID uuid.UUID, page, limit int) ([]interface{}, *Pagination, error) {
	offset := (page - 1) * limit

	var follows []models.Follow
	if err := database.DB.Preload("Athlete").Preload("Athlete.AthleteProfile").
		Where("follower_id = ?", followerID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&follows).Error; err != nil {
		return nil, nil, err
	}

	// Get total count
	var total int64
	database.DB.Model(&models.Follow{}).Where("follower_id = ?", followerID).Count(&total)

	// Format response
	following := make([]interface{}, len(follows))
	for i, follow := range follows {
		isLive := false
		if follow.Athlete.AthleteProfile != nil {
			isLive = follow.Athlete.AthleteProfile.IsLive
		}

		following[i] = map[string]interface{}{
			"id":           follow.Athlete.ID,
			"username":     follow.Athlete.Username,
			"display_name": follow.Athlete.DisplayName,
			"avatar_url":   follow.Athlete.AvatarURL,
			"is_live":      isLive,
			"followed_at":  follow.CreatedAt,
		}
	}

	pagination := &Pagination{
		Page:       page,
		Limit:      limit,
		Total:      int(total),
		TotalPages: (int(total) + limit - 1) / limit,
	}

	return following, pagination, nil
}

func (s *UserService) IsFollowing(followerID, athleteID uuid.UUID) (bool, error) {
	var count int64
	err := database.DB.Model(&models.Follow{}).
		Where("follower_id = ? AND athlete_id = ?", followerID, athleteID).
		Count(&count).Error

	return count > 0, err
}

type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}
