package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"git.myservermanager.com/varakh/ecolinker/internal/meta"
	"git.myservermanager.com/varakh/ecolinker/internal/server/config"
	"git.myservermanager.com/varakh/ecolinker/internal/server/constant"
	"git.myservermanager.com/varakh/ecolinker/internal/server/model"
	"git.myservermanager.com/varakh/ecolinker/internal/service_error"
	"git.myservermanager.com/varakh/go-ecoflow"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type EcoFlowMqttService struct {
	ecoFlowHttpService      *EcoFlowHttpService
	mqttSubRetrievalService *MqttSubscriptionReadService
	mqttForwardService      *MqttForwardService
	prometheusService       *PrometheusService
	taskService             *TaskService
	config                  *config.EcoFlow
	mqttClient              *ecoflow.MqttClient
	subscriptions           subscriptionsSet
}

type subscriptionsSet struct {
	sync.RWMutex
	values map[string]struct{}
}

const (
	jobNameEcoFlowMqttConnect = "ECOFLOW_MQTT_CONNECT"
	jobNameEcoFlowMqttSync    = "ECOFLOW_MQTT_SYNC"
)

var (
	ErrEcoFlowAlreadySubscribed = service_error.NewServiceError(service_error.ErrCodeConflict, errors.New("already subscribed to broker topic"))
	ErrEcoFlowNotSubscribed     = service_error.NewServiceError(service_error.ErrCodeConflict, errors.New("not subscribed to broker topic"))
)

func NewEcoFlowMqttService(ef *EcoFlowHttpService, mr *MqttSubscriptionReadService, mf *MqttForwardService, p *PrometheusService, t *TaskService, c *config.EcoFlow) *EcoFlowMqttService {
	return &EcoFlowMqttService{
		ecoFlowHttpService:      ef,
		mqttSubRetrievalService: mr,
		mqttForwardService:      mf,
		prometheusService:       p,
		taskService:             t,
		config:                  c,
	}
}

// Init initializes the service, bootstrapping necessary configuration, should be called directly after NewEcoFlowMqttService
func (s *EcoFlowMqttService) Init() error {
	s.subscriptions.values = make(map[string]struct{})

	if !s.config.MqttEnabled {
		return nil
	}

	var err error
	var mqttClient *ecoflow.MqttClient

	mqttClientConfig := ecoflow.MqttClientConfiguration{
		OnConnect: func(client mqtt.Client) {
			optionsReader := client.OptionsReader()
			log.Info().Msgf("Connected to broker: %s", optionsReader.Servers()[0].String())
			s.SyncSubscriptions()
		},
		OnConnectionLost: func(client mqtt.Client, err error) {
			optionsReader := client.OptionsReader()
			log.Warn().Msgf("Connection to broker '%s' lost: %v", optionsReader.Servers()[0].String(), err)
			s.ClearSubscriptions()
		},
		OnReconnect: func(client mqtt.Client, options *mqtt.ClientOptions) {
			optionsReader := client.OptionsReader()
			log.Info().Msgf("Reconnecting to broker: %s", optionsReader.Servers()[0].String())
			if _, connected := s.Status(); !connected {
				log.Warn().Msg("Not connected yet")
				return
			}

			log.Info().Msgf("Reconnected to broker: %s", optionsReader.Servers()[0].String())
		},
		MaxReconnectInterval: s.config.MqttMaxReconnectInterval,
	}

	var mqttCredentialsResponse *ecoflow.MqttCredentialsResponse
	if mqttCredentialsResponse, err = s.ecoFlowHttpService.httpClient.GetOpenSignCertification(context.Background()); err != nil {
		return fmt.Errorf("unable to retrieve required MQTT credentials: %w", err)
	}

	mqttConnectionConfig := &ecoflow.MqttConnectionConfig{
		CertificateAccount:  mqttCredentialsResponse.Data.CertificateAccount,
		CertificatePassword: mqttCredentialsResponse.Data.CertificatePassword,
		Url:                 mqttCredentialsResponse.Data.Url,
		Port:                mqttCredentialsResponse.Data.Port,
		Protocol:            mqttCredentialsResponse.Data.Protocol,
		ClientId:            fmt.Sprintf("%s_%s", strings.ToUpper(meta.Name), uuid.New()),
	}

	if mqttClient, err = ecoflow.NewOpenMqttClient(mqttConnectionConfig, mqttClientConfig); err != nil {
		return fmt.Errorf("unable to create MQTT client: %w", err)
	}

	s.mqttClient = mqttClient

	runnable := func() {
		if conErr := s.Connect(); conErr != nil {
			log.Error().Msgf("Unable to connect to EcoFlow MQTT: %v", conErr)
		}
	}

	if _, err = s.taskService.EnqueueOnce(gocron.OneTimeJob(gocron.OneTimeJobStartImmediately()), gocron.NewTask(runnable), jobNameEcoFlowMqttConnect, gocron.WithDisabledDistributedJobLocker(true)); err != nil {
		return err
	}

	return nil
}

