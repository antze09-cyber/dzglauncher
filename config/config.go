package config

import (
	"encoding/json"
	"errors"
	"os"
)

type Mod struct {
	Name       string `json:"name"`
	WorkshopID string `json:"workshopId"`
}

type Config struct {
	SteamAPIKey   string `json:"steamApiKey"`
	GameID        int    `json:"gameId"`
	GameName      string `json:"gameName"`
	DefaultServer string `json:"defaultServer"`
	Port          int    `json:"port"`
	GameDir       string `json:"gameDir"`
	WorkshopDir   string `json:"workshopDir"`
	Mods          []Mod  `json:"mods"`
}

// Load читает конфиг с диска; если файл отсутствует, использует fallback (встроенный).
func Load(path string, fallback []byte) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && fallback != nil {
			return parse(fallback)
		}
		return nil, err
	}
	return parse(data)
}

func parse(data []byte) (*Config, error) {
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Port == 0 {
		cfg.Port = 8080
	}
	if cfg.GameID == 0 {
		cfg.GameID = 221100
	}
	return &cfg, nil
}

// Save записывает конфиг на диск (для сохранения пользовательских настроек).
func (c *Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// ModParam возвращает список имён модов, объединённых ";"
func (c *Config) ModParam() string {
	out := ""
	for i, m := range c.Mods {
		if i > 0 {
			out += ";"
		}
		out += m.Name
	}
	return out
}