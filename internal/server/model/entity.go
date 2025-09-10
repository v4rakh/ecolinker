package model

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"time"
)

// Device entity holding information for devices
type Device struct {
	SN        string    `gorm:"primary_key"`
	Kind      string    `gorm:"not null"`
	Label     string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"time;autoCreateTime;not null"`
	UpdatedAt time.Time `gorm:"time;autoUpdateTime;not null"`
}

// BeforeCreate creates a new UUID
func (e *MqttSubscription) BeforeCreate(tx *gorm.DB) (err error) {
	e.ID = uuid.New()
	return
}

// MqttSubscription entity holding information for MQTT subscriptions
type MqttSubscription struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;unique;not null"`
	TopicKind string    `gorm:"uniqueIndex:idx_d_tk;not null"`
	Device    Device    `gorm:"foreignKey:DeviceSN;references:SN;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	DeviceSN  string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"time;autoCreateTime;not null"`
	UpdatedAt time.Time `gorm:"time;autoUpdateTime;not null"`
}

// BeforeCreate creates a new UUID
func (e *Collector) BeforeCreate(tx *gorm.DB) (err error) {
	e.ID = uuid.New()
	return
}

// Collector entity holding information for MQTT subscriptions
type Collector struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;unique;not null"`
	Kind      string         `gorm:"not null"`
	Frequency string         `gorm:"not null"`
	Device    Device         `gorm:"foreignKey:DeviceSN;references:SN;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	DeviceSN  string         `gorm:"not null"`
	Payload   datatypes.JSON `gorm:"jsonb;not null"`
	CreatedAt time.Time      `gorm:"time;autoCreateTime;not null"`
	UpdatedAt time.Time      `gorm:"time;autoUpdateTime;not null"`
}
