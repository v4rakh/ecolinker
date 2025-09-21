package service

import (
	"git.myservermanager.com/varakh/ecolinker/internal/float"
	"git.myservermanager.com/varakh/ecolinker/internal/server/config"
	"git.myservermanager.com/varakh/ecolinker/internal/server/constant"
	"github.com/go-co-op/gocron/v2"
	"github.com/rs/zerolog/log"
)

type PrometheusTask struct {
	prometheusConfig   *config.Prometheus
	ecoFlowMqttService *EcoFlowMqttService
	mqttForwardService *MqttForwardService
	prometheusService  *PrometheusService
	taskService        *TaskService
}

const (
	jobNamePrometheusRefresh = "PROMETHEUS_REFRESH"
)

func NewPrometheusTask(c *config.Prometheus, em *EcoFlowMqttService, mf *MqttForwardService, p *PrometheusService, t *TaskService) *PrometheusTask {
	return &PrometheusTask{
		prometheusConfig:   c,
		ecoFlowMqttService: em,
		mqttForwardService: mf,
		prometheusService:  p,
		taskService:        t,
	}
}

// Init initializes background tasks for the service, should be called directly after NewPrometheusTask
func (s *PrometheusTask) Init() error {
	return s.configurePrometheusRefreshTask()
}

func (s *PrometheusTask) configurePrometheusRefreshTask() error {
	if !s.prometheusConfig.Enabled {
		return nil
	}

	if err := s.prometheusService.RegisterGaugeNoLabels(constant.MetricEcoFlowMqttEnabled, constant.MetricEcoFlowMqttEnabledHelp); err != nil {
		return err
	}
	if err := s.prometheusService.RegisterGaugeNoLabels(constant.MetricEcoFlowMqttConnected, constant.MetricEcoFlowMqttConnectedHelp); err != nil {
		return err
	}
	if err := s.prometheusService.RegisterGaugeNoLabels(constant.MetricMqttForwardEnabled, constant.MetricMqttForwardEnabledHelp); err != nil {
		return err
	}
	if err := s.prometheusService.RegisterGaugeNoLabels(constant.MetricMqttForwardConnected, constant.MetricMqttForwardConnectedHelp); err != nil {
		return err
	}

	runnable := func() {
		enabledEcoFlow, connectedEcoFlow := s.ecoFlowMqttService.Status()
		if err := s.prometheusService.SetGaugeNoLabels(constant.MetricEcoFlowMqttEnabled, float.BoolToFloat(enabledEcoFlow)); err != nil {
			log.Error().Msgf("Could not refresh EcoFlow MQTT enabled status. Reason: %s", err.Error())
		}
		if err := s.prometheusService.SetGaugeNoLabels(constant.MetricEcoFlowMqttConnected, float.BoolToFloat(connectedEcoFlow)); err != nil {
			log.Error().Msgf("Could not refresh EcoFlow MQTT connected status. Reason: %s", err.Error())
		}

		enabledMqttForward, connectedMqttForward := s.mqttForwardService.Status()
		if err := s.prometheusService.SetGaugeNoLabels(constant.MetricMqttForwardEnabled, float.BoolToFloat(enabledMqttForward)); err != nil {
			log.Error().Msgf("Could not refresh MQTT forward enabled status. Reason: %s", err.Error())
		}
		if err := s.prometheusService.SetGaugeNoLabels(constant.MetricMqttForwardConnected, float.BoolToFloat(connectedMqttForward)); err != nil {
			log.Error().Msgf("Could not refresh MQTT forward connected status. Reason: %s", err.Error())
		}
	}

	_, err := s.taskService.Enqueue(gocron.DurationJob(s.prometheusConfig.RefreshInterval), gocron.NewTask(runnable), jobNamePrometheusRefresh)
	return err
}
