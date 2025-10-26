package repository

import (
	"encoding/json"
	"git.myservermanager.com/varakh/ecolinker/internal/server/model"
	"git.myservermanager.com/varakh/ecolinker/internal/service_error"
	"gorm.io/gorm"
)

type CollectorRepository interface {
	FindById(id string) (*model.Collector, error)
	FindAll(deviceSN string) ([]*model.Collector, error)
	Create(deviceSN string, kind string, frequency string, payload interface{}) (*model.Collector, error)
	Update(id string, deviceSN string, kind string, frequency string, payload interface{}) (*model.Collector, error)
	Delete(id string) (int64, error)
}

type CollectorDbRepo struct {
	db *gorm.DB
}

func NewCollectorDbRepo(db *gorm.DB) *CollectorDbRepo {
	return &CollectorDbRepo{
		db: db,
	}
}

func (r *CollectorDbRepo) FindById(id string) (*model.Collector, error) {
	if id == "" {
		return nil, service_error.ErrValidationNotBlank
	}

	var e model.Collector
	var res *gorm.DB
	if res = r.db.Find(&e, "id = ?", id); res.Error != nil {
		return nil, service_error.NewServiceDatabaseError(res.Error)
	}

	if res.RowsAffected == 0 {
		return nil, service_error.ErrResourceNotFound
	}

	return &e, nil
}

func (r *CollectorDbRepo) Create(deviceSN string, kind string, frequency string, payload interface{}) (*model.Collector, error) {
	if deviceSN == "" || kind == "" || frequency == "" {
		return nil, service_error.ErrValidationNotBlank
	}

	e := &model.Collector{
		DeviceSN:  deviceSN,
		Kind:      kind,
		Frequency: frequency,
	}

	if payload != nil {
		pb, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		e.Payload = pb
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

func (r *CollectorDbRepo) Update(id string, deviceSN string, kind string, frequency string, payload interface{}) (*model.Collector, error) {
	if id == "" || deviceSN == "" || kind == "" || frequency == "" {
		return nil, service_error.ErrValidationNotBlank
	}

	var err error
	var e *model.Collector

	if e, err = r.FindById(id); err != nil {
		return nil, err
	}

	e.DeviceSN = deviceSN
	e.Kind = kind
	e.Frequency = frequency

	if payload != nil {
		var pb []byte
		if pb, err = json.Marshal(payload); err != nil {
			return nil, err
		}
		e.Payload = pb
	}

	var res *gorm.DB
	if res = r.db.Save(&e); res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return e, service_error.ErrDatabaseRowsExpected
	}

	return r.FindById(e.ID.String())
}

func (r *CollectorDbRepo) Delete(id string) (int64, error) {
	if id == "" {
		return 0, service_error.ErrValidationNotBlank
	}

	var res *gorm.DB
	if res = r.db.Delete(&model.Collector{}, "id = ?", id); res.Error != nil {
		return 0, service_error.NewServiceDatabaseError(res.Error)
	}

	return res.RowsAffected, nil
}

func (r *CollectorDbRepo) FindAll(deviceSN string) ([]*model.Collector, error) {
	var e []*model.Collector

	if res := r.db.Model(&model.Collector{}).
		Scopes(criterionCollectorDeviceSN(deviceSN)).
		Order("created_at desc").
		Find(&e); res.Error != nil {
		return nil, service_error.NewServiceDatabaseError(res.Error)
	}

	return e, nil
}

func criterionCollectorDeviceSN(deviceSN string) func(db *gorm.DB) *gorm.DB {
	if deviceSN == "" {
		return func(db *gorm.DB) *gorm.DB {
			return db
		}
	}
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("device_sn = ?", deviceSN)
	}
}
