package service

import (
	"errors"
	"git.myservermanager.com/varakh/ecolinker/internal/server/constant"
	"git.myservermanager.com/varakh/ecolinker/internal/server/model"
	"git.myservermanager.com/varakh/ecolinker/internal/server/repository"
	"git.myservermanager.com/varakh/ecolinker/internal/server/service_error"
	"github.com/rs/zerolog/log"
)

type DeviceService struct {
	mqttSubReadService *MqttSubscriptionReadService
	ecoFlowMqttTask    *EcoFlowMqttTask
	repo               repository.DeviceRepository
}

func NewDeviceService(m *MqttSubscriptionReadService, t *EcoFlowMqttTask, r repository.DeviceRepository) *DeviceService {
	return &DeviceService{
		mqttSubReadService: m,
		ecoFlowMqttTask:    t,
		repo:               r,
	}
}

// GetAll finds all devices
func (s *DeviceService) GetAll() ([]*model.Device, error) {
	return s.repo.FindAll()
}

// Get finds a device by SN
func (s *DeviceService) Get(sn string) (*model.Device, error) {
	if sn == "" {
		return nil, service_error.ErrValidationNotBlank
	}

	return s.repo.FindBySN(sn)
}

// Create creates a new device
func (s *DeviceService) Create(sn string, kind constant.DeviceKind, label string) (*model.Device, error) {
	if sn == "" || kind == "" || label == "" {
		return nil, service_error.ErrValidationNotBlank
	}

	var e *model.Device
	var err error

	e, err = s.repo.FindBySN(sn)

	if err != nil && !errors.Is(err, service_error.ErrResourceNotFound) {
		return nil, err
	} else if err != nil && errors.Is(err, service_error.ErrResourceNotFound) {
		if e, err = s.repo.Create(sn, kind.String(), label); err != nil {
			return nil, err
		}
		log.Debug().Msgf("Created device '%+v'", e)
	} else {
		return nil, service_error.ErrResourceConflict
	}

	return e, err
}

// Update updates a device
func (s *DeviceService) Update(sn string, kind constant.DeviceKind, label string) (*model.Device, error) {
	if sn == "" || kind == "" || label == "" {
		return nil, service_error.ErrValidationNotBlank
	}

	var e *model.Device
	var err error

	if e, err = s.Get(sn); err != nil {
		return nil, err
	}

	if e, err = s.repo.Update(sn, kind.String(), label); err != nil {
		return nil, err
	}

	log.Debug().Msgf("Modified device '%v'", sn)
	return e, nil
}

// Delete deletes a device by id
func (s *DeviceService) Delete(id string) error {
	if id == "" {
		return service_error.ErrValidationNotBlank
	}

	var err error
	var device *model.Device
	if device, err = s.Get(id); err != nil {
		return err
	}

	var mqttSubscriptions []*model.MqttSubscription
	if mqttSubscriptions, err = s.mqttSubReadService.Get(device.SN); err != nil {
		return err
	}

	if _, err = s.repo.Delete(id); err != nil {
		return err
	}

	for _, sub := range mqttSubscriptions {
		s.ecoFlowMqttTask.Subscribe(sub.DeviceSN, constant.TopicKind(sub.TopicKind))
	}

	log.Debug().Msgf("Deleted device '%v'", id)
	return nil
}
