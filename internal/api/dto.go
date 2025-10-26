package api

import "time"

// Requests

// json/body

type EcoFlowDeviceParametersRequest struct {
	Parameters []string `json:"parameters" binding:"required,min=1"`
}

type CreateDeviceRequest struct {
	SN    string `json:"sn" binding:"required,min=1,max=255"`
	Label string `json:"label" binding:"required,min=1,max=255"`
	Kind  string `json:"kind" binding:"required,oneof=other powerocean"`
}

type ModifyDeviceRequest struct {
	SN    string `json:"sn" binding:"required,min=1,max=255"`
	Label string `json:"label" binding:"required,min=1,max=255"`
	Kind  string `json:"kind" binding:"required,oneof=other powerocean"`
}

type CreateMqttSubscriptionRequest struct {
	DeviceSN  string `json:"deviceSN" binding:"required,min=1"`
	TopicKind string `json:"topicKind" binding:"required,oneof=quota status"`
}

type ModifyMqttSubscriptionRequest struct {
	DeviceSN  string `json:"deviceSN" binding:"required,min=1"`
	TopicKind string `json:"topicKind" binding:"required,oneof=quota status"`
}

type CreateCollectorRequest struct {
	DeviceSN  string                 `json:"deviceSN" binding:"required,min=1"`
	Kind      string                 `json:"kind" binding:"required,oneof=device_parameters device_historical_data"`
	Frequency string                 `json:"frequency" binding:"required"`
	Payload   map[string]interface{} `json:"payload"`
}

type ModifyCollectorRequest struct {
	DeviceSN  string                 `json:"deviceSN" binding:"required,min=1"`
	Kind      string                 `json:"kind" binding:"required,oneof=device_parameters device_historical_data"`
	Frequency string                 `json:"frequency" binding:"required"`
	Payload   map[string]interface{} `json:"payload"`
}

// uri parameters

type SNUriRequest struct {
	SN string `uri:"sn" binding:"required,min=1"`
}

type IDUriRequest struct {
	ID string `uri:"id" binding:"required,uuid4"`
}

// query parameters

type SNQueryParameterRequest struct {
	SN string `form:"sn"`
}

type HistoryQueryParameterRequest struct {
	BeginTime string `form:"beginTime" binding:"required"`
	EndTime   string `form:"endTime" binding:"required"`
}

// Responses

type Response struct {
}

