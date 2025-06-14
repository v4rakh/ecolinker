package repository

import (
	"git.myservermanager.com/varakh/ecolinker/internal/server/model"
	"git.myservermanager.com/varakh/ecolinker/internal/server/service_error"
	"gorm.io/gorm"
)

type MqttSubscriptionRepository interface {
	FindById(id string) (*model.MqttSubscription, error)
	FindAll(deviceSN string) ([]*model.MqttSubscription, error)
	ExistsByDeviceSNAndTopicKind(deviceSN string, topicKind string) (bool, error)
	Create(deviceSN string, topicKind string) (*model.MqttSubscription, error)
	Update(id string, deviceSN string, topicKind string) (*model.MqttSubscription, error)
	Delete(id string) (int64, error)
}

type MqttSubscriptionDbRepo struct {
	db *gorm.DB
}

func NewMqttSubscriptionDbRepo(db *gorm.DB) *MqttSubscriptionDbRepo {
	return &MqttSubscriptionDbRepo{
		db: db,
	}
}

func (r *MqttSubscriptionDbRepo) FindById(id string) (*model.MqttSubscription, error) {
	if id == "" {
		return nil, service_error.ErrValidationNotBlank
	}

	var e model.MqttSubscription
	var res *gorm.DB
	if res = r.db.Find(&e, "id = ?", id); res.Error != nil {
		return nil, service_error.NewServiceDatabaseError(res.Error)
	}

	if res.RowsAffected == 0 {
		return nil, service_error.ErrResourceNotFound
	}

	return &e, nil
}

func (r *MqttSubscriptionDbRepo) Create(deviceSN string, topicKind string) (*model.MqttSubscription, error) {
	if deviceSN == "" || topicKind == "" {
		return nil, service_error.ErrValidationNotBlank
	}

	e := &model.MqttSubscription{
		DeviceSN:  deviceSN,
		TopicKind: topicKind,
	}

	var res *gorm.DB
	if res = r.db.Create(&e); res.Error != nil {
		return nil, service_error.NewServiceDatabaseError(res.Error)
	}
	if res.RowsAffected == 0 {
		return nil, service_error.ErrDatabaseRowsExpected
	}

	return r.FindById(e.ID.String())
}

func (r *MqttSubscriptionDbRepo) Update(id string, deviceSN string, topicKind string) (*model.MqttSubscription, error) {
	if id == "" || deviceSN == "" || topicKind == "" {
		return nil, service_error.ErrValidationNotBlank
	}

	var err error
	var e *model.MqttSubscription

	if e, err = r.FindById(id); err != nil {
		return nil, err
	}

	e.DeviceSN = deviceSN
	e.TopicKind = topicKind

	var res *gorm.DB
	if res = r.db.Save(&e); res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return e, service_error.ErrDatabaseRowsExpected
	}

	return r.FindById(e.ID.String())
}

func (r *MqttSubscriptionDbRepo) Delete(id string) (int64, error) {
	if id == "" {
		return 0, service_error.ErrValidationNotBlank
	}

	var res *gorm.DB
	if res = r.db.Delete(&model.MqttSubscription{}, "id = ?", id); res.Error != nil {
		return 0, service_error.NewServiceDatabaseError(res.Error)
	}

	return res.RowsAffected, nil
}

func (r *MqttSubscriptionDbRepo) FindAll(deviceSN string) ([]*model.MqttSubscription, error) {
	var e []*model.MqttSubscription

	if res := r.db.Model(&model.MqttSubscription{}).
		Scopes(criterionMqttSubscriptionDeviceSN(deviceSN)).
		Order("created_at desc").
		Find(&e); res.Error != nil {
		return nil, service_error.NewServiceDatabaseError(res.Error)
	}

	return e, nil
}

func (r *MqttSubscriptionDbRepo) ExistsByDeviceSNAndTopicKind(deviceSN string, topicKind string) (bool, error) {
	var c int64

	if res := r.db.Model(&model.MqttSubscription{}).
		Scopes(allGetMqttSubscriptionCriterion(deviceSN, topicKind)).
		Count(&c); res.Error != nil {
		return false, service_error.NewServiceDatabaseError(res.Error)
	}

	return c > 0, nil
}

func criterionMqttSubscriptionDeviceSN(deviceSN string) func(db *gorm.DB) *gorm.DB {
	if deviceSN == "" {
		return func(db *gorm.DB) *gorm.DB {
			return db
		}
	}
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("device_sn = ?", deviceSN)
	}
}

func criterionMqttSubscriptionTopicKind(topicKind string) func(db *gorm.DB) *gorm.DB {
	if topicKind == "" {
		return func(db *gorm.DB) *gorm.DB {
			return db
		}
	}
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("topic_kind = ?", topicKind)
	}
}

func allGetMqttSubscriptionCriterion(deviceSN string, topicKind string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Scopes(criterionMqttSubscriptionDeviceSN(deviceSN), criterionMqttSubscriptionTopicKind(topicKind))
	}
}
