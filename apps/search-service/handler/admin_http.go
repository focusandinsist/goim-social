package handler

import (
	"github.com/gin-gonic/gin"

	"goim-social/api/rest"
	"goim-social/apps/search-service/service"
	"goim-social/pkg/httpx"
	"goim-social/pkg/logger"
)

// CreateIndex 创建索引
func (h *HTTPHandler) CreateIndex(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		resp interface{}
		err  error
	)

	req := &rest.CreateIndexRequest{}
	if err = c.Bind(req); err != nil {
		resp = h.converter.BuildHTTPErrorResponse("invalid request: " + err.Error())
		httpx.WriteObject(c, resp, err)
		return
	}

	err = h.indexService.CreateIndex(ctx, req.IndexName, req.IndexType)
	if err != nil {
		h.logger.Error(ctx, "CreateIndex failed", logger.F("error", err.Error()))
		resp = h.converter.BuildHTTPErrorResponse("create index failed: " + err.Error())
	} else {
		resp = map[string]interface{}{
			"code":    0,
			"message": "success",
			"data":    map[string]interface{}{"success": true},
		}
	}

	httpx.WriteObject(c, resp, err)
}

// DeleteIndex 删除索引
func (h *HTTPHandler) DeleteIndex(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		resp interface{}
		err  error
	)

	req := &rest.DeleteIndexRequest{}
	req.IndexName = c.Param("name")

	err = h.indexService.DeleteIndex(ctx, req.IndexName)
	if err != nil {
		h.logger.Error(ctx, "DeleteIndex failed", logger.F("error", err.Error()))
		resp = h.converter.BuildHTTPErrorResponse("delete index failed: " + err.Error())
	} else {
		resp = map[string]interface{}{
			"code":    0,
			"message": "success",
			"data":    map[string]interface{}{"success": true},
		}
	}

	httpx.WriteObject(c, resp, err)
}

// ReindexAll 重建所有索引
func (h *HTTPHandler) ReindexAll(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		resp interface{}
		err  error
	)

	req := &rest.ReindexAllRequest{}
	if err = c.Bind(req); err != nil {
		resp = h.converter.BuildHTTPErrorResponse("invalid request: " + err.Error())
		httpx.WriteObject(c, resp, err)
		return
	}

	err = h.indexService.ReindexAll(ctx)
	if err != nil {
		h.logger.Error(ctx, "ReindexAll failed", logger.F("error", err.Error()))
		resp = h.converter.BuildHTTPErrorResponse("reindex all failed: " + err.Error())
	} else {
		resp = map[string]interface{}{
			"code":    0,
			"message": "success",
			"data":    map[string]interface{}{"success": true},
		}
	}

	httpx.WriteObject(c, resp, err)
}

// ReindexByType 按类型重建索引
func (h *HTTPHandler) ReindexByType(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		resp interface{}
		err  error
	)

	req := &rest.ReindexByTypeRequest{}
	req.IndexType = c.Param("type")
	if err = c.Bind(req); err != nil {
		resp = h.converter.BuildHTTPErrorResponse("invalid request: " + err.Error())
		httpx.WriteObject(c, resp, err)
		return
	}

	err = h.indexService.ReindexByType(ctx, req.IndexType)
	if err != nil {
		h.logger.Error(ctx, "ReindexByType failed", logger.F("error", err.Error()))
		resp = h.converter.BuildHTTPErrorResponse("reindex by type failed: " + err.Error())
	} else {
		resp = map[string]interface{}{
			"code":    0,
			"message": "success",
			"data":    map[string]interface{}{"success": true},
		}
	}

	httpx.WriteObject(c, resp, err)
}

// IndexDocument 索引文档
func (h *HTTPHandler) IndexDocument(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		resp interface{}
		err  error
	)

	req := &rest.IndexDocumentRequest{}
	req.IndexType = c.Param("type")
	req.DocumentId = c.Param("id")
	if err = c.Bind(req); err != nil {
		resp = h.converter.BuildHTTPErrorResponse("invalid request: " + err.Error())
		httpx.WriteObject(c, resp, err)
		return
	}

	err = h.indexService.IndexDocument(ctx, req.IndexType, req.DocumentId, req.Document)
	if err != nil {
		h.logger.Error(ctx, "IndexDocument failed", logger.F("error", err.Error()))
		resp = h.converter.BuildHTTPErrorResponse("index document failed: " + err.Error())
	} else {
		resp = map[string]interface{}{
			"code":    0,
			"message": "success",
			"data":    map[string]interface{}{"success": true},
		}
	}

	httpx.WriteObject(c, resp, err)
}

// UpdateDocument 更新文档
func (h *HTTPHandler) UpdateDocument(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		resp interface{}
		err  error
	)

	req := &rest.UpdateDocumentRequest{}
	req.IndexType = c.Param("type")
	req.DocumentId = c.Param("id")
	if err = c.Bind(req); err != nil {
		resp = h.converter.BuildHTTPErrorResponse("invalid request: " + err.Error())
		httpx.WriteObject(c, resp, err)
		return
	}

	err = h.indexService.UpdateDocument(ctx, req.IndexType, req.DocumentId, req.Document)
	if err != nil {
		h.logger.Error(ctx, "UpdateDocument failed", logger.F("error", err.Error()))
		resp = h.converter.BuildHTTPErrorResponse("update document failed: " + err.Error())
	} else {
		resp = map[string]interface{}{
			"code":    0,
			"message": "success",
			"data":    map[string]interface{}{"success": true},
		}
	}

	httpx.WriteObject(c, resp, err)
}

