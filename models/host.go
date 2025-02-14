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

	RetryCount       int          `json:"retry_count" gorm:"default:3"`
	LastAlert        string       `json:"last_alert"`
	NumOfRetry       int          `json:"num_of_retry" gorm:"default:3"`
	IsActive         bool         `json:"is_active" gorm:"default:true"`
	LastCheckedDate  time.Time    `json:"last_checked_date" gorm:"0000-00-00"`
	AlertChannelName string       `json:"alert_channel_name"`
	AlertChannel     AlertChannel `gorm:"foreignKey:AlertChannelName;references:Name"`
	ExpectedResponse *int         `json:"expected_response"`
}
type HostHistory struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	HostID      uint      `json:"host_id"`
	HostName    string    `json:"host_name"`
	Status      string    `json:"status"` // "up" or "down"
	CheckedAt   time.Time `json:"checked_at"`
	AlertStatus bool      `json:"alert_status"`
}

type UpdatedFields struct {
	Name             string `json:"name"`
	IP               string `json:"ip"`
	MethodID         uint   `json:"methodId"`
	Interval         int    `json:"interval"`
	RetryCount       int    `json:"retry_count"`
	NumOfRetry       int    `json:"num_of_retry"`
	IsActive         bool   `json:"is_active"`
	ExpectedResponse *int   `json:"expected_response"`
}
