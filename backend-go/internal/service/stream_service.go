package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/phitonias/streak/internal/database"
	"github.com/phitonias/streak/internal/models"
	"gorm.io/gorm"
)

type StreamService struct{}

func NewStreamService() *StreamService {
	return &StreamService{}
}

type CreateStreamInput struct {
	AthleteID     uuid.UUID
	Title         string
	Description   string
	SportCategory string
	Tags          []string
}

type UpdateStreamInput struct {
	Title         *string
	Description   *string
	SportCategory *string
	Tags          []string
	ThumbnailURL  *string
}

func (s *StreamService) CreateStream(input CreateStreamInput) (*models.Stream, error) {
	// Check if athlete already has an active stream
	var activeStream models.Stream
	result := database.DB.Where("athlete_id = ? AND is_live = true", input.AthleteID).First(&activeStream)
	if result.Error == nil {
		return nil, errors.New("you already have an active stream")
	}

	// Create stream
	stream := models.Stream{
		AthleteID:     input.AthleteID,
		Title:         input.Title,
		Description:   &input.Description,
		SportCategory: input.SportCategory,
		Tags:          input.Tags,
	}

	if err := database.DB.Create(&stream).Error; err != nil {
		return nil, err
	}

	return &stream, nil
}

func (s *StreamService) GetStreamByID(streamID uuid.UUID) (interface{}, error) {
	var stream models.Stream
	result := database.DB.Preload("Athlete").Preload("Athlete.AthleteProfile").
		Where("id = ?", streamID).
		First(&stream)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("stream not found")
		}
		return nil, result.Error
	}

	// Get viewer count from Redis if live
	if stream.IsLive {
		viewerCount, _ := s.GetViewerCount(streamID)
		stream.CurrentViewerCount = viewerCount
	}

	return formatStream(&stream), nil
}

func (s *StreamService) UpdateStream(streamID, athleteID uuid.UUID, input UpdateStreamInput) (interface{}, error) {
	// Verify ownership
	var stream models.Stream
	if err := database.DB.Where("id = ? AND athlete_id = ?", streamID, athleteID).First(&stream).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("stream not found or not authorized")
		}
		return nil, err
	}

	// Update fields
	updates := make(map[string]interface{})
	if input.Title != nil {
		updates["title"] = *input.Title
	}
	if input.Description != nil {
		updates["description"] = *input.Description
	}
	if input.SportCategory != nil {
		updates["sport_category"] = *input.SportCategory
	}
	if input.Tags != nil {
		updates["tags"] = input.Tags
	}
	if input.ThumbnailURL != nil {
		updates["thumbnail_url"] = *input.ThumbnailURL
	}

	if len(updates) > 0 {
		if err := database.DB.Model(&stream).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	return s.GetStreamByID(streamID)
}

func (s *StreamService) StartStream(athleteID, streamID uuid.UUID) (interface{}, error) {
	var stream models.Stream
	if err := database.DB.Where("id = ? AND athlete_id = ?", streamID, athleteID).First(&stream).Error; err != nil {
		return nil, errors.New("stream not found or not authorized")
	}

	if stream.IsLive {
		return nil, errors.New("stream is already live")
	}

	now := time.Now()

	// Update stream status
	if err := database.DB.Model(&stream).Updates(map[string]interface{}{
		"is_live":    true,
		"started_at": now,
	}).Error; err != nil {
		return nil, err
	}

	// Update athlete profile
	database.DB.Model(&models.AthleteProfile{}).
		Where("user_id = ?", athleteID).
		Update("is_live", true)

	// Create stream session
	session := models.StreamSession{
		StreamID:  streamID,
		StartedAt: now,
	}
	database.DB.Create(&session)

	// Set stream status in Redis
	ctx := context.Background()
	database.RedisClient.Set(ctx, fmt.Sprintf("stream:%s:status", streamID), "live", 24*time.Hour)

	return s.GetStreamByID(streamID)
}

func (s *StreamService) StopStream(athleteID, streamID uuid.UUID) error {
	var stream models.Stream
	if err := database.DB.Where("id = ? AND athlete_id = ?", streamID, athleteID).First(&stream).Error; err != nil {
		return errors.New("stream not found or not authorized")
	}

	if !stream.IsLive {
		return errors.New("stream is not live")
	}

	// Get current viewer count for peak tracking
	viewerCount, _ := s.GetViewerCount(streamID)

	now := time.Now()

	// Update stream status
	peakViewers := stream.PeakViewerCount
	if viewerCount > peakViewers {
		peakViewers = viewerCount
	}

	if err := database.DB.Model(&stream).Updates(map[string]interface{}{
		"is_live":            false,
		"ended_at":           now,
		"peak_viewer_count":  peakViewers,
	}).Error; err != nil {
		return err
	}

	// Update athlete profile (set is_live to false only if no other streams are live)
	var liveCount int64
	database.DB.Model(&models.Stream{}).
		Where("athlete_id = ? AND is_live = true", athleteID).
		Count(&liveCount)

	if liveCount == 0 {
		database.DB.Model(&models.AthleteProfile{}).
			Where("user_id = ?", athleteID).
			Update("is_live", false)
	}

	// Close stream session
	database.DB.Model(&models.StreamSession{}).
		Where("stream_id = ? AND ended_at IS NULL", streamID).
		Updates(map[string]interface{}{
			"ended_at":       now,
			"peak_viewers":   peakViewers,
		})

	// Clear Redis data
	ctx := context.Background()
	database.RedisClient.Del(ctx, fmt.Sprintf("stream:%s:status", streamID))
	database.RedisClient.Del(ctx, fmt.Sprintf("stream:%s:viewers", streamID))

	return nil
}

