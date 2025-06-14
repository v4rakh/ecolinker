package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"git.myservermanager.com/varakh/ecolinker/internal/app"
	"git.myservermanager.com/varakh/ecolinker/internal/float"
	"git.myservermanager.com/varakh/ecolinker/internal/server/constant"
	"git.myservermanager.com/varakh/go-ecoflow"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	metricHelp = "Consult documentation of your device"
)

type EcoFlowMqttMessageHandler struct {
	DeviceSN           string
	TopicKind          constant.TopicKind
	DebugMessages      bool
	prometheusService  *PrometheusService
	mqttForwardService *MqttForwardService
}

func NewEcoFlowMqttMessageHandler(deviceSN string, topicKind constant.TopicKind, debugMessages bool, p *PrometheusService, m *MqttForwardService) *EcoFlowMqttMessageHandler {
	return &EcoFlowMqttMessageHandler{
		DeviceSN:           deviceSN,
		TopicKind:          topicKind,
		DebugMessages:      debugMessages,
		prometheusService:  p,
		mqttForwardService: m,
	}
}

// HandleMessage handles quota EcoFlow mqtt messages
func (h *EcoFlowMqttMessageHandler) HandleMessage(_ mqtt.Client, message mqtt.Message) {
	go func() {
		h.processMessage(message)
	}()
}

// processMessage processes the message (should be called in another goroutine if heavy lifting is done)
func (h *EcoFlowMqttMessageHandler) processMessage(message mqtt.Message) {
	received := float64(time.Now().Unix())

	generalMetricLabels := []string{"device", "topicKind"}
	genericMetricLabelValues := []string{h.DeviceSN, h.TopicKind.String()}

	if promErr := h.prometheusService.RegisterCounter(constant.MetricEcoFlowMqttMessagesReceived, constant.MetricEcoFlowMqttMessagesReceivedHelp, generalMetricLabels); promErr != nil {
		if !errors.Is(promErr, ErrPrometheusMetricAlreadyRegistered) {
			zap.L().Sugar().Warnf("Unable to register prometheus metric for '%s': %v", constant.MetricEcoFlowMqttMessagesReceived, promErr)
		}
	}

	if promErr := h.prometheusService.IncreaseCounter(constant.MetricEcoFlowMqttMessagesReceived, genericMetricLabelValues); promErr != nil {
		zap.L().Sugar().Warnf("Unable to set prometheus metric for '%s': %v", constant.MetricEcoFlowMqttMessagesReceived, promErr)
	}

	if promErr := h.prometheusService.RegisterGauge(constant.MetricEcoFlowMqttMessageLastReceived, constant.MetricEcoFlowMqttMessageLastReceivedHelp, generalMetricLabels); promErr != nil {
		if !errors.Is(promErr, ErrPrometheusMetricAlreadyRegistered) {
			zap.L().Sugar().Warnf("Unable to register prometheus metric for '%s': %v", constant.MetricEcoFlowMqttMessageLastReceived, promErr)
		}
	}
	if promErr := h.prometheusService.SetGauge(constant.MetricEcoFlowMqttMessageLastReceived, genericMetricLabelValues, received); promErr != nil {
		zap.L().Sugar().Warnf("Unable to set prometheus metric for '%s': %v", constant.MetricEcoFlowMqttMessageLastReceived, promErr)
	}

	var err error
	var data ecoflow.MqttOpenMessage

	if err = json.Unmarshal(message.Payload(), &data); err != nil {
		zap.L().Sugar().Errorf("Unable to parse message from topic '%s': %v", message.Topic(), err)
		return
	}

	if h.DebugMessages {
		zap.L().Sugar().Debugf("Message from topic '%s': %+v", message.Topic(), data)
		var b []byte
		if b, err = json.MarshalIndent(data, "", "  "); err != nil {
			zap.L().Sugar().Warnf("Unable to marshal message: %v", err)
		} else {
			zap.L().Sugar().Debugf("Parsed message payload: %s", string(b))
		}
	}

	forwardTopic := fmt.Sprintf("/%s/%s/%s", strings.ToLower(app.Name), h.DeviceSN, strings.ToLower(h.TopicKind.String()))
	if err = h.mqttForwardService.Publish(forwardTopic, message.Qos(), message.Retained(), message.Payload()); err != nil {
		zap.L().Sugar().Warnf("Unable to forward message: %v", err)
	}

	if data.Param == nil || data.Param != nil && len(data.Param) == 0 {
		return
	}

	if promErr := h.prometheusService.RegisterGauge(constant.MetricEcoFlowMqttMessageLastReceivedWithPayload, constant.MetricEcoFlowMqttMessageLastReceivedWithPayloadHelp, generalMetricLabels); promErr != nil {
		if !errors.Is(promErr, ErrPrometheusMetricAlreadyRegistered) {
			zap.L().Sugar().Warnf("Unable to register prometheus metric for '%s': %v", constant.MetricEcoFlowMqttMessageLastReceived, promErr)
		}
	}
	if promErr := h.prometheusService.SetGauge(constant.MetricEcoFlowMqttMessageLastReceivedWithPayload, genericMetricLabelValues, received); promErr != nil {
		zap.L().Sugar().Warnf("Unable to set prometheus metric for '%s': %v", constant.MetricEcoFlowMqttMessageLastReceivedWithPayloadHelp, promErr)
	}

	flattenedPayload := make(map[string]interface{})
	flatten(data.Param, flattenedPayload)

	if h.DebugMessages {
		var b []byte
		if b, err = json.MarshalIndent(flattenedPayload, "", "  "); err != nil {
			zap.L().Sugar().Warnf("Unable to marshal message: %v", err)
		} else {
			zap.L().Sugar().Debugf("Flattened message param payload: %s", string(b))
		}
	}

	extracted := extractIndicesAndValueList(flattenedPayload)

	for _, valueMap := range extracted {
		metricValue, ok := float.ToFloat(valueMap.Value)

		if !ok {
			zap.L().Sugar().Warnf("Unable to cast value to prometheus metric type: %v", metricValue)
			continue
		}

		metricKey := fmt.Sprintf("%s_%s", strings.ToLower(app.Name), valueMap.Key)

		metricLabelKeys := []string{"device"}
		metricLabelKeys = append(metricLabelKeys, slices.Collect(maps.Keys(valueMap.Indices))...)

		metricLabelValues := []string{h.DeviceSN}

		indicesValues := slices.Collect(maps.Values(valueMap.Indices))
		for _, v := range indicesValues {
			metricLabelValues = append(metricLabelValues, strconv.Itoa(v))
		}

		if promErr := h.prometheusService.RegisterGauge(metricKey, metricHelp, metricLabelKeys); promErr != nil {
			if !errors.Is(promErr, ErrPrometheusMetricAlreadyRegistered) {
				zap.L().Sugar().Warnf("Unable to register prometheus metric for '%s': %v", valueMap.Key, promErr)
				continue
			}
		}
		if promErr := h.prometheusService.SetGauge(metricKey, metricLabelValues, metricValue); promErr != nil {
			zap.L().Sugar().Warnf("Unable to set prometheus metric for '%s': %v", valueMap.Key, promErr)
			continue
		}
	}
}
