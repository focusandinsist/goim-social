package handler

import (
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
	var (
		ctx = c.Request.Context()
		req rest.DeleteFriendRequest
		res *rest.DeleteFriendResponse
		err error
	)
	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid delete friend request", logger.F("error", err.Error()))
		res := &rest.DeleteFriendResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	err = h.service.DeleteFriend(ctx, req.GetUserId(), req.GetFriendId())
	res = &rest.DeleteFriendResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "删除好友成功"
		}(),
	}
	if err != nil {
		h.logger.Error(ctx, "Delete friend failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

func (h *Handler) ListFriends(c *gin.Context) {
	var (
		ctx = c.Request.Context()
		req rest.ListFriendsRequest
		res *rest.ListFriendsResponse
		err error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "bind failed", logger.F("error", err.Error()))
		utils.WriteObject(c, res, err)
		return
	}

	// 调用服务层
	friends, err := h.service.ListFriends(ctx, req.GetUserId())
	if err != nil {
		h.logger.Error(ctx, "List friends failed", logger.F("error", err.Error()))
		res := &rest.ListFriendsResponse{
			Success: false,
			Message: err.Error(),
		}
		utils.WriteObject(c, res, err)
		return
	}

	var friendList []*rest.FriendInfo
	for _, friend := range friends {
		friendList = append(friendList, &rest.FriendInfo{
			UserId:    friend.UserID,
			FriendId:  friend.FriendID,
			Remark:    friend.Remark,
			CreatedAt: friend.CreatedAt,
		})
	}

	// 返回成功响应
	res = &rest.ListFriendsResponse{
		Success:  true,
		Message:  "查询成功",
		Friends:  friendList,
		Total:    int32(len(friendList)),
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
	}

	utils.WriteObject(c, res, err)
}

func (h *Handler) GetFriendProfile(c *gin.Context) {

	var (
		ctx = c.Request.Context()
		req rest.FriendProfileRequest
		res *rest.FriendProfileResponse
		err error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid friend profile request", logger.F("error", err.Error()))
		res := &rest.FriendProfileResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	friend, err := h.service.GetFriend(ctx, req.GetUserId(), req.GetFriendId())
	res = &rest.FriendProfileResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "获取好友简介成功"
		}(),
		Alias: func() string {
			if friend != nil {
				return friend.Remark
			}
			return ""
		}(),
		Nickname: "好友昵称",  // TODO: 待丰富
		Avatar:   "头像URL", // TODO: 待丰富
		Age:      25,      // TODO: 待丰富
		Gender:   "未知",    // TODO: 待丰富
	}
	if err != nil {
		h.logger.Error(ctx, "Get friend failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

func (h *Handler) UpdateFriendRemark(c *gin.Context) {

	var (
		ctx = c.Request.Context()
		req rest.SetFriendAliasRequest
		res *rest.SetFriendAliasResponse
		err error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid set friend alias request", logger.F("error", err.Error()))
		res := &rest.SetFriendAliasResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	err = h.service.UpdateFriendRemark(ctx, req.GetUserId(), req.GetFriendId(), req.GetAlias())
	res = &rest.SetFriendAliasResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "更新好友备注成功"
		}(),
	}
	if err != nil {
		h.logger.Error(ctx, "Update friend remark failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

func (h *Handler) ApplyFriend(c *gin.Context) {

	var (
		ctx = c.Request.Context()
		req rest.ApplyFriendRequest
		res *rest.ApplyFriendResponse
		err error
	)
	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid apply friend request", logger.F("error", err.Error()))
		res := &rest.ApplyFriendResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	err = h.service.ApplyFriend(ctx, req.UserId, req.FriendId, req.Remark)
	res = &rest.ApplyFriendResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "好友申请已提交，等待对方处理"
		}(),
	}
	if err != nil {
		h.logger.Error(ctx, "Apply friend failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

func (h *Handler) RespondFriendApply(c *gin.Context) {
	var (
		ctx = c.Request.Context()
		req rest.RespondFriendApplyRequest
		res *rest.RespondFriendApplyResponse
		err error
	)
	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid respond friend apply request", logger.F("error", err.Error()))
		res := &rest.RespondFriendApplyResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	err = h.service.RespondFriendApply(ctx, req.UserId, req.ApplicantId, req.Agree)
	res = &rest.RespondFriendApplyResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "操作成功"
		}(),
	}
	if err != nil {
		h.logger.Error(ctx, "Respond friend apply failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

// ListFriendApply 查询好友申请列表
func (h *Handler) ListFriendApply(c *gin.Context) {

	var (
		ctx = c.Request.Context()
		req rest.ListFriendApplyRequest
		res *rest.ListFriendApplyResponse
		err error
	)

	if err := c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid list friend apply request", logger.F("error", err.Error()))
		res := &rest.ListFriendApplyResponse{
			Applies: []*rest.FriendApplyInfo{},
		}
		utils.WriteObject(c, res, err)
		return
	}

	// 调用服务层
	applies, err := h.service.ListFriendApply(ctx, req.UserId)

	var applyList []*rest.FriendApplyInfo
	if err == nil {
		for _, a := range applies {
			applyList = append(applyList, &rest.FriendApplyInfo{
				ApplicantId: a.ApplicantID,
				Remark:      a.Remark,
				Timestamp:   a.Timestamp,
				Status:      a.Status,
			})
		}
	}

	res = &rest.ListFriendApplyResponse{Applies: applyList}
	if err != nil {
		h.logger.Error(ctx, "List friend apply failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}
