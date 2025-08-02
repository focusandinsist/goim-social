package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"google.golang.org/protobuf/proto"
)

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
