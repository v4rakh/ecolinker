package service

import (
	"git.myservermanager.com/varakh/ecolinker/internal/server/constant"
	"git.myservermanager.com/varakh/ecolinker/internal/server/model"
	"git.myservermanager.com/varakh/ecolinker/internal/server/repository"
	"git.myservermanager.com/varakh/ecolinker/internal/server/service_error"
	"go.uber.org/zap"
)

type MqttSubscriptionWriteService struct {
	ecoFlowMqttTask    *EcoFlowMqttTask
	mqttSubReadService *MqttSubscriptionReadService
	repo               repository.MqttSubscriptionRepository
}

func NewMqttSubscriptionWriteService(m *MqttSubscriptionReadService, t *EcoFlowMqttTask, r repository.MqttSubscriptionRepository) *MqttSubscriptionWriteService {
	return &MqttSubscriptionWriteService{
		mqttSubReadService: m,
		ecoFlowMqttTask:    t,
		repo:               r,
	}
}

// Create creates a new subscription (must be unique for device SN and topic)
func (s *MqttSubscriptionWriteService) Create(deviceSN string, topicKind constant.TopicKind) (*model.MqttSubscription, error) {
	if topicKind == "" || deviceSN == "" {
		return nil, service_error.ErrValidationNotBlank
	}

	var err error
	var exists bool
	if exists, err = s.repo.ExistsByDeviceSNAndTopicKind(deviceSN, topicKind.String()); err != nil {
		return nil, err
	}

	if exists {
		return nil, service_error.ErrResourceConflict
	}

	var e *model.MqttSubscription
	if e, err = s.repo.Create(deviceSN, topicKind.String()); err != nil {
		return nil, err
	}

	s.ecoFlowMqttTask.Subscribe(e.DeviceSN, constant.TopicKind(e.TopicKind))

	zap.L().Sugar().Debugf("Created MQTT subscription '%+v'", e)
	return e, nil
}

// Update updates an existing subscription
func (s *MqttSubscriptionWriteService) Update(id string, deviceSN string, topicKind constant.TopicKind) (*model.MqttSubscription, error) {
	if id == "" || deviceSN == "" || topicKind == "" {
		return nil, service_error.ErrValidationNotBlank
	}

	var err error
	var oldEntity *model.MqttSubscription

	if oldEntity, err = s.mqttSubReadService.GetById(id); err != nil {
		return nil, err
	}

	var newEntity *model.MqttSubscription
	if newEntity, err = s.repo.Update(id, deviceSN, topicKind.String()); err != nil {
		return nil, err
	}

	s.ecoFlowMqttTask.Unsubscribe(oldEntity.DeviceSN, constant.TopicKind(oldEntity.TopicKind))
	s.ecoFlowMqttTask.Subscribe(newEntity.DeviceSN, constant.TopicKind(newEntity.TopicKind))

	zap.L().Sugar().Debugf("Modified MQTT subscription '%v'", id)
	return newEntity, nil
}

// Delete deletes an existing subscription
func (s *MqttSubscriptionWriteService) Delete(id string) error {
	if id == "" {
		return service_error.ErrValidationNotBlank
	}

	e, err := s.mqttSubReadService.GetById(id)
	if err != nil {
		return err
	}

	if _, err = s.repo.Delete(e.ID.String()); err != nil {
		return err
	}

	s.ecoFlowMqttTask.Subscribe(e.DeviceSN, constant.TopicKind(e.TopicKind))

	zap.L().Sugar().Debugf("Deleted MQTT subscription '%v'", id)

	return nil
}
