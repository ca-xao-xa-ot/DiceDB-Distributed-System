package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"dice-dashboard/services"
)

type PageHandler struct {
	cluster *services.ClusterService
}

func NewPageHandler(cluster *services.ClusterService) *PageHandler {
	return &PageHandler{cluster: cluster}
}

func (h *PageHandler) Index(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "Dice Distributed KV Store Dashboard",
	})
}
