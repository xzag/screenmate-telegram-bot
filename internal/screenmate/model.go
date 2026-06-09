package screenmate

import "time"

type AirConditioner struct {
	Number       int
	Power        bool
	ToggleTarget string

	HasSetpoint bool
	Setpoint    TemperatureControl
}

type TemperatureControl struct {
	Value string
	Unit  string

	DecreaseTarget string
	IncreaseTarget string
}

type TemperatureResult struct {
	Changed         bool
	StateChanged    bool
	CurrentSetpoint string
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
