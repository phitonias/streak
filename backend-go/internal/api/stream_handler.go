package api

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/phitonias/streak/internal/middleware"
	"github.com/phitonias/streak/internal/models"
	"github.com/phitonias/streak/internal/service"
	"github.com/phitonias/streak/internal/utils"
)

type StreamHandler struct {
	service *service.StreamService
}

func NewStreamHandler(service *service.StreamService) *StreamHandler {
	return &StreamHandler{service: service}
}

func (h *StreamHandler) CreateStream(c *gin.Context) {
	claims, ok := middleware.GetUserClaims(c)
	if !ok {
		utils.ErrorResponse(c, 401, "Not authenticated")
		return
	}

	var req middleware.CreateStreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, 400, "Invalid request body")
		return
	}

	if !models.IsValidSportCategory(req.SportCategory) {
		utils.ErrorResponse(c, 400, "Invalid sport category")
		return
	}

	input := service.CreateStreamInput{
		AthleteID:     claims.UserID,
		Title:         req.Title,
		Description:   req.Description,
		SportCategory: req.SportCategory,
		Tags:          req.Tags,
	}

	stream, err := h.service.CreateStream(input)
	if err != nil {
		utils.ErrorResponse(c, 400, err.Error())
		return
	}

	utils.SuccessMessageResponse(c, 201, "Stream created successfully", stream)
}

func (h *StreamHandler) GetStream(c *gin.Context) {
	streamID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid stream ID")
		return
	}

	stream, err := h.service.GetStreamByID(streamID)
	if err != nil {
		utils.ErrorResponse(c, 404, err.Error())
		return
	}

	utils.SuccessResponse(c, 200, stream)
}

func (h *StreamHandler) UpdateStream(c *gin.Context) {
	claims, ok := middleware.GetUserClaims(c)
	if !ok {
		utils.ErrorResponse(c, 401, "Not authenticated")
		return
	}

	streamID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid stream ID")
		return
	}

	var req middleware.UpdateStreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, 400, "Invalid request body")
		return
	}

	input := service.UpdateStreamInput{}

	if req.Title != "" {
		input.Title = &req.Title
	}
	if req.Description != "" {
		input.Description = &req.Description
	}
	if req.SportCategory != "" {
		if !models.IsValidSportCategory(req.SportCategory) {
			utils.ErrorResponse(c, 400, "Invalid sport category")
			return
		}
		input.SportCategory = &req.SportCategory
	}
	if req.Tags != nil {
		input.Tags = req.Tags
	}
	if req.ThumbnailURL != "" {
		input.ThumbnailURL = &req.ThumbnailURL
	}

	stream, err := h.service.UpdateStream(streamID, claims.UserID, input)
	if err != nil {
		utils.ErrorResponse(c, 400, err.Error())
		return
	}

	utils.SuccessMessageResponse(c, 200, "Stream updated successfully", stream)
}

func (h *StreamHandler) StartStream(c *gin.Context) {
	claims, ok := middleware.GetUserClaims(c)
	if !ok {
		utils.ErrorResponse(c, 401, "Not authenticated")
		return
	}

	streamID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid stream ID")
		return
	}

	stream, err := h.service.StartStream(claims.UserID, streamID)
	if err != nil {
		utils.ErrorResponse(c, 400, err.Error())
		return
	}

	utils.SuccessMessageResponse(c, 200, "Stream started successfully", stream)
}

func (h *StreamHandler) StopStream(c *gin.Context) {
	claims, ok := middleware.GetUserClaims(c)
	if !ok {
		utils.ErrorResponse(c, 401, "Not authenticated")
		return
	}

	streamID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid stream ID")
		return
	}

	if err := h.service.StopStream(claims.UserID, streamID); err != nil {
		utils.ErrorResponse(c, 400, err.Error())
		return
	}

	utils.SuccessMessageResponse(c, 200, "Stream stopped successfully", nil)
}

func (h *StreamHandler) GetLiveStreams(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	category := c.Query("category")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	streams, pagination, err := h.service.GetLiveStreams(page, limit, category)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed to get live streams")
		return
	}

	utils.PaginatedSuccessResponse(c, 200, streams, &utils.Pagination{
		Page:       pagination.Page,
		Limit:      pagination.Limit,
		Total:      pagination.Total,
		TotalPages: pagination.TotalPages,
	})
}

func (h *StreamHandler) GetAthleteStreams(c *gin.Context) {
	athleteID, err := uuid.Parse(c.Param("athleteId"))
	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid athlete ID")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	streams, pagination, err := h.service.GetStreamsByAthlete(athleteID, page, limit)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed to get athlete streams")
		return
	}

	utils.PaginatedSuccessResponse(c, 200, streams, &utils.Pagination{
		Page:       pagination.Page,
		Limit:      pagination.Limit,
		Total:      pagination.Total,
		TotalPages: pagination.TotalPages,
	})
}

func (h *StreamHandler) GetFollowingStreams(c *gin.Context) {
	claims, ok := middleware.GetUserClaims(c)
	if !ok {
		utils.ErrorResponse(c, 401, "Not authenticated")
		return
	}

	streams, err := h.service.GetFollowingStreams(claims.UserID)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed to get following streams")
		return
	}

	utils.SuccessResponse(c, 200, streams)
}
