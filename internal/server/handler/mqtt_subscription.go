package handler

import (
	"net/http"

	"git.myservermanager.com/varakh/ecolinker/internal/api"
	httpcommons "git.myservermanager.com/varakh/ecolinker/internal/http"
	"git.myservermanager.com/varakh/ecolinker/internal/server/constant"
	"git.myservermanager.com/varakh/ecolinker/internal/server/model"
	"git.myservermanager.com/varakh/ecolinker/internal/server/service"
	"github.com/gin-gonic/gin"
)

type MqttSubscriptionHandler struct {
	mqttSubReadService  *service.MqttSubscriptionReadService
	mqttSubWriteService *service.MqttSubscriptionWriteService
}

func NewMqttSubscriptionHandler(r *service.MqttSubscriptionReadService, s *service.MqttSubscriptionWriteService) *MqttSubscriptionHandler {
	return &MqttSubscriptionHandler{
		mqttSubReadService:  r,
		mqttSubWriteService: s,
	}
}

func (h *MqttSubscriptionHandler) Get(c *gin.Context) {
	var err error

	var queryParams api.SNQueryParameterRequest
	if err = c.ShouldBindQuery(&queryParams); err != nil {
		AbortWithValidatorPayload(c, err)
		return
	}

	var mqttSubscriptions []*model.MqttSubscription
	if mqttSubscriptions, err = h.mqttSubReadService.Get(queryParams.SN); err != nil {
		_ = c.AbortWithError(ToHttpStatus(err), err)
		return
	}

	var data []*api.MqttSubscriptionResponse
	data = make([]*api.MqttSubscriptionResponse, 0, len(mqttSubscriptions))

	for _, e := range mqttSubscriptions {
		data = append(data, &api.MqttSubscriptionResponse{
			ID:        e.ID.String(),
			DeviceSN:  e.DeviceSN,
			TopicKind: e.TopicKind,
			CreatedAt: e.CreatedAt,
			UpdatedAt: e.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, api.NewDataResponseWithPayload(api.NewMqttSubscriptionPageResponse(data)))
}

func (h *MqttSubscriptionHandler) Create(c *gin.Context) {
	var err error

	var req api.CreateMqttSubscriptionRequest

	if err = c.ShouldBindJSON(&req); err != nil {
		AbortWithValidatorPayload(c, err)
		return
	}

	var e *model.MqttSubscription
	if e, err = h.mqttSubWriteService.Create(req.DeviceSN, constant.TopicKind(req.TopicKind)); err != nil {
		_ = c.AbortWithError(ToHttpStatus(err), err)
		return
	}

	c.JSON(http.StatusOK, api.NewMqttSubscriptionSingleResponse(e.ID.String(), e.DeviceSN, e.TopicKind, e.CreatedAt, e.UpdatedAt))
}

func (h *MqttSubscriptionHandler) Update(c *gin.Context) {
	var e *model.MqttSubscription
	var err error

	var req api.ModifyMqttSubscriptionRequest

	if err = c.ShouldBindJSON(&req); err != nil {
		AbortWithValidatorPayload(c, err)
		return
	}

	var pathParams api.IDUriRequest

	if err = c.ShouldBindUri(&pathParams); err != nil {
		AbortWithValidatorPayload(c, err)
		return
	}

	if e, err = h.mqttSubWriteService.Update(pathParams.ID, req.DeviceSN, constant.TopicKind(req.TopicKind)); err != nil {
		_ = c.AbortWithError(ToHttpStatus(err), err)
		return
	}

	c.JSON(http.StatusOK, api.NewMqttSubscriptionSingleResponse(e.ID.String(), e.DeviceSN, e.TopicKind, e.CreatedAt, e.UpdatedAt))
}

func (h *MqttSubscriptionHandler) Delete(c *gin.Context) {
	var err error
	var pathParams api.IDUriRequest

	if err = c.ShouldBindUri(&pathParams); err != nil {
		AbortWithValidatorPayload(c, err)
		return
	}

	if err = h.mqttSubWriteService.Delete(pathParams.ID); err != nil {
		_ = c.AbortWithError(ToHttpStatus(err), err)
		return
	}

	c.Header(httpcommons.HeaderContentType, httpcommons.HeaderContentTypeApplicationJson)
	c.Status(http.StatusNoContent)
}
