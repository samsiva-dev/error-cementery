package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Cemetery CemeteryConfig `toml:"cemetery"`
	Claude   ClaudeConfig   `toml:"claude"`
}

type CemeteryConfig struct {
	DBPath    string `toml:"db_path"`
	SmartMode bool   `toml:"smart_mode"`
}

type ClaudeConfig struct {
	APIKey string `toml:"api_key"`
	Model  string `toml:"model"`
}

func DefaultPath() string {
	exe, err := os.Executable()
	if err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		return filepath.Join(filepath.Dir(exe), "config.toml")
	}
	// fallback: OS config dir
	dir, _ := os.UserConfigDir()
	return filepath.Join(dir, "cemetery", "config.toml")
}

func DefaultDBPath() string {
	exe, err := os.Executable()
	if err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		return filepath.Join(filepath.Dir(exe), "cemetery.db")
	}
	// fallback: alongside config file
	dir, _ := os.UserConfigDir()
	return filepath.Join(dir, "cemetery", "cemetery.db")
}

func Load() (*Config, error) {
	cfg := defaults()
	path := DefaultPath()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, err
	}

	if cfg.Cemetery.DBPath == "" {
		cfg.Cemetery.DBPath = DefaultDBPath()
	}

	// env var overrides toml
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		cfg.Claude.APIKey = key
	}

	return cfg, nil
}

func defaults() *Config {
	return &Config{
		Cemetery: CemeteryConfig{
			DBPath:    DefaultDBPath(),
			SmartMode: false,
		},
		Claude: ClaudeConfig{
			APIKey: os.Getenv("ANTHROPIC_API_KEY"),
			Model:  "claude-haiku-4-5-20251001",
		},
	}
}

func Write(cfg *Config) error {
	path := DefaultPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}
