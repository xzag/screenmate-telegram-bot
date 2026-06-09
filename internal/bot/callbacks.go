package bot

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	callbackRoomsList    = "rooms"
	callbackOpenRoom     = "r"
	callbackRefreshRoom  = "rr"
	callbackTogglePrefix = "t"
	callbackTempPrefix   = "tp"

	callbackWait = "wait"
	callbackNoop = "noop"
)

func roomCallback(roomKey string) string {
	return callbackOpenRoom + ":" + roomKey
}

func refreshRoomCallback(roomKey string) string {
	return callbackRefreshRoom + ":" + roomKey
}

func toggleCallback(roomKey string, acNumber int, expectedPower bool) string {
	expected := 0
	if expectedPower {
		expected = 1
	}

	return fmt.Sprintf("%s:%s:%d:%d", callbackTogglePrefix, roomKey, acNumber, expected)
}

func parseRoomCallback(data string, prefix string) (string, error) {
	parts := strings.Split(data, ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid callback data: %q", data)
	}

	if parts[0] != prefix {
		return "", fmt.Errorf("invalid callback prefix: %q", parts[0])
	}

	return parts[1], nil
}

func parseToggleCallback(data string) (roomKey string, acNumber int, expectedPower bool, err error) {
	parts := strings.Split(data, ":")
	if len(parts) != 4 {
		return "", 0, false, fmt.Errorf("invalid callback data: %q", data)
	}

	if parts[0] != callbackTogglePrefix {
		return "", 0, false, fmt.Errorf("invalid callback prefix: %q", parts[0])
	}

	acNumber, err = strconv.Atoi(parts[2])
	if err != nil {
		return "", 0, false, fmt.Errorf("invalid ac number: %w", err)
	}

	switch parts[3] {
	case "1":
		expectedPower = true
	case "0":
		expectedPower = false
	default:
		return "", 0, false, fmt.Errorf("invalid expected power: %q", parts[3])
	}

	return parts[1], acNumber, expectedPower, nil
}

func temperatureCallback(roomKey string, acNumber int, direction string, expectedSetpoint string) string {
	return fmt.Sprintf("%s:%s:%d:%s:%s", callbackTempPrefix, roomKey, acNumber, direction, expectedSetpoint)
}

func parseTemperatureCallback(data string) (roomKey string, acNumber int, direction string, expectedSetpoint string, err error) {
	parts := strings.Split(data, ":")
	if len(parts) != 5 {
		return "", 0, "", "", fmt.Errorf("invalid temperature callback data: %q", data)
	}

	if parts[0] != callbackTempPrefix {
		return "", 0, "", "", fmt.Errorf("invalid temperature callback prefix: %q", parts[0])
	}

	acNumber, err = strconv.Atoi(parts[2])
	if err != nil {
		return "", 0, "", "", fmt.Errorf("invalid ac number: %w", err)
	}

	switch parts[3] {
	case "u", "d":
	default:
		return "", 0, "", "", fmt.Errorf("invalid temperature direction: %q", parts[3])
	}

	return parts[1], acNumber, parts[3], parts[4], nil
}
