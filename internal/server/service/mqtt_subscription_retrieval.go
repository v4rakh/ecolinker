package service

import (
	"git.myservermanager.com/varakh/ecolinker/internal/server/model"
	"git.myservermanager.com/varakh/ecolinker/internal/server/repository"
	"git.myservermanager.com/varakh/ecolinker/internal/server/service_error"
)

type MqttSubscriptionReadService struct {
	repo repository.MqttSubscriptionRepository
}

func NewMqttSubscriptionReadService(r repository.MqttSubscriptionRepository) *MqttSubscriptionReadService {
	return &MqttSubscriptionReadService{
		repo: r,
	}
}

// GetAll retrieves subscription information about all subscriptions
func (s *MqttSubscriptionReadService) GetAll() ([]*model.MqttSubscription, error) {
	return s.repo.FindAll("")
}

// Get retrieves subscription information by device SN, if device SN is blank, all subscriptions are returned
func (s *MqttSubscriptionReadService) Get(deviceSN string) ([]*model.MqttSubscription, error) {
	return s.repo.FindAll(deviceSN)
}

// GetById retrieves information by subscription ifd
func (s *MqttSubscriptionReadService) GetById(id string) (*model.MqttSubscription, error) {
	if id == "" {
		return nil, service_error.ErrValidationNotBlank
	}

	e, err := s.repo.FindById(id)

	if err != nil {
		return nil, err
	}

	return e, nil
}
