package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Port        string `json:"port"`
	DatabaseURL string `json:"database_url"`
}

func Load() (Config, error) {
	path := "config/config.jsonc"
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		path = envPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}

	stripped := stripJSONCComments(string(data))

	var cfg Config
	if err := json.Unmarshal([]byte(stripped), &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("database_url is required in %s", path)
	}

	return cfg, nil
}

func stripJSONCComments(input string) string {
	var builder strings.Builder
	builder.Grow(len(input))

	inString := false
	escaped := false
	inLineComment := false
	inBlockComment := false

	for index := 0; index < len(input); index++ {
		char := input[index]
		next := byte(0)
		if index+1 < len(input) {
			next = input[index+1]
		}

		if inLineComment {
			if char == '\n' {
				inLineComment = false
				builder.WriteByte(char)
			}
			continue
		}

		if inBlockComment {
			if char == '*' && next == '/' {
				inBlockComment = false
				index++
			}
			continue
		}

		if inString {
			builder.WriteByte(char)
			if escaped {
				escaped = false
				continue
			}
			if char == '\\' {
				escaped = true
				continue
			}
			if char == '"' {
				inString = false
			}
			continue
		}

		if char == '"' {
			inString = true
			builder.WriteByte(char)
			continue
		}

		if char == '/' && next == '/' {
			inLineComment = true
			index++
			continue
		}

		if char == '/' && next == '*' {
			inBlockComment = true
			index++
			continue
		}

		builder.WriteByte(char)
	}

	return builder.String()
}
