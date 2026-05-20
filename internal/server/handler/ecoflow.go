package handler

import (
	"net/http"
	"time"

	"git.myservermanager.com/varakh/ecolinker/internal/api"
	"git.myservermanager.com/varakh/ecolinker/internal/server/dto"
	"git.myservermanager.com/varakh/ecolinker/internal/server/service"
	"git.myservermanager.com/varakh/ecolinker/internal/service_error"
	"github.com/gin-gonic/gin"
)

type EcoFlowHandler struct {
	httpService *service.EcoFlowHttpService
	mqttService *service.EcoFlowMqttService
}

func NewEcoFlowHandler(h *service.EcoFlowHttpService, m *service.EcoFlowMqttService) *EcoFlowHandler {
	return &EcoFlowHandler{httpService: h, mqttService: m}
}

func (h *EcoFlowHandler) Devices(c *gin.Context) {
	var err error

	var entities []dto.EcoFlowDeviceItem
	if entities, err = h.httpService.GetDevices(c.Request.Context()); err != nil {
		_ = c.AbortWithError(ToHttpStatus(err), err)
		return
	}

	var data []*api.EcoFlowDeviceResponse
	data = make([]*api.EcoFlowDeviceResponse, 0, len(entities))

	for _, e := range entities {
		data = append(data, &api.EcoFlowDeviceResponse{
			SN:     e.SN,
			Online: e.Online,
		})
	}

	c.JSON(http.StatusOK, api.NewDataResponseWithPayload(api.NewEcoFlowDeviceListResponse(data)))
}

func (h *EcoFlowHandler) Parameters(c *gin.Context) {
	var err error
	var pathParams api.SNUriRequest

	if err = c.ShouldBindUri(&pathParams); err != nil {
		AbortWithValidatorPayload(c, err)
		return
	}

	var req api.EcoFlowDeviceParametersRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		AbortWithValidatorPayload(c, err)
		return
	}

	var data map[string]interface{}
	if data, err = h.httpService.GetParameters(c.Request.Context(), pathParams.SN, req.Parameters); err != nil {
		_ = c.AbortWithError(ToHttpStatus(err), err)
		return
	}

	c.JSON(http.StatusOK, api.NewDataResponseWithPayload(data))
}

func (h *EcoFlowHandler) ParametersAll(c *gin.Context) {
	var err error
	var pathParams api.SNUriRequest

	if err = c.ShouldBindUri(&pathParams); err != nil {
		AbortWithValidatorPayload(c, err)
		return
	}

	var data map[string]interface{}
	if data, err = h.httpService.GetAllParameters(c.Request.Context(), pathParams.SN); err != nil {
		_ = c.AbortWithError(ToHttpStatus(err), err)
		return
	}

	c.JSON(http.StatusOK, api.NewDataResponseWithPayload(data))
}

func (h *EcoFlowHandler) Batteries(c *gin.Context) {
	var err error
	var pathParams api.SNUriRequest

	if err = c.ShouldBindUri(&pathParams); err != nil {
		AbortWithValidatorPayload(c, err)
		return
	}

	var data map[string]interface{}
	if data, err = h.httpService.GetBatteries(c.Request.Context(), pathParams.SN); err != nil {
		_ = c.AbortWithError(ToHttpStatus(err), err)
		return
	}

	c.JSON(http.StatusOK, api.NewDataResponseWithPayload(data))
}

func (h *EcoFlowHandler) History(c *gin.Context) {
	var err error

	var pathParams api.SNUriRequest
	if err = c.ShouldBindUri(&pathParams); err != nil {
		AbortWithValidatorPayload(c, err)
		return
	}

	var queryParams api.HistoryQueryParameterRequest
	if err = c.ShouldBindQuery(&queryParams); err != nil {
		AbortWithValidatorPayload(c, err)
		return
	}

	var beginTime, endTime time.Time
	if beginTime, err = time.Parse(time.DateTime, queryParams.BeginTime); err != nil {
		_ = c.AbortWithError(ToHttpStatus(service_error.ErrValidationTimeFormatDateTime), service_error.ErrValidationTimeFormatDateTime)
		return
	}
	if endTime, err = time.Parse(time.DateTime, queryParams.EndTime); err != nil {
		_ = c.AbortWithError(ToHttpStatus(service_error.ErrValidationTimeFormatDateTime), service_error.ErrValidationTimeFormatDateTime)
		return
	}

	var data dto.EcoFlowHistoryData
	if data, err = h.httpService.GetHistory(c.Request.Context(), pathParams.SN, beginTime, endTime); err != nil {
		_ = c.AbortWithError(ToHttpStatus(err), err)
		return
	}

	c.JSON(http.StatusOK, api.NewDataResponseWithPayload(data))
}

func (h *EcoFlowHandler) BrokerStatus(c *gin.Context) {
	enabled, connected := h.mqttService.Status()
	c.JSON(http.StatusOK, api.NewDataResponseWithPayload(api.NewEcoFlowBrokerStatusResponse(enabled, connected)))
}
