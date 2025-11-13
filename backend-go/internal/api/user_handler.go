package api

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/phitonias/streak/internal/middleware"
	"github.com/phitonias/streak/internal/service"
	"github.com/phitonias/streak/internal/utils"
)

type UserHandler struct {
	service *service.UserService
}

func NewUserHandler(service *service.UserService) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) GetUserByID(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid user ID")
		return
	}

	user, err := h.service.GetUserByID(userID)
	if err != nil {
		utils.ErrorResponse(c, 404, err.Error())
		return
	}

	utils.SuccessResponse(c, 200, user)
}

func (h *UserHandler) GetUserByUsername(c *gin.Context) {
	username := c.Param("username")

	user, err := h.service.GetUserByUsername(username)
	if err != nil {
		utils.ErrorResponse(c, 404, err.Error())
		return
	}

	utils.SuccessResponse(c, 200, user)
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	claims, ok := middleware.GetUserClaims(c)
	if !ok {
		utils.ErrorResponse(c, 401, "Not authenticated")
		return
	}

	var req middleware.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, 400, "Invalid request body")
		return
	}

	input := service.UpdateProfileInput{
		SportCategories: req.SportCategories,
	}

	if req.DisplayName != "" {
		input.DisplayName = &req.DisplayName
	}
	if req.Bio != "" {
		input.Bio = &req.Bio
	}
	if req.AvatarURL != "" {
		input.AvatarURL = &req.AvatarURL
	}
	if req.BannerURL != "" {
		input.BannerURL = &req.BannerURL
	}

	user, err := h.service.UpdateProfile(claims.UserID, input)
	if err != nil {
		utils.ErrorResponse(c, 400, err.Error())
		return
	}

	utils.SuccessMessageResponse(c, 200, "Profile updated successfully", user)
}

func (h *UserHandler) FollowUser(c *gin.Context) {
	claims, ok := middleware.GetUserClaims(c)
	if !ok {
		utils.ErrorResponse(c, 401, "Not authenticated")
		return
	}

	athleteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid user ID")
		return
	}

	if err := h.service.FollowUser(claims.UserID, athleteID); err != nil {
		utils.ErrorResponse(c, 400, err.Error())
		return
	}

	utils.SuccessMessageResponse(c, 200, "Successfully followed user", nil)
}

func (h *UserHandler) UnfollowUser(c *gin.Context) {
	claims, ok := middleware.GetUserClaims(c)
	if !ok {
		utils.ErrorResponse(c, 401, "Not authenticated")
		return
	}

	athleteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid user ID")
		return
	}

	if err := h.service.UnfollowUser(claims.UserID, athleteID); err != nil {
		utils.ErrorResponse(c, 400, err.Error())
		return
	}

	utils.SuccessMessageResponse(c, 200, "Successfully unfollowed user", nil)
}

func (h *UserHandler) GetFollowers(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid user ID")
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

	followers, pagination, err := h.service.GetFollowers(userID, page, limit)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed to get followers")
		return
	}

	utils.PaginatedSuccessResponse(c, 200, followers, &utils.Pagination{
		Page:       pagination.Page,
		Limit:      pagination.Limit,
		Total:      pagination.Total,
		TotalPages: pagination.TotalPages,
	})
}

func (h *UserHandler) GetFollowing(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid user ID")
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

	following, pagination, err := h.service.GetFollowing(userID, page, limit)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed to get following")
		return
	}

	utils.PaginatedSuccessResponse(c, 200, following, &utils.Pagination{
		Page:       pagination.Page,
		Limit:      pagination.Limit,
		Total:      pagination.Total,
		TotalPages: pagination.TotalPages,
	})
}

func (h *UserHandler) CheckFollowing(c *gin.Context) {
	claims, ok := middleware.GetUserClaims(c)
	if !ok {
		utils.ErrorResponse(c, 401, "Not authenticated")
		return
	}

	athleteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid user ID")
		return
	}

	isFollowing, err := h.service.IsFollowing(claims.UserID, athleteID)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed to check following status")
		return
	}

	utils.SuccessResponse(c, 200, gin.H{"isFollowing": isFollowing})
}
