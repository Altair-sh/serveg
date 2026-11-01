package server

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/BurntSushi/toml"
)

// TODO: add example config file
type Config struct {
	Addr string `toml:"address"`
}

func ReadConfig(path string) (*Config, error) {
	var config = DefaultConfig()
	tomlMeta, err := toml.DecodeFile(path, config)
	if err != nil {
		return nil, err
	}

	// check if toml contains unknown keys
	if undecodedKeys := tomlMeta.Undecoded(); len(undecodedKeys) > 0 {
		return nil, fmt.Errorf("%s: unknown keys: %v", path, undecodedKeys)
	}

	return config, nil
}

func DefaultConfig() *Config {
	return &Config{
		Addr: "127.0.0.1:8080",
	}
}

func ParseOrCreateConfig(path string) (*Config, error) {
	config, err := ReadConfig(path)
	if !errors.Is(err, fs.ErrNotExist) {
		return config, err // either a parsed config, or a real error
	}
	config = DefaultConfig()
	return config, WriteConfig(config, path)
}

func WriteConfig(config *Config, path string) (err error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		// Close can report a failed write, so its error matters
		if closeErr := f.Close(); err == nil {
			err = closeErr
		}
	}()
	enc := toml.NewEncoder(f)
	enc.Indent = "    " // 4
	return enc.Encode(config)
}
