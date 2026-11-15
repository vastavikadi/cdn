package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type EdgeConfig struct {
	ID              string `json:"id"`
	Region          string `json:"region"`
	Port            string `json:"port"`
	OriginURL       string `json:"origin_url"`
	CacheDir        string `json:"cache_dir"`
	InvalidateToken string `json:"invalidate_token"`
}

func LoadConfig(path string) (*EdgeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg EdgeConfig

	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.ID == "" || cfg.Region == "" || cfg.Port == "" || cfg.OriginURL == "" || cfg.CacheDir == "" || cfg.InvalidateToken == "" {
		return nil, errors.New("missing required edge configuration")
	}

	cfg.OriginURL = strings.TrimRight(cfg.OriginURL, "/")

	if strings.Contains(cfg.OriginURL, "?") || strings.Contains(cfg.OriginURL, "#") {
		return nil, errors.New("origin URL must not contain query or fragment")
	}

	if err := os.MkdirAll(cfg.CacheDir, 0755); err != nil {
		return nil, err
	}

	absCacheDir, err := filepath.Abs(cfg.CacheDir)
	if err != nil {
		return nil, err
	}

	cfg.CacheDir = absCacheDir
	return &cfg, nil
}
