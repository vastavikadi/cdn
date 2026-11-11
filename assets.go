package main

import (
	"os"
)

func (cfg *apiConfig) EnsureAssetsDir() error {
	return os.MkdirAll(cfg.assetsRoot, 0755)
}
