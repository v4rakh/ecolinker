package handler

import (
	"git.myservermanager.com/varakh/ecolinker/internal/api"
	httpcommons "git.myservermanager.com/varakh/ecolinker/internal/http"
	"git.myservermanager.com/varakh/ecolinker/internal/server/constant"
	"git.myservermanager.com/varakh/ecolinker/internal/server/model"
	"git.myservermanager.com/varakh/ecolinker/internal/server/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

type DeviceHandler struct {
	service service.DeviceService
}

func NewDeviceHandler(s *service.DeviceService) *DeviceHandler {
	return &DeviceHandler{service: *s}
}

func (h *DeviceHandler) GetAll(c *gin.Context) {
	var entities []*model.Device
	var err error

	if entities, err = h.service.GetAll(); err != nil {
		_ = c.AbortWithError(ToHttpStatus(err), err)
		return
	}

	var data []*api.DeviceResponse
	data = make([]*api.DeviceResponse, 0, len(entities))

	for _, e := range entities {
		data = append(data, &api.DeviceResponse{
			SN:        e.SN,
			Kind:      e.Kind,
			Label:     e.Label,
			CreatedAt: e.CreatedAt,
			UpdatedAt: e.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, api.NewDataResponseWithPayload(api.NewDevicePageResponse(data)))
}

func (h *DeviceHandler) Get(c *gin.Context) {
	var err error
	var pathParams api.SNUriRequest

	if err = c.ShouldBindUri(&pathParams); err != nil {
		AbortWithValidatorPayload(c, err)
		return
	}

	var e *model.Device

	if e, err = h.service.Get(pathParams.SN); err != nil {
		_ = c.AbortWithError(ToHttpStatus(err), err)
		return
	}

	c.JSON(http.StatusOK, api.NewDeviceSingleResponse(e.SN, e.Kind, e.Label, e.CreatedAt, e.UpdatedAt))
}

func (h *DeviceHandler) Create(c *gin.Context) {
	var e *model.Device
	var err error

	var req api.CreateDeviceRequest

	if err = c.ShouldBindJSON(&req); err != nil {
		AbortWithValidatorPayload(c, err)
		return
	}

	if e, err = h.service.Create(req.SN, constant.DeviceKind(req.Kind), req.Label); err != nil {
		_ = c.AbortWithError(ToHttpStatus(err), err)
		return
	}

	c.JSON(http.StatusOK, api.NewDeviceSingleResponse(e.SN, e.Kind, e.Label, e.CreatedAt, e.UpdatedAt))
}

func (h *DeviceHandler) Update(c *gin.Context) {
	var e *model.Device
	var err error

	var req api.ModifyDeviceRequest

	if err = c.ShouldBindJSON(&req); err != nil {
		AbortWithValidatorPayload(c, err)
		return
	}

	if e, err = h.service.Update(req.SN, constant.DeviceKind(req.Kind), req.Label); err != nil {
		_ = c.AbortWithError(ToHttpStatus(err), err)
		return
	}

	c.JSON(http.StatusOK, api.NewDeviceSingleResponse(e.SN, e.Kind, e.Label, e.CreatedAt, e.UpdatedAt))
}

func (h *DeviceHandler) Delete(c *gin.Context) {
	var err error
	var pathParams api.SNUriRequest

	if err = c.ShouldBindUri(&pathParams); err != nil {
		AbortWithValidatorPayload(c, err)
		return
	}

	if err = h.service.Delete(pathParams.SN); err != nil {
		_ = c.AbortWithError(ToHttpStatus(err), err)
		return
	}

	c.Header(httpcommons.HeaderContentType, httpcommons.HeaderContentTypeApplicationJson)
	c.Status(http.StatusNoContent)
}
