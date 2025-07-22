package handler

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"

	"websocket-server/api/rest"
	"websocket-server/apps/friend-service/model"
	"websocket-server/apps/friend-service/service"
	"websocket-server/pkg/logger"
)

type Handler struct {
	service *service.Service
	logger  logger.Logger
}

func NewHandler(service *service.Service, logger logger.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// sendProtoResponse 发送protobuf响应
func (h *Handler) sendProtoResponse(c *gin.Context, msg proto.Message) {
	data, err := proto.Marshal(msg)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to marshal protobuf response", logger.F("error", err.Error()))
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Data(http.StatusOK, "application/x-protobuf", data)
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/friend")
	{
		api.POST("/add", h.AddFriend)                    // 添加好友
		api.POST("/delete", h.DeleteFriend)              // 删除好友
		api.POST("/list", h.ListFriends)                 // 查询好友列表
		api.POST("/get", h.GetFriend)                    // 获取单个好友信息
		api.POST("/update_remark", h.UpdateFriendRemark) // 更新好友备注
		api.POST("/apply", h.ApplyFriend)                // 申请加好友
		api.POST("/respond_apply", h.RespondFriendApply) // 同意/拒绝好友申请
		api.POST("/apply_list", h.ListFriendApply)       // 查询好友申请列表
	}
}

func (h *Handler) AddFriend(c *gin.Context) {
	ctx := c.Request.Context()

	// 读取protobuf请求体
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error(ctx, "Failed to read request body", logger.F("error", err.Error()))
		h.sendProtoResponse(c, &rest.AddFriendResponse{
			Success: false,
			Message: "Failed to read request body",
		})
		return
	}

	// 解析protobuf请求
	var req rest.AddFriendRequest
	if err := proto.Unmarshal(body, &req); err != nil {
		h.logger.Error(ctx, "Invalid protobuf request", logger.F("error", err.Error()))
		h.sendProtoResponse(c, &rest.AddFriendResponse{
			Success: false,
			Message: "Invalid protobuf request",
		})
		return
	}

	// 调用服务层
	if err := h.service.AddFriend(ctx, req.UserId, req.FriendId, req.Remark); err != nil {
		h.logger.Error(ctx, "Add friend failed", logger.F("error", err.Error()))
		h.sendProtoResponse(c, &rest.AddFriendResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	// 返回成功响应
	h.sendProtoResponse(c, &rest.AddFriendResponse{
		Success: true,
		Message: "添加好友成功",
	})
}

func (h *Handler) DeleteFriend(c *gin.Context) {
	ctx := c.Request.Context()
	var req model.DeleteFriendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid delete friend request", logger.F("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.DeleteFriend(ctx, req.UserID, req.FriendID); err != nil {
		h.logger.Error(ctx, "Delete friend failed", logger.F("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除好友成功"})
}

func (h *Handler) ListFriends(c *gin.Context) {
	ctx := c.Request.Context()
	var req model.ListFriendsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid list friends request", logger.F("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	friends, err := h.service.ListFriends(ctx, req.UserID)
	if err != nil {
		h.logger.Error(ctx, "List friends failed", logger.F("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"friends": friends})
}

func (h *Handler) GetFriend(c *gin.Context) {
	ctx := c.Request.Context()
	var req model.GetFriendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid get friend request", logger.F("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	friend, err := h.service.GetFriend(ctx, req.UserID, req.FriendID)
	if err != nil {
		h.logger.Error(ctx, "Get friend failed", logger.F("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"friend": friend})
}

func (h *Handler) UpdateFriendRemark(c *gin.Context) {
	ctx := c.Request.Context()
	var req model.UpdateFriendRemarkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid update friend remark request", logger.F("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.UpdateFriendRemark(ctx, req.UserID, req.FriendID, req.NewRemark); err != nil {
		h.logger.Error(ctx, "Update friend remark failed", logger.F("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "更新好友备注成功"})
}

// 申请加好友
func (h *Handler) ApplyFriend(c *gin.Context) {
	ctx := c.Request.Context()
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error(ctx, "Failed to read request body", logger.F("error", err.Error()))
		h.sendProtoResponse(c, &rest.ApplyFriendResponse{
			Success: false,
			Message: "Failed to read request body",
		})
		return
	}
	var req rest.ApplyFriendRequest
	if err := proto.Unmarshal(body, &req); err != nil {
		h.logger.Error(ctx, "Invalid protobuf request", logger.F("error", err.Error()))
		h.sendProtoResponse(c, &rest.ApplyFriendResponse{
			Success: false,
			Message: "Invalid protobuf request",
		})
		return
	}
	if err := h.service.ApplyFriend(ctx, req.UserId, req.FriendId, req.Remark); err != nil {
		h.logger.Error(ctx, "Apply friend failed", logger.F("error", err.Error()))
		h.sendProtoResponse(c, &rest.ApplyFriendResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	h.sendProtoResponse(c, &rest.ApplyFriendResponse{
		Success: true,
		Message: "好友申请已提交，等待对方处理",
	})
}

// 同意/拒绝好友申请
func (h *Handler) RespondFriendApply(c *gin.Context) {
	ctx := c.Request.Context()
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error(ctx, "Failed to read request body", logger.F("error", err.Error()))
		h.sendProtoResponse(c, &rest.RespondFriendApplyResponse{
			Success: false,
			Message: "Failed to read request body",
		})
		return
	}
	var req rest.RespondFriendApplyRequest
	if err := proto.Unmarshal(body, &req); err != nil {
		h.logger.Error(ctx, "Invalid protobuf request", logger.F("error", err.Error()))
		h.sendProtoResponse(c, &rest.RespondFriendApplyResponse{
			Success: false,
			Message: "Invalid protobuf request",
		})
		return
	}
	if err := h.service.RespondFriendApply(ctx, req.UserId, req.ApplicantId, req.Agree); err != nil {
		h.logger.Error(ctx, "Respond friend apply failed", logger.F("error", err.Error()))
		h.sendProtoResponse(c, &rest.RespondFriendApplyResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	h.sendProtoResponse(c, &rest.RespondFriendApplyResponse{
		Success: true,
		Message: "操作成功",
	})
}

// 查询好友申请列表
func (h *Handler) ListFriendApply(c *gin.Context) {
	ctx := c.Request.Context()
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error(ctx, "Failed to read request body", logger.F("error", err.Error()))
		c.Status(http.StatusBadRequest)
		return
	}
	var req rest.ListFriendApplyRequest
	if err := proto.Unmarshal(body, &req); err != nil {
		h.logger.Error(ctx, "Invalid protobuf request", logger.F("error", err.Error()))
		c.Status(http.StatusBadRequest)
		return
	}
	applies, err := h.service.ListFriendApply(ctx, req.UserId)
	if err != nil {
		h.logger.Error(ctx, "List friend apply failed", logger.F("error", err.Error()))
		c.Status(http.StatusInternalServerError)
		return
	}
	var resp rest.ListFriendApplyResponse
	for _, a := range applies {
		resp.Applies = append(resp.Applies, &rest.FriendApplyInfo{
			ApplicantId: a.ApplicantID,
			Remark:      a.Remark,
			Timestamp:   a.Timestamp,
			Status:      a.Status,
		})
	}
	data, _ := proto.Marshal(&resp)
	c.Data(http.StatusOK, "application/x-protobuf", data)
}
