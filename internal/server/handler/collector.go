package handler

import (
	"git.myservermanager.com/varakh/ecolinker/api"
	"git.myservermanager.com/varakh/ecolinker/internal/server/constant"
	"git.myservermanager.com/varakh/ecolinker/internal/server/model"
	"git.myservermanager.com/varakh/ecolinker/internal/server/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

type CollectorHandler struct {
	collectorService *service.CollectorService
}

func NewCollectorHandler(s *service.CollectorService) *CollectorHandler {
	return &CollectorHandler{
		collectorService: s,
	}
}

func (h *CollectorHandler) Get(c *gin.Context) {
	var err error

	var queryParams api.SNQueryParameterRequest
	if err = c.ShouldBindQuery(&queryParams); err != nil {
		AbortWithValidatorPayload(c, err)
		return
	}

	var Collectors []*model.Collector
	if Collectors, err = h.collectorService.Get(queryParams.SN); err != nil {
		_ = c.AbortWithError(ToHttpStatus(err), err)
		return
	}

	var data []*api.CollectorResponse
	data = make([]*api.CollectorResponse, 0, len(Collectors))

	for _, e := range Collectors {
		data = append(data, &api.CollectorResponse{
			ID:        e.ID.String(),
			DeviceSN:  e.DeviceSN,
			Kind:      e.Kind,
			Frequency: e.Frequency,
			Payload:   e.Payload,
			CreatedAt: e.CreatedAt,
			UpdatedAt: e.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, api.NewDataResponseWithPayload(api.NewCollectorPageResponse(data)))
}

func (h *CollectorHandler) Create(c *gin.Context) {
	var err error

	var req api.CreateCollectorRequest

	if err = c.ShouldBindJSON(&req); err != nil {
		AbortWithValidatorPayload(c, err)
		return
	}

	var e *model.Collector
	if e, err = h.collectorService.Create(req.DeviceSN, constant.CollectorKind(req.Kind), req.Frequency, req.Payload); err != nil {
		_ = c.AbortWithError(ToHttpStatus(err), err)
		return
	}

	c.JSON(http.StatusOK, api.NewCollectorSingleResponse(e.ID.String(), e.DeviceSN, e.Kind, e.Frequency, e.Payload, e.CreatedAt, e.UpdatedAt))
}

func (h *CollectorHandler) Update(c *gin.Context) {
	var e *model.Collector
	var err error

	var req api.ModifyCollectorRequest

	if err = c.ShouldBindJSON(&req); err != nil {
		AbortWithValidatorPayload(c, err)
		return
	}

	var pathParams api.IDUriRequest

	if err = c.ShouldBindUri(&pathParams); err != nil {
		AbortWithValidatorPayload(c, err)
		return
	}

	if e, err = h.collectorService.Update(pathParams.ID, req.DeviceSN, constant.CollectorKind(req.Kind), req.Frequency, req.Payload); err != nil {
		_ = c.AbortWithError(ToHttpStatus(err), err)
		return
	}

	c.JSON(http.StatusOK, api.NewCollectorSingleResponse(e.ID.String(), e.DeviceSN, e.Kind, e.Frequency, e.Payload, e.CreatedAt, e.UpdatedAt))
}

func (h *CollectorHandler) Delete(c *gin.Context) {
	var err error
	var pathParams api.IDUriRequest

	if err = c.ShouldBindUri(&pathParams); err != nil {
		AbortWithValidatorPayload(c, err)
		return
	}

	if err = h.collectorService.Delete(pathParams.ID); err != nil {
		_ = c.AbortWithError(ToHttpStatus(err), err)
		return
	}

	c.Header(api.HeaderContentType, api.HeaderContentTypeApplicationJson)
	c.Status(http.StatusNoContent)
}
