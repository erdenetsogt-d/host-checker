package models

type Cameras struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	Name      string `json:"name"`
	RTSPURL   string `json:"rtsp_url"`
	Transport string `json:"transport_type"`
	OnDemand  string `json:"on_demand"`
}
