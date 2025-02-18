package models

import (
	"time"

	"gorm.io/gorm"
)

// CheckConfig table stores available check methods
type CheckConfig struct {
	ID     uint   `gorm:"primaryKey;autoIncrement"`
	Method string `gorm:"unique;not null"` // Possible values: "ping", "http_get", "http_post"
}
type DeviceType struct {
	gorm.Model

	DevType string `json:"device_type" gorm:"type:varchar(255);unique;primaryKey"`
}

// Host table with reference to CheckConfig
type Host struct {
	gorm.Model
	Name        string      `json:"name"`
	IP          string      `json:"ip"`
	MethodID    uint        `json:"methodId"` // Foreign key to CheckConfig
	Method      CheckConfig `gorm:"foreignKey:MethodID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	IsPending   bool        `json:"is_pending" gorm:"default:false"`
	AlertStatus bool        `json:"alert_status" gorm:"default:false"`
	Interval    int         `json:"interval" gorm:"default:1"`

	RetryCount int    `json:"retry_count" gorm:"default:3"`
	LastAlert  string `json:"last_alert"`
	LastNormal string `json:"last_normal"`

	NumOfRetry       int          `json:"num_of_retry" gorm:"default:3"`
	IsActive         bool         `json:"is_active" gorm:"default:true"`
	LastCheckedDate  time.Time    `json:"last_checked_date" gorm:"0000-00-00"`
	AlertChannelName string       `json:"alert_channel_name"`
	AlertChannel     AlertChannel `gorm:"foreignKey:AlertChannelName;references:Name"`
	DeviceTypeName   string       `json:"device_type_name" gorm:"type:varchar(255)"`
	DeviceType       DeviceType   `gorm:"foreignKey:DeviceTypeName;references:DevType"`
	HttpBody         *string      `json:"http_body"`
	HttpHeader       *string      `json:"http_header"`

	ExpectedResponse *int `json:"expected_response"`
}
type HostHistory struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	HostID       uint      `json:"host_id"`
	HostName     string    `json:"host_name"`
	Status       string    `json:"status"` // "up" or "down"
	CheckedAt    time.Time `json:"checked_at"`
	DeviceType   string    `json:"dev_type"`
	AlertStatus  bool      `json:"alert_status"`
	DownDuration float64   `gorm:"type:float"`
}

type UpdatedFields struct {
	Name             string `json:"name"`
	IP               string `json:"ip"`
	MethodID         uint   `json:"methodId"`
	Interval         int    `json:"interval"`
	RetryCount       int    `json:"retry_count"`
	NumOfRetry       int    `json:"num_of_retry"`
	IsActive         bool   `json:"is_active"`
	DevType          string `json:"device_type_name"`
	ExpectedResponse *int   `json:"expected_response"`
}
