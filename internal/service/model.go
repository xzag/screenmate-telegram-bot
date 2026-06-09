package service

type RoomView struct {
	Key          string
	Name         string
	RoomID       string
	Conditioners []ConditionerView
	Err          error
}

type ConditionerView struct {
	RoomKey string
	Number  int
	Comment string
	Power   bool
	Found   bool
}

type ToggleResult struct {
	Toggled      bool
	StateChanged bool
	CurrentPower bool
}

type RoomShort struct {
	Key  string
	Name string
}
