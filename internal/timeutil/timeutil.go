package timeutil

import "time"

const NovosibirskTZ = "Asia/Novosibirsk"

type Clock struct {
	location *time.Location
}

func NewNovosibirskClock() (*Clock, error) {
	loc, err := time.LoadLocation(NovosibirskTZ)
	if err != nil {
		return nil, err
	}

	return &Clock{
		location: loc,
	}, nil
}

func (c *Clock) Now() time.Time {
	return time.Now().In(c.location)
}

func (c *Clock) FormatTime(t time.Time) string {
	return t.In(c.location).Format("15:04:05")
}

func (c *Clock) FormatDateTime(t time.Time) string {
	return t.In(c.location).Format("02.01.2006 15:04:05")
}
