package service

import (
	"git.myservermanager.com/varakh/ecolinker/internal/server/config"
	"git.myservermanager.com/varakh/ecolinker/internal/server/constant"
	"github.com/go-co-op/gocron/v2"
	"github.com/rs/zerolog/log"
)

type EcoFlowMqttTask struct {
	ecoFlowMqttService *EcoFlowMqttService
	mqttForwardService *MqttForwardService
	prometheusService  *PrometheusService
	taskService        *TaskService
	ecoFlowConfig      *config.EcoFlow
}

const (
	jobNameEcoFlowMqttSubscribe   = "ECOFLOW_MQTT_SUBSCRIBE"
	jobNameEcoFlowMqttUnsubscribe = "ECOFLOW_MQTT_UNSUBSCRIBE"
)

func NewEcoFlowMqttTask(em *EcoFlowMqttService, mf *MqttForwardService, p *PrometheusService, t *TaskService, efc *config.EcoFlow) *EcoFlowMqttTask {
	return &EcoFlowMqttTask{
		ecoFlowMqttService: em,
		mqttForwardService: mf,
		prometheusService:  p,
		taskService:        t,
		ecoFlowConfig:      efc,
	}
}

func (s *EcoFlowMqttTask) Subscribe(deviceSN string, topicKind constant.TopicKind) {
	if deviceSN == "" || topicKind == "" {
		return
	}

	runnable := func() {
		var err error

		switch topicKind {
		case constant.TopicKindStatus:
		case constant.TopicKindQuota:
			messageHandler := NewEcoFlowMqttMessageHandler(deviceSN, topicKind, s.ecoFlowConfig.MqttDebugMessages, s.prometheusService, s.mqttForwardService)
			if err = s.ecoFlowMqttService.Subscribe(messageHandler); err != nil {
				log.Error().Err(err).Msgf("Device '%s' unable to subscribe to topic '%s' of EcoFlow MQTT", deviceSN, topicKind.String())
				return
			}
		}
	}

	if _, enqueueErr := s.taskService.EnqueueOnce(gocron.OneTimeJob(gocron.OneTimeJobStartImmediately()), gocron.NewTask(runnable), jobNameEcoFlowMqttSubscribe, gocron.WithDisabledDistributedJobLocker(true)); enqueueErr != nil {
		log.Error().Err(enqueueErr).Msg("Unable to enqueue task")
	}
}

func (s *EcoFlowMqttTask) Unsubscribe(deviceSN string, topicKind constant.TopicKind) {
	if deviceSN == "" || topicKind == "" {
		return
	}

	runnable := func() {
		if err := s.ecoFlowMqttService.Unsubscribe(deviceSN, topicKind); err != nil {
			log.Error().Err(err).Msgf("Device '%s' unable to unsubscribe from topic '%s' of EcoFlow MQTT", deviceSN, topicKind.String())
			return
		}
	}

	if _, enqueueErr := s.taskService.EnqueueOnce(gocron.OneTimeJob(gocron.OneTimeJobStartImmediately()), gocron.NewTask(runnable), jobNameEcoFlowMqttUnsubscribe, gocron.WithDisabledDistributedJobLocker(true)); enqueueErr != nil {
		log.Error().Err(enqueueErr).Msg("Unable to enqueue task")
	}
}