func (s *EcoFlowMqttService) SyncSubscriptions() {
	runnable := func() {
		var err error
		var subs []*model.MqttSubscription
		if subs, err = s.mqttSubRetrievalService.GetAll(); err != nil {
			log.Error().Msgf("Cannot synchronize MQTT subscriptions, retrieval failed: %v", err)
			return
		}

		for _, sub := range subs {
			switch sub.TopicKind {
			case constant.TopicKindStatus.String():
			case constant.TopicKindQuota.String():
				messageHandler := NewEcoFlowMqttMessageHandler(sub.DeviceSN, constant.TopicKind(sub.TopicKind), s.config.MqttDebugMessages, s.prometheusService, s.mqttForwardService)
				if err = s.Subscribe(messageHandler); err != nil {
					log.Error().Msgf("Device '%s' unable to subscribe to topic '%s' of EcoFlow MQTT: %v", sub.DeviceSN, sub.TopicKind, err)
					continue
				}
			}
		}
	}

	if _, enqueueErr := s.taskService.EnqueueOnce(gocron.OneTimeJob(gocron.OneTimeJobStartImmediately()), gocron.NewTask(runnable), jobNameEcoFlowMqttSync, gocron.WithDisabledDistributedJobLocker(false)); enqueueErr != nil {
		log.Error().Msgf("Unable to enqueue task: %v", enqueueErr)
	}
}

// Status returns broker status
func (s *EcoFlowMqttService) Status() (bool, bool) {
	return s.config.MqttEnabled, s.mqttClient != nil && s.mqttClient.Client.IsConnected() && s.mqttClient.Client.IsConnectionOpen()
}

// Subscribe adds a subscription
func (s *EcoFlowMqttService) Subscribe(handler *EcoFlowMqttMessageHandler) error {
	if !s.config.MqttEnabled {
		return nil
	}
	if _, connected := s.Status(); !connected {
		log.Warn().Msg("Not connected")
		return nil
	}

	s.subscriptions.Lock()
	defer s.subscriptions.Unlock()

	topicName := s.topicNameFrom(handler.DeviceSN, handler.TopicKind)

	if _, ok := s.subscriptions.values[topicName]; ok {
		log.Error().Msgf("Cannot subscribe to topic '%s': %v", topicName, ErrEcoFlowAlreadySubscribed)
		return ErrEcoFlowAlreadySubscribed
	}

	if err := s.mqttClient.SubscribeToTopics([]string{topicName}, handler.HandleMessage); err != nil {
		log.Error().Msgf("Cannot subscribe to topic '%s': %v", topicName, err)
		return service_error.NewServiceError(service_error.ErrCodeGeneral, fmt.Errorf("cannot subscribe to topic '%s': %w", topicName, err))
	}

	s.subscriptions.values[topicName] = struct{}{}

	log.Info().Msgf("Subscribed to topic '%s'", topicName)
	return nil
}

// Unsubscribe deletes a subscription
func (s *EcoFlowMqttService) Unsubscribe(deviceSN string, topicKind constant.TopicKind) error {
	if !s.config.MqttEnabled {
		return nil
	}
	if _, connected := s.Status(); !connected {
		log.Warn().Msg("Not connected")
		return nil
	}

	s.subscriptions.Lock()
	defer s.subscriptions.Unlock()

	topicName := s.topicNameFrom(deviceSN, topicKind)

	if _, ok := s.subscriptions.values[topicName]; !ok {
		return ErrEcoFlowNotSubscribed
	}

	if err := s.mqttClient.UnsubscribeFromTopics([]string{topicName}); err != nil {
		log.Error().Msgf("Cannot unsubscribe from topic '%s': %v", topicName, err)
		return service_error.NewServiceError(service_error.ErrCodeGeneral, fmt.Errorf("cannot unsubscribe from topic '%s': %w", topicName, err))
	}

	delete(s.subscriptions.values, topicName)

	log.Info().Msgf("Unsubscribed from topic '%s'", topicName)
	return nil
}

// ClearSubscriptions clears subscriptions
func (s *EcoFlowMqttService) ClearSubscriptions() {
	s.subscriptions.Lock()
	defer s.subscriptions.Unlock()

	clear(s.subscriptions.values)
	log.Debug().Msgf("Cleared subscriptions")
}

// Connect connects to MQTT
// If already connected, skips
func (s *EcoFlowMqttService) Connect() error {
	if !s.config.MqttEnabled {
		return nil
	}
	if _, connected := s.Status(); connected {
		log.Warn().Msg("Already connected")
		return nil
	}

	var err error
	if err = s.mqttClient.Connect(); err != nil {
		return fmt.Errorf("unable to connect to MQTT broker: %w", err)
	}

	return nil
}

// Disconnect disconnects from MQTT
// If not connected, skips
func (s *EcoFlowMqttService) Disconnect() {
	if !s.config.MqttEnabled {
		return
	}
	if _, connected := s.Status(); !connected {
		log.Warn().Msg("Not connected")
		return
	}

	wait := s.config.MqttWaitDisconnect
	log.Info().Msgf("Disconnecting from broker, waiting at maximum %dms", wait)
	s.mqttClient.Disconnect(wait)
	log.Info().Msg("Disconnected from broker")
}

// topicNameFrom constructs a new topic name given the device's serial number, the kind of the topic
func (s *EcoFlowMqttService) topicNameFrom(sn string, topicKind constant.TopicKind) string {
	return fmt.Sprintf("/open/%s/%s/%s", s.mqttClient.ConnectionConfig.CertificateAccount, sn, strings.ToLower(topicKind.String()))
}
