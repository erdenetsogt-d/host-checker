package models

import (
	"gorm.io/gorm"
)

// Host table with reference to CheckConfig
type AlertChannel struct {
	gorm.Model
	Name    string `json:"name" gorm:"type:varchar(255);uniqueIndex"`
	Config1 string `json:"config1"`
	Config2 string `json:"config2"`
	Config3 string `json:"config3"`
	Config4 string `json:"config4"`
}

type SendTxt struct {
	UpMessage   string `json:"up"`
	DownMessage string `json:"down"`
	Method      string `json:"method"`
	AlertType   string `json:"alert_type"`
}
