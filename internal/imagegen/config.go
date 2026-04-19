package imagegen

import (
	"errors"
	"os"
)

type Config struct {
	APIKey     string
	DefaultDir string
}

func Load() (Config, error) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return Config{}, errors.New("OPENAI_API_KEY is not set")
	}
	return Config{
		APIKey:     key,
		DefaultDir: os.Getenv("IMAGE_GEN_DEFAULT_DIR"),
	}, nil
}
