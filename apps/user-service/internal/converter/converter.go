package converter

import (
	"goim-social/api/rest"
	"goim-social/apps/user-service/internal/model"
)

// Converter 转换器，提供Model到Protobuf的转换
type Converter struct{}

// NewConverter 创建转换器实例
func NewConverter() *Converter {
	return &Converter{}
}

// UserModelToProto 将用户Model转换为Protobuf
func (c *Converter) UserModelToProto(user *model.User) *rest.UserInfo {
	if user == nil {
		return nil
	}
	return &rest.UserInfo{
		Id:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Nickname:  user.Nickname,
		Avatar:    user.Avatar,
		Status:    int32(user.Status),
		CreatedAt: user.CreatedAt.Unix(),
		UpdatedAt: user.UpdatedAt.Unix(),
	}
}

// UserModelsToProto 将用户Model列表转换为Protobuf列表
func (c *Converter) UserModelsToProto(users []*model.User) []*rest.UserInfo {
	if users == nil {
		return []*rest.UserInfo{}
	}

	result := make([]*rest.UserInfo, 0, len(users))
	for _, user := range users {
		if protoUser := c.UserModelToProto(user); protoUser != nil {
			result = append(result, protoUser)
		}
	}
	return result
}

// 响应构建方法

// BuildRegisterResponse 构建注册响应
func (c *Converter) BuildRegisterResponse(success bool, message string, user *model.User, token string) *rest.RegisterResponse {
	return &rest.RegisterResponse{
		Success: success,
		Message: message,
		User:    c.UserModelToProto(user),
		Token:   token,
	}
}

// BuildLoginResponse 构建登录响应
func (c *Converter) BuildLoginResponse(success bool, message string, user *model.User, token string) *rest.LoginResponse {
	return &rest.LoginResponse{
		Success: success,
		Message: message,
		User:    c.UserModelToProto(user),
		Token:   token,
	}
}

// BuildGetUserResponse 构建获取用户信息响应
func (c *Converter) BuildGetUserResponse(success bool, message string, user *model.User) *rest.GetUserResponse {
	return &rest.GetUserResponse{
		Success: success,
		Message: message,
		User:    c.UserModelToProto(user),
	}
}

// BuildUpdateUserResponse 构建更新用户信息响应
func (c *Converter) BuildUpdateUserResponse(success bool, message string, user *model.User) *rest.UpdateUserResponse {
	return &rest.UpdateUserResponse{
		Success: success,
		Message: message,
		User:    c.UserModelToProto(user),
	}
}

// BuildDeleteUserResponse 构建删除用户响应
func (c *Converter) BuildDeleteUserResponse(success bool, message string) *rest.DeleteUserResponse {
	return &rest.DeleteUserResponse{
		Success: success,
		Message: message,
	}
}

// BuildListUsersResponse 构建用户列表响应
func (c *Converter) BuildListUsersResponse(success bool, message string, users []*model.User, total int64, page, pageSize int32) *rest.ListUsersResponse {
	return &rest.ListUsersResponse{
		Success:  success,
		Message:  message,
		Users:    c.UserModelsToProto(users),
		Total:    int32(total),
		Page:     page,
		PageSize: pageSize,
	}
}

// BuildChangePasswordResponse 构建修改密码响应
func (c *Converter) BuildChangePasswordResponse(success bool, message string) *rest.ChangePasswordResponse {
	return &rest.ChangePasswordResponse{
		Success: success,
		Message: message,
	}
}

// BuildUploadAvatarResponse 构建上传头像响应
func (c *Converter) BuildUploadAvatarResponse(success bool, message, avatarUrl string) *rest.UploadAvatarResponse {
	return &rest.UploadAvatarResponse{
		Success:   success,
		Message:   message,
		AvatarUrl: avatarUrl,
	}
}

// 便捷方法：构建错误响应

// BuildErrorRegisterResponse 构建注册错误响应
func (c *Converter) BuildErrorRegisterResponse(message string) *rest.RegisterResponse {
	return c.BuildRegisterResponse(false, message, nil, "")
}

// BuildErrorLoginResponse 构建登录错误响应
func (c *Converter) BuildErrorLoginResponse(message string) *rest.LoginResponse {
	return c.BuildLoginResponse(false, message, nil, "")
}

// BuildErrorGetUserResponse 构建获取用户错误响应
func (c *Converter) BuildErrorGetUserResponse(message string) *rest.GetUserResponse {
	return c.BuildGetUserResponse(false, message, nil)
}

// BuildSuccessResponse 构建通用成功响应
func (c *Converter) BuildSuccessResponse(message string) map[string]interface{} {
	return map[string]interface{}{
		"success": true,
		"message": message,
	}
}

// BuildErrorResponse 构建通用错误响应
func (c *Converter) BuildErrorResponse(message string) map[string]interface{} {
	return map[string]interface{}{
		"success": false,
		"message": message,
	}
}
