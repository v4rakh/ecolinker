package handler

import (
	"git.myservermanager.com/varakh/ecolinker/api"
	"git.myservermanager.com/varakh/ecolinker/internal/app"
	"git.myservermanager.com/varakh/ecolinker/internal/server/config"
	"github.com/gin-gonic/gin"
	"net/http"
)

type InfoHandler struct {
	appConfig config.App
}

func NewInfoHandler(a *config.App) *InfoHandler {
	return &InfoHandler{appConfig: *a}
}

func (h *InfoHandler) Status(c *gin.Context) {
	c.JSON(http.StatusOK, api.NewDataResponseWithPayload(api.NewInfoResponse(app.Name, app.Version, h.appConfig.TimeZone)))
}
