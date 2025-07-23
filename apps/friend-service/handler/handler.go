package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"websocket-server/api/rest"
	"websocket-server/apps/friend-service/service"
	"websocket-server/pkg/logger"
	"websocket-server/pkg/utils"
)

type Handler struct {
	service *service.Service
	logger  logger.Logger
}

func NewHandler(service *service.Service, logger logger.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/friend")
	{
		api.POST("/list", h.ListFriends)                 // 查询好友列表
		api.POST("/profile", h.GetFriendProfile)         // 获取单个好友个人简介
		api.POST("/update_remark", h.UpdateFriendRemark) // 更新好友备注
		api.POST("/apply_list", h.ListFriendApply)       // 查询好友申请列表
		api.POST("/apply", h.ApplyFriend)                // 申请加好友
		api.POST("/respond_apply", h.RespondFriendApply) // 回应好友申请
		api.POST("/delete", h.DeleteFriend)              // 删除好友
	}
}

func (h *Handler) DeleteFriend(c *gin.Context) {
	ctx := c.Request.Context()

	var req rest.DeleteFriendRequest
	if err := utils.ReadProtoRequest(c, &req); err != nil {
		h.logger.Error(ctx, "Invalid protobuf request", logger.F("error", err.Error()))
		utils.SendProtoError(c, http.StatusBadRequest, &rest.DeleteFriendResponse{
			Success: false,
			Message: "Invalid protobuf request",
		})
		return
	}

	// 调用服务层
	if err := h.service.DeleteFriend(ctx, req.GetUserId(), req.GetFriendId()); err != nil {
		h.logger.Error(ctx, "Delete friend failed", logger.F("error", err.Error()))
		utils.SendProtoError(c, http.StatusInternalServerError, &rest.DeleteFriendResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	// 返回成功响应
	utils.SendProtoResponse(c, &rest.DeleteFriendResponse{
		Success: true,
		Message: "删除好友成功",
	})
}

func (h *Handler) ListFriends(c *gin.Context) {
	ctx := c.Request.Context()

	var req rest.ListFriendsRequest
	if err := utils.ReadProtoRequest(c, &req); err != nil {
		h.logger.Error(ctx, "Invalid protobuf request", logger.F("error", err.Error()))
		utils.SendProtoError(c, http.StatusBadRequest, &rest.ListFriendsResponse{
			Success: false,
			Message: "Invalid protobuf request",
		})
		return
	}

	// 调用服务层
	friends, err := h.service.ListFriends(ctx, req.GetUserId())
	if err != nil {
		h.logger.Error(ctx, "List friends failed", logger.F("error", err.Error()))
		utils.SendProtoError(c, http.StatusInternalServerError, &rest.ListFriendsResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	// 转换为protobuf格式
	var pbFriends []*rest.FriendInfo
	for _, friend := range friends {
		pbFriends = append(pbFriends, &rest.FriendInfo{
			UserId:    friend.UserID,
			FriendId:  friend.FriendID,
			Remark:    friend.Remark,
			CreatedAt: friend.CreatedAt,
		})
	}

	// 返回成功响应
	utils.SendProtoResponse(c, &rest.ListFriendsResponse{
		Success:  true,
		Message:  "查询成功",
		Friends:  pbFriends,
		Total:    int32(len(pbFriends)),
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
	})
}

func (h *Handler) GetFriendProfile(c *gin.Context) {
	ctx := c.Request.Context()

	var req rest.FriendProfileRequest
	if err := utils.ReadProtoRequest(c, &req); err != nil {
		h.logger.Error(ctx, "Invalid protobuf request", logger.F("error", err.Error()))
		utils.SendProtoError(c, http.StatusBadRequest, &rest.FriendProfileResponse{
			Success: false,
			Message: "Invalid protobuf request",
		})
		return
	}

	friend, err := h.service.GetFriend(ctx, req.GetUserId(), req.GetFriendId())
	if err != nil {
		h.logger.Error(ctx, "Get friend failed", logger.F("error", err.Error()))
		utils.SendProtoError(c, http.StatusInternalServerError, &rest.FriendProfileResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	utils.SendProtoResponse(c, &rest.FriendProfileResponse{
		Success: true,
		Message: "查询成功",
		Alias:   friend.Remark,
		// TODO: 还有其他字段，回头丰富一下
	})
}

func (h *Handler) UpdateFriendRemark(c *gin.Context) {
	ctx := c.Request.Context()

	var req rest.SetFriendAliasRequest
	if err := utils.ReadProtoRequest(c, &req); err != nil {
		h.logger.Error(ctx, "Invalid protobuf request", logger.F("error", err.Error()))
		utils.SendProtoError(c, http.StatusBadRequest, &rest.SetFriendAliasResponse{
			Success: false,
			Message: "Invalid protobuf request",
		})
		return
	}

	if err := h.service.UpdateFriendRemark(ctx, req.GetUserId(), req.GetFriendId(), req.GetAlias()); err != nil {
		h.logger.Error(ctx, "Update friend remark failed", logger.F("error", err.Error()))
		utils.SendProtoError(c, http.StatusInternalServerError, &rest.SetFriendAliasResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	utils.SendProtoResponse(c, &rest.SetFriendAliasResponse{
		Success: true,
		Message: "更新好友备注成功",
	})
}

func (h *Handler) ApplyFriend(c *gin.Context) {
	ctx := c.Request.Context()

	var req rest.ApplyFriendRequest
	if err := utils.ReadProtoRequest(c, &req); err != nil {
		h.logger.Error(ctx, "Invalid protobuf request", logger.F("error", err.Error()))
		utils.SendProtoError(c, http.StatusBadRequest, &rest.ApplyFriendResponse{
			Success: false,
			Message: "Invalid protobuf request",
		})
		return
	}

	// 调用服务层
	if err := h.service.ApplyFriend(ctx, req.UserId, req.FriendId, req.Remark); err != nil {
		h.logger.Error(ctx, "Apply friend failed", logger.F("error", err.Error()))
		utils.SendProtoError(c, http.StatusInternalServerError, &rest.ApplyFriendResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	// 返回成功响应
	utils.SendProtoResponse(c, &rest.ApplyFriendResponse{
		Success: true,
		Message: "好友申请已提交，等待对方处理",
	})
}

func (h *Handler) RespondFriendApply(c *gin.Context) {
	ctx := c.Request.Context()

	var req rest.RespondFriendApplyRequest
	if err := utils.ReadProtoRequest(c, &req); err != nil {
		h.logger.Error(ctx, "Invalid protobuf request", logger.F("error", err.Error()))
		utils.SendProtoError(c, http.StatusBadRequest, &rest.RespondFriendApplyResponse{
			Success: false,
			Message: "Invalid protobuf request",
		})
		return
	}

	// 调用服务层
	if err := h.service.RespondFriendApply(ctx, req.UserId, req.ApplicantId, req.Agree); err != nil {
		h.logger.Error(ctx, "Respond friend apply failed", logger.F("error", err.Error()))
		utils.SendProtoError(c, http.StatusInternalServerError, &rest.RespondFriendApplyResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	// 返回成功响应
	utils.SendProtoResponse(c, &rest.RespondFriendApplyResponse{
		Success: true,
		Message: "操作成功",
	})
}

// ListFriendApply 查询好友申请列表
func (h *Handler) ListFriendApply(c *gin.Context) {
	ctx := c.Request.Context()

	var req rest.ListFriendApplyRequest
	if err := utils.ReadProtoRequest(c, &req); err != nil {
		h.logger.Error(ctx, "Invalid protobuf request", logger.F("error", err.Error()))
		c.Status(http.StatusBadRequest)
		return
	}

	// 调用服务层
	applies, err := h.service.ListFriendApply(ctx, req.UserId)
	if err != nil {
		h.logger.Error(ctx, "List friend apply failed", logger.F("error", err.Error()))
		c.Status(http.StatusInternalServerError)
		return
	}

	// 转换为protobuf格式
	var pbApplies []*rest.FriendApplyInfo
	for _, a := range applies {
		pbApplies = append(pbApplies, &rest.FriendApplyInfo{
			ApplicantId: a.ApplicantID,
			Remark:      a.Remark,
			Timestamp:   a.Timestamp,
			Status:      a.Status,
		})
	}

	// 返回成功响应
	utils.SendProtoResponse(c, &rest.ListFriendApplyResponse{
		Applies: pbApplies,
	})
}
