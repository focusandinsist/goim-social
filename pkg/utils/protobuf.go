package utils

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"google.golang.org/protobuf/proto"
)

// ReadProtoRequest 读取并解析protobuf请求
func ReadProtoRequest(c *gin.Context, req proto.Message) error {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return err
	}
	return proto.Unmarshal(body, req)
}

// SendProtoResponse 发送protobuf响应
func SendProtoResponse(c *gin.Context, msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}
	c.Data(http.StatusOK, "application/x-protobuf", data)
	return nil
}

// SendProtoError 发送protobuf错误响应
func SendProtoError(c *gin.Context, statusCode int, msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}
	c.Data(statusCode, "application/x-protobuf", data)
	return nil
}

// WriteObject 兼容protobuf和json
func WriteObject(c *gin.Context, obj interface{}, err error) {
	status := http.StatusOK
	if err != nil {
		status = http.StatusBadRequest
	}

	switch c.ContentType() {
	case binding.MIMEPROTOBUF:
		if msg, ok := obj.(proto.Message); ok {
			c.ProtoBuf(status, msg)
			return
		}
		c.String(http.StatusInternalServerError, "expected proto.Message for protobuf response")

	default:
		c.JSON(status, obj)
	}
}
