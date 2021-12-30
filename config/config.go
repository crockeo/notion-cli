package config

import (
	"errors"
	"path/filepath"
	"os"

	"github.com/jomei/notionapi"
	"gopkg.in/yaml.v2"
)

type Config struct {
	DatabaseID string            `yaml:"database"`
	Defaults   map[string]string `yaml:"defaults"`
	Order      []string          `yaml:"order"`
	Token      notionapi.Token   `yaml:"token"`
}

func Load() (*Config, error) {
	home, ok := os.LookupEnv("HOME")
	if !ok {
		return nil, errors.New("program not provided HOME directory")
	}

	files := []string{
		filepath.Join(home, ".notion-cli"),
		filepath.Join(home, ".config", "notion-cli"),
		".notion-cli",
	}
	suffixes := []string{
		".yaml",
		".yml",
	}

	paths := make([]string, len(files)*len(suffixes))
	for i, file := range files {
		for j, suffix := range suffixes {
			paths[i*len(suffixes)+j] = file + suffix
		}
	}

	var contents []byte
	var err error
	for _, path := range paths {
		contents, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}

	if len(contents) == 0 {
		return nil, errors.New("could not find a .notion-cli")
	}

	config := &Config{}
	err = yaml.Unmarshal(contents, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func (config *Config) HasDefault(propName string) bool {
	_, ok := config.Defaults[propName]
	return ok
}

func (config *Config) HasOrder(propName string) bool {
	// at low order counts (which i imagine there will be)
	// iterating like this is often just as fast as a hash lookup
	// so don't come knocking with preemptive optimization requests
	for _, targetPropName := range config.Order {
		if propName == targetPropName {
			return true
		}
	}
	return false
}