// DeleteDocument 删除文档
func (h *HTTPHandler) DeleteDocument(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		resp interface{}
		err  error
	)

	req := &rest.DeleteDocumentRequest{}
	req.IndexType = c.Param("type")
	req.DocumentId = c.Param("id")

	err = h.indexService.DeleteDocument(ctx, req.IndexType, req.DocumentId)
	if err != nil {
		h.logger.Error(ctx, "DeleteDocument failed", logger.F("error", err.Error()))
		resp = h.converter.BuildHTTPErrorResponse("delete document failed: " + err.Error())
	} else {
		resp = map[string]interface{}{
			"code":    0,
			"message": "success",
			"data":    map[string]interface{}{"success": true},
		}
	}

	httpx.WriteObject(c, resp, err)
}

// BulkIndexDocuments 批量索引文档
func (h *HTTPHandler) BulkIndexDocuments(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		resp interface{}
		err  error
	)

	req := &rest.BulkIndexDocumentsRequest{}
	req.IndexType = c.Param("type")
	if err = c.Bind(req); err != nil {
		resp = h.converter.BuildHTTPErrorResponse("invalid request: " + err.Error())
		httpx.WriteObject(c, resp, err)
		return
	}

	// 转换proto的IndexDocument到service的IndexDocument
	serviceDocuments := make([]service.IndexDocument, len(req.Documents))
	for i, doc := range req.Documents {
		serviceDocuments[i] = service.IndexDocument{
			ID:       doc.Id,
			Document: doc.Data,
		}
	}

	err = h.indexService.BulkIndexDocuments(ctx, req.IndexType, serviceDocuments)
	if err != nil {
		h.logger.Error(ctx, "BulkIndexDocuments failed", logger.F("error", err.Error()))
		resp = h.converter.BuildHTTPErrorResponse("bulk index documents failed: " + err.Error())
	} else {
		resp = map[string]interface{}{
			"code":    0,
			"message": "success",
			"data":    map[string]interface{}{"success": true},
		}
	}

	httpx.WriteObject(c, resp, err)
}

// SyncFromDatabase 从数据库同步
func (h *HTTPHandler) SyncFromDatabase(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		resp interface{}
		err  error
	)

	req := &rest.SyncFromDatabaseRequest{}
	if err = c.Bind(req); err != nil {
		resp = h.converter.BuildHTTPErrorResponse("invalid request: " + err.Error())
		httpx.WriteObject(c, resp, err)
		return
	}

	err = h.indexService.SyncFromDatabase(ctx, req.SourceService, req.SourceTable, req.TargetIndex)
	if err != nil {
		h.logger.Error(ctx, "SyncFromDatabase failed", logger.F("error", err.Error()))
		resp = h.converter.BuildHTTPErrorResponse("sync from database failed: " + err.Error())
	} else {
		resp = map[string]interface{}{
			"code":    0,
			"message": "success",
			"data":    map[string]interface{}{"success": true},
		}
	}

	httpx.WriteObject(c, resp, err)
}

// GetSyncStatus 获取同步状态
func (h *HTTPHandler) GetSyncStatus(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		resp interface{}
		err  error
	)

	req := &rest.GetSyncStatusRequest{}
	req.SourceTable = c.Query("source_table")
	req.TargetIndex = c.Query("target_index")

	result, err := h.indexService.GetSyncStatus(ctx, req.SourceTable, req.TargetIndex)
	if err != nil {
		h.logger.Error(ctx, "GetSyncStatus failed", logger.F("error", err.Error()))
		resp = h.converter.BuildHTTPErrorResponse("get sync status failed: " + err.Error())
	} else {
		resp = map[string]interface{}{
			"code":    0,
			"message": "success",
			"data":    result,
		}
	}

	httpx.WriteObject(c, resp, err)
}

// ListSyncStatuses 列出同步状态
func (h *HTTPHandler) ListSyncStatuses(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		resp interface{}
		err  error
	)

	sourceService := c.Query("source_service")
	result, err := h.indexService.ListSyncStatuses(ctx, sourceService)
	if err != nil {
		h.logger.Error(ctx, "ListSyncStatuses failed", logger.F("error", err.Error()))
		resp = h.converter.BuildHTTPErrorResponse("list sync statuses failed: " + err.Error())
	} else {
		resp = map[string]interface{}{
			"code":    0,
			"message": "success",
			"data":    result,
		}
	}

	httpx.WriteObject(c, resp, err)
}

// HealthCheck 健康检查
func (h *HTTPHandler) HealthCheck(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		resp interface{}
		err  error
	)

	err = h.indexService.HealthCheck(ctx)
	if err != nil {
		h.logger.Error(ctx, "HealthCheck failed", logger.F("error", err.Error()))
		resp = h.converter.BuildHTTPErrorResponse("health check failed: " + err.Error())
	} else {
		resp = map[string]interface{}{
			"code":    0,
			"message": "success",
			"data":    map[string]interface{}{"status": "healthy"},
		}
	}

	httpx.WriteObject(c, resp, err)
}

// GetClusterInfo 获取集群信息
func (h *HTTPHandler) GetClusterInfo(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		resp interface{}
		err  error
	)

	result, err := h.indexService.GetClusterInfo(ctx)
	if err != nil {
		h.logger.Error(ctx, "GetClusterInfo failed", logger.F("error", err.Error()))
		resp = h.converter.BuildHTTPErrorResponse("get cluster info failed: " + err.Error())
	} else {
		resp = map[string]interface{}{
			"code":    0,
			"message": "success",
			"data":    result,
		}
	}

	httpx.WriteObject(c, resp, err)
}