type DataResponse struct {
	Response
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type ErrorResponse struct {
	Status string `json:"status,omitempty"`
	DataResponse
}

func NewDataResponseWithPayload(payload interface{}) *DataResponse {
	e := new(DataResponse)
	e.Data = payload
	return e
}

func NewErrorResponseWithStatusAndMessage(status string, message string) *ErrorResponse {
	e := new(ErrorResponse)
	e.Status = status
	e.Message = message
	return e
}

type HealthResponse struct {
	Healthy bool `json:"healthy"`
}

func NewHealthResponse(b bool) *HealthResponse {
	r := new(HealthResponse)
	r.Healthy = b
	return r
}

type InfoResponse struct {
	Version  string `json:"version"`
	Name     string `json:"name"`
	TimeZone string `json:"timeZone"`
}

func NewInfoResponse(name string, version string, tz string) *InfoResponse {
	r := new(InfoResponse)
	r.Name = name
	r.Version = version
	r.TimeZone = tz
	return r
}

type DeviceResponse struct {
	SN        string    `json:"sn"`
	Kind      string    `json:"kind"`
	Label     string    `json:"label"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type DeviceSingleResponse struct {
	Data DeviceResponse `json:"data"`
}

func NewDeviceSingleResponse(sn string, kind string, label string, createdAt time.Time, updatedAt time.Time) *DeviceSingleResponse {
	e := new(DeviceSingleResponse)
	e.Data.SN = sn
	e.Data.Kind = kind
	e.Data.Label = label
	e.Data.CreatedAt = createdAt
	e.Data.UpdatedAt = updatedAt
	return e
}

type DevicePageResponse struct {
	Content []*DeviceResponse `json:"content"`
}

type DevicePageDataResponse struct {
	Data DevicePageResponse `json:"data"`
}

func NewDevicePageResponse(content []*DeviceResponse) *DevicePageResponse {
	e := new(DevicePageResponse)
	e.Content = content
	return e
}

type MqttSubscriptionResponse struct {
	ID        string    `json:"id"`
	TopicKind string    `json:"topicKind"`
	DeviceSN  string    `json:"deviceSN"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type MqttSubscriptionSingleResponse struct {
	Data MqttSubscriptionResponse `json:"data"`
}

func NewMqttSubscriptionSingleResponse(id string, deviceSN string, topicKind string, createdAt time.Time, updatedAt time.Time) *MqttSubscriptionSingleResponse {
	e := new(MqttSubscriptionSingleResponse)
	e.Data.ID = id
	e.Data.DeviceSN = deviceSN
	e.Data.TopicKind = topicKind
	e.Data.CreatedAt = createdAt
	e.Data.UpdatedAt = updatedAt
	return e
}

type MqttSubscriptionPageResponse struct {
	Content []*MqttSubscriptionResponse `json:"content"`
}

type MqttSubscriptionPageDataResponse struct {
	Data *MqttSubscriptionPageResponse `json:"data"`
}

func NewMqttSubscriptionPageResponse(content []*MqttSubscriptionResponse) *MqttSubscriptionPageResponse {
	e := new(MqttSubscriptionPageResponse)
	e.Content = content
	return e
}

type CollectorResponse struct {
	ID        string      `json:"id"`
	Kind      string      `json:"kind"`
	DeviceSN  string      `json:"deviceSN"`
	Frequency string      `json:"frequency"`
	Payload   interface{} `json:"payload"`
	CreatedAt time.Time   `json:"createdAt"`
	UpdatedAt time.Time   `json:"updatedAt"`
}

type CollectorSingleResponse struct {
	Data CollectorResponse `json:"data"`
}

func NewCollectorSingleResponse(id string, deviceSN string, kind string, frequency string, payload interface{}, createdAt time.Time, updatedAt time.Time) *CollectorSingleResponse {
	e := new(CollectorSingleResponse)
	e.Data.ID = id
	e.Data.DeviceSN = deviceSN
	e.Data.Kind = kind
	e.Data.Frequency = frequency
	e.Data.Payload = payload
	e.Data.CreatedAt = createdAt
	e.Data.UpdatedAt = updatedAt
	return e
}

type CollectorPageResponse struct {
	Content []*CollectorResponse `json:"content"`
}

type CollectorPageDataResponse struct {
	Data *CollectorPageResponse `json:"data"`
}

func NewCollectorPageResponse(content []*CollectorResponse) *CollectorPageResponse {
	e := new(CollectorPageResponse)
	e.Content = content
	return e
}

type EcoFlowBrokerStatusResponse struct {
	Enabled   bool `json:"enabled"`
	Connected bool `json:"connected"`
}

func NewEcoFlowBrokerStatusResponse(enabled bool, connected bool) *EcoFlowBrokerStatusResponse {
	r := new(EcoFlowBrokerStatusResponse)
	r.Enabled = enabled
	r.Connected = connected
	return r
}

type EcoFlowBrokerStatusDataResponse struct {
	Data *EcoFlowBrokerStatusResponse `json:"data"`
}

type EcoFlowDeviceResponse struct {
	SN     string `json:"sn"`
	Online int    `json:"online"`
}

type EcoFlowDeviceListResponse struct {
	Content []*EcoFlowDeviceResponse `json:"content"`
}

func NewEcoFlowDeviceListResponse(content []*EcoFlowDeviceResponse) *EcoFlowDeviceListResponse {
	e := new(EcoFlowDeviceListResponse)
	e.Content = content
	return e
}

type EcoFlowDeviceListDataResponse struct {
	Data *EcoFlowDeviceListResponse `json:"data"`
}

type EcoFlowDeviceParametersDataResponse struct {
	Data map[string]interface{} `json:"data"`
}

type EcoFlowDeviceBatteriesDataResponse struct {
	Data map[string]map[string]interface{} `json:"data"`
}
type EcoFlowHistoryItemResponse struct {
	IndexName  string   `json:"indexName"`
	IndexValue *float64 `json:"indexValue,omitempty"`
	Unit       string   `json:"unit"`
}

type EcoFlowHistoryDataResponse struct {
	Data []*EcoFlowHistoryItemResponse `json:"data"`
}
