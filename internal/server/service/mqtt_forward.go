package service

import (
	"fmt"
	"git.myservermanager.com/varakh/ecolinker/internal/app"
	"git.myservermanager.com/varakh/ecolinker/internal/server/config"
	"git.myservermanager.com/varakh/ecolinker/internal/server/service_error"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"strings"
)

type MqttForwardService struct {
	taskService *TaskService
	config      *config.MqttForward
	mqttClient  mqtt.Client
}

const (
	jobNameForwardMqttConnect = "FORWARD_MQTT_CONNECT"
)

func NewMqttForwardService(t *TaskService, c *config.MqttForward) *MqttForwardService {
	return &MqttForwardService{
		taskService: t,
		config:      c,
	}
}

// Init initializes the service, bootstrapping necessary metrics, should be called directly after NewMqttForwardService
func (s *MqttForwardService) Init() error {
	if !s.config.Enabled {
		return nil
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("%s://%s:%d", s.config.Protocol, s.config.Host, s.config.Port))
	opts.SetClientID(fmt.Sprintf("%s_%s", strings.ToUpper(app.Name), uuid.New()))

	if s.config.Username != "" && s.config.Password != "" {
		opts.SetUsername(s.config.Username)
		opts.SetPassword(s.config.Password)
	}

	if s.config.MaxReconnectInterval != 0 {
		opts.MaxReconnectInterval = s.config.MaxReconnectInterval
	}

	opts.SetConnectRetry(true)

	opts.OnConnect = func(client mqtt.Client) {
		optionsReader := client.OptionsReader()
		zap.L().Sugar().Infof("Connected to broker: %s", optionsReader.Servers()[0].String())
	}
	opts.OnConnectionLost = func(client mqtt.Client, err error) {
		optionsReader := client.OptionsReader()
		zap.L().Sugar().Warnf("Connection to broker '%s' lost: %v", optionsReader.Servers()[0].String(), err)
	}
	opts.OnReconnecting = func(client mqtt.Client, options *mqtt.ClientOptions) {
		optionsReader := client.OptionsReader()
		zap.L().Sugar().Infof("Reconnected to broker: %s", optionsReader.Servers()[0].String())
	}

	s.mqttClient = mqtt.NewClient(opts)

	runnable := func() {
		if err := s.Connect(); err != nil {
			zap.L().Sugar().Errorf("Unable to connect to forward MQTT: %v", err)
		}
	}

	var err error
	if _, err = s.taskService.EnqueueOnce(gocron.OneTimeJob(gocron.OneTimeJobStartImmediately()), gocron.NewTask(runnable), jobNameForwardMqttConnect,
		gocron.WithDisabledDistributedJobLocker(true)); err != nil {
		return err
	}

	return nil
}

// Status returns broker status
func (s *MqttForwardService) Status() (bool, bool) {
	return s.config.Enabled, s.mqttClient != nil && s.mqttClient.IsConnected() && s.mqttClient.IsConnectionOpen()
}

// Connect connects to MQTT
// If already connected, skips
func (s *MqttForwardService) Connect() error {
	if !s.config.Enabled {
		return nil
	}
	if _, connected := s.Status(); connected {
		zap.L().Warn("Already connected")
		return nil
	}

	if token := s.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		return service_error.NewServiceError(service_error.ErrCodeGeneral, token.Error())
	}
	return nil
}

// Publish publishes to a topic
// If not connected, skips
func (s *MqttForwardService) Publish(topic string, qos byte, retained bool, payload interface{}) error {
	if topic == "" {
		return service_error.ErrValidationNotBlank
	}
	if payload == nil {
		return service_error.ErrValidationNotEmpty
	}

	if !s.config.Enabled {
		return nil
	}
	if _, connected := s.Status(); !connected {
		zap.L().Warn("Not connected")
		return nil
	}

	if token := s.mqttClient.Publish(topic, qos, retained, payload); token.Wait() && token.Error() != nil {
		return service_error.NewServiceError(service_error.ErrCodeGeneral, fmt.Errorf("could not publish to '%s': %w", topic, token.Error()))
	}

	zap.L().Sugar().Debugf("Published to topic '%s'", topic)

	return nil
}

// Disconnect disconnects from MQTT
// If not connected, skips
func (s *MqttForwardService) Disconnect() {
	if !s.config.Enabled {
		return
	}
	if _, connected := s.Status(); !connected {
		zap.L().Warn("Not connected")
		return
	}

	wait := s.config.WaitDisconnect
	zap.L().Sugar().Infof("Disconnecting from broker, waiting at maximum %dms", wait)
	s.mqttClient.Disconnect(wait)
	zap.L().Info("Disconnected from broker")
}
