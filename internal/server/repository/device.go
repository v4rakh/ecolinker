package repository

import (
	"git.myservermanager.com/varakh/ecolinker/internal/server/model"
	"git.myservermanager.com/varakh/ecolinker/internal/server/service_error"
	"gorm.io/gorm"
)

type DeviceRepository interface {
	FindAll() ([]*model.Device, error)
	FindBySN(sn string) (*model.Device, error)
	Create(sn string, kind string, label string) (*model.Device, error)
	Update(sn string, kind string, label string) (*model.Device, error)
	Delete(sn string) (int64, error)
}

type DeviceDbRepo struct {
	db *gorm.DB
}

func NewDeviceDbRepo(db *gorm.DB) *DeviceDbRepo {
	return &DeviceDbRepo{
		db: db,
	}
}

func (r *DeviceDbRepo) FindAll() ([]*model.Device, error) {
	var e []*model.Device
	var res *gorm.DB

	if res = r.db.Order("sn asc").Find(&e); res.Error != nil {
		return nil, service_error.NewServiceDatabaseError(res.Error)
	}

	return e, nil
}

func (r *DeviceDbRepo) FindBySN(sn string) (*model.Device, error) {
	if sn == "" {
		return nil, service_error.ErrValidationNotBlank
	}

	var e model.Device
	var res *gorm.DB

	if res = r.db.Find(&e, "sn = ?", sn); res.Error != nil {
		return nil, service_error.NewServiceDatabaseError(res.Error)
	}
	if res.RowsAffected == 0 {
		return nil, service_error.ErrResourceNotFound
	}

	return &e, nil
}

func (r *DeviceDbRepo) Create(sn string, kind string, label string) (*model.Device, error) {
	if sn == "" || kind == "" || label == "" {
		return nil, service_error.ErrValidationNotBlank
	}

	var e *model.Device

	e = &model.Device{
		SN:    sn,
		Kind:  kind,
		Label: label,
	}

	var res *gorm.DB
	if res = r.db.Create(&e); res.Error != nil {
		return nil, service_error.NewServiceDatabaseError(res.Error)
	}
	if res.RowsAffected == 0 {
		return nil, service_error.ErrDatabaseRowsExpected
	}

	return e, nil
}

func (r *DeviceDbRepo) Update(sn string, kind string, label string) (*model.Device, error) {
	if sn == "" || kind == "" || label == "" {
		return nil, service_error.ErrValidationNotBlank
	}

	var err error
	var e *model.Device

	if e, err = r.FindBySN(sn); err != nil {
		return nil, err
	}

	e.Kind = kind
	e.Label = label

	var res *gorm.DB
	if res = r.db.Save(&e); res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return e, service_error.ErrDatabaseRowsExpected
	}

	return e, nil
}

func (r *DeviceDbRepo) Delete(sn string) (int64, error) {
	if sn == "" {
		return 0, service_error.ErrValidationNotBlank
	}

	var res *gorm.DB
	if res = r.db.Delete(&model.Device{}, "sn = ?", sn); res.Error != nil {
		return 0, service_error.NewServiceDatabaseError(res.Error)
	}
	return res.RowsAffected, nil
}