func (s *StreamService) GetLiveStreams(page, limit int, category string) ([]interface{}, *Pagination, error) {
	offset := (page - 1) * limit

	query := database.DB.Preload("Athlete").Preload("Athlete.AthleteProfile").
		Where("is_live = true")

	if category != "" {
		query = query.Where("sport_category = ?", category)
	}

	var streams []models.Stream
	if err := query.Order("current_viewer_count DESC, started_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&streams).Error; err != nil {
		return nil, nil, err
	}

	// Get total count
	var total int64
	countQuery := database.DB.Model(&models.Stream{}).Where("is_live = true")
	if category != "" {
		countQuery = countQuery.Where("sport_category = ?", category)
	}
	countQuery.Count(&total)

	// Enhance with real-time viewer counts from Redis
	result := make([]interface{}, len(streams))
	for i, stream := range streams {
		viewerCount, _ := s.GetViewerCount(stream.ID)
		stream.CurrentViewerCount = viewerCount
		result[i] = formatStream(&stream)
	}

	pagination := &Pagination{
		Page:       page,
		Limit:      limit,
		Total:      int(total),
		TotalPages: (int(total) + limit - 1) / limit,
	}

	return result, pagination, nil
}

func (s *StreamService) GetStreamsByAthlete(athleteID uuid.UUID, page, limit int) ([]interface{}, *Pagination, error) {
	offset := (page - 1) * limit

	var streams []models.Stream
	if err := database.DB.Preload("Athlete").
		Where("athlete_id = ?", athleteID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&streams).Error; err != nil {
		return nil, nil, err
	}

	// Get total count
	var total int64
	database.DB.Model(&models.Stream{}).Where("athlete_id = ?", athleteID).Count(&total)

	result := make([]interface{}, len(streams))
	for i, stream := range streams {
		result[i] = formatStream(&stream)
	}

	pagination := &Pagination{
		Page:       page,
		Limit:      limit,
		Total:      int(total),
		TotalPages: (int(total) + limit - 1) / limit,
	}

	return result, pagination, nil
}

func (s *StreamService) GetFollowingStreams(userID uuid.UUID) ([]interface{}, error) {
	var streams []models.Stream
	if err := database.DB.Preload("Athlete").Preload("Athlete.AthleteProfile").
		Joins("JOIN follows ON follows.athlete_id = streams.athlete_id").
		Where("follows.follower_id = ? AND streams.is_live = true", userID).
		Order("streams.current_viewer_count DESC").
		Find(&streams).Error; err != nil {
		return nil, err
	}

	result := make([]interface{}, len(streams))
	for i, stream := range streams {
		viewerCount, _ := s.GetViewerCount(stream.ID)
		stream.CurrentViewerCount = viewerCount
		result[i] = formatStream(&stream)
	}

	return result, nil
}

func (s *StreamService) AddViewer(streamID uuid.UUID, userID uuid.UUID) (int, error) {
	ctx := context.Background()
	key := fmt.Sprintf("stream:%s:viewers", streamID)

	// Add viewer to set
	if err := database.RedisClient.SAdd(ctx, key, userID.String()).Err(); err != nil {
		return 0, err
	}

	// Get count
	count, err := database.RedisClient.SCard(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	// Update database periodically
	if count%10 == 0 {
		database.DB.Model(&models.Stream{}).
			Where("id = ?", streamID).
			Update("current_viewer_count", count)
	}

	return int(count), nil
}

func (s *StreamService) RemoveViewer(streamID uuid.UUID, userID uuid.UUID) (int, error) {
	ctx := context.Background()
	key := fmt.Sprintf("stream:%s:viewers", streamID)

	// Remove viewer from set
	if err := database.RedisClient.SRem(ctx, key, userID.String()).Err(); err != nil {
		return 0, err
	}

	// Get count
	count, err := database.RedisClient.SCard(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	return int(count), nil
}

func (s *StreamService) GetViewerCount(streamID uuid.UUID) (int, error) {
	ctx := context.Background()
	key := fmt.Sprintf("stream:%s:viewers", streamID)

	count, err := database.RedisClient.SCard(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	return int(count), nil
}

func formatStream(stream *models.Stream) map[string]interface{} {
	result := map[string]interface{}{
		"id":                   stream.ID,
		"title":                stream.Title,
		"description":          stream.Description,
		"sport_category":       stream.SportCategory,
		"tags":                 stream.Tags,
		"thumbnail_url":        stream.ThumbnailURL,
		"is_live":              stream.IsLive,
		"started_at":           stream.StartedAt,
		"current_viewer_count": stream.CurrentViewerCount,
		"created_at":           stream.CreatedAt,
	}

	if stream.Athlete.ID != uuid.Nil {
		result["athlete_id"] = stream.Athlete.ID
		result["username"] = stream.Athlete.Username
		result["display_name"] = stream.Athlete.DisplayName
		result["avatar_url"] = stream.Athlete.AvatarURL

		if stream.Athlete.AthleteProfile != nil {
			result["follower_count"] = stream.Athlete.AthleteProfile.FollowerCount
		}
	}

	return result
}
