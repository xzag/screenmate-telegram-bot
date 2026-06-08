package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Telegram   TelegramConfig   `yaml:"telegram"`
	ScreenMate ScreenMateConfig `yaml:"screenmate"`
	Rooms      []RoomConfig     `yaml:"rooms"`
}

type TelegramConfig struct {
	Token string `yaml:"token"`
}

type ScreenMateConfig struct {
	BaseURL          string `yaml:"base_url"`
	MaxParallelRooms int    `yaml:"max_parallel_rooms"`
}

type RoomConfig struct {
	Key          string         `yaml:"key"`
	Name         string         `yaml:"name"`
	Username     string         `yaml:"username"`
	Password     string         `yaml:"password"`
	RoomID       string         `yaml:"room_id"`
	Conditioners map[int]string `yaml:"conditioners"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	if cfg.ScreenMate.MaxParallelRooms <= 0 {
		cfg.ScreenMate.MaxParallelRooms = 3
	}

	for i := range cfg.Rooms {
		if cfg.Rooms[i].Password == "" {
			return Config{}, fmt.Errorf("empty password for room %q", cfg.Rooms[i].Key)
		}
	}

	return cfg, nil
}
