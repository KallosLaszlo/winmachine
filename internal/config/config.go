package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

const (
	appName    = "WinMachine"
	configFile = "config.json"
)

type RetentionPolicy struct {
	HourlyForHours   int `json:"hourlyForHours"`
	DailyForDays     int `json:"dailyForDays"`
	WeeklyForWeeks   int `json:"weeklyForWeeks"`
	MonthlyForMonths int `json:"monthlyForMonths"`
}

type SMBShareConfig struct {
	Server   string `json:"server"`
	Share    string `json:"share"`
	Username string `json:"username"`
	Password string `json:"password"`
	Domain   string `json:"domain"`
	Drive    string `json:"drive"`
}

type Config struct {
	SourceDirs         []string        `json:"sourceDirs"`
	TargetDir          string          `json:"targetDir"`
	TargetType         string          `json:"targetType"` // "local" or "smb"
	SMBTarget          SMBShareConfig  `json:"smbTarget"`
	ScheduleInterval   string          `json:"scheduleInterval"`
	Retention          RetentionPolicy `json:"retention"`
	AutoStart          bool            `json:"autoStart"`
	ExcludePatterns    []string        `json:"excludePatterns"`
	StackBehindOffset  int             `json:"stackBehindOffset"` // % of stage height per behind layer (1-20)
	DisclaimerAccepted bool            `json:"disclaimerAccepted"`
	mu                 sync.RWMutex    `json:"-"`
	path               string          `json:"-"`
}

func DefaultConfig() *Config {
	return &Config{
		SourceDirs:       []string{},
		TargetDir:        "",
		TargetType:       "local",
		SMBTarget:        SMBShareConfig{Drive: "Z:"},
		ScheduleInterval: "@every 1h",
		Retention: RetentionPolicy{
			HourlyForHours:   24,
			DailyForDays:     7,
			WeeklyForWeeks:   4,
			MonthlyForMonths: 12,
		},
		AutoStart:          false,
		ExcludePatterns:    []string{"*.tmp", "~$*", "Thumbs.db", "desktop.ini", ".git", "node_modules"},
		StackBehindOffset:  5,
		DisclaimerAccepted: false,
	}
}

func configDir() (string, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		appData = filepath.Join(home, "AppData", "Roaming")
	}
	dir := filepath.Join(appData, appName)
	return dir, os.MkdirAll(dir, 0755)
}

func Load() (*Config, error) {
	dir, err := configDir()
	if err != nil {
		return nil, err
	}

	cfgPath := filepath.Join(dir, configFile)
	cfg := DefaultConfig()
	cfg.path = cfgPath

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, cfg.Save()
		}
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.path == "" {
		dir, err := configDir()
		if err != nil {
			return err
		}
		c.path = filepath.Join(dir, configFile)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.path, data, 0644)
}

func (c *Config) Update(fn func(c *Config)) error {
	c.mu.Lock()
	fn(c)
	c.mu.Unlock()
	return c.Save()
}
