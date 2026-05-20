package handler

import (
	"net/http"

	"git.myservermanager.com/varakh/ecolinker/internal/api"
	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Status(c *gin.Context) {
	c.JSON(http.StatusOK, api.NewDataResponseWithPayload(api.NewHealthResponse(true)))
}
