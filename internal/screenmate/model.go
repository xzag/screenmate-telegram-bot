package screenmate

import "time"

type AirConditioner struct {
	Number       int
	Power        bool
	ToggleTarget string
}

type RoomStatus struct {
	RoomID          string
	AirConditioners []AirConditioner
	UpdatedAt       time.Time
}

type ToggleResult struct {
	Toggled      bool
	StateChanged bool
	CurrentPower bool
}
