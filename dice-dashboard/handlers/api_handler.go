package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"dice-dashboard/services"
)

type APIHandler struct {
	cluster *services.ClusterService
}

func NewAPIHandler(cluster *services.ClusterService) *APIHandler {
	return &APIHandler{cluster: cluster}
}

func (h *APIHandler) GetOverview(c *gin.Context) {
	c.JSON(http.StatusOK, h.cluster.GetOverview())
}

func (h *APIHandler) GetNodes(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"nodes": h.cluster.GetNodes()})
}

func (h *APIHandler) GetHeartbeat(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"rows": h.cluster.GetHeartbeatRows()})
}

func (h *APIHandler) GetLogs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "30"))
	c.JSON(http.StatusOK, gin.H{"logs": h.cluster.GetActivityLogs(limit)})
}

func (h *APIHandler) GetReplicationLogs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	c.JSON(http.StatusOK, gin.H{"logs": h.cluster.GetReplicationLogs(limit)})
}

func (h *APIHandler) SetKey(c *gin.Context) {
	var payload services.KeyValuePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Key và Value là bắt buộc"})
		return
	}

	if err := h.cluster.Set(payload.Key, payload.Value); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "SET thành công", "key": payload.Key, "value": payload.Value})
}

func (h *APIHandler) GetKey(c *gin.Context) {
	key := c.Param("key")
	value, err := h.cluster.Get(key)
	if err != nil {
		if errors.Is(err, services.ErrKeyNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy key"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"key": key, "value": value})
}

func (h *APIHandler) DeleteKey(c *gin.Context) {
	key := c.Param("key")
	if err := h.cluster.Delete(key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "DELETE thành công", "key": key})
}
