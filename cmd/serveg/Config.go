package main

import (
	"encoding"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Port uint16 `toml:"port"`
	Ip   IP     `toml:"ip"`
}

func ReadConfig(path string) (*Config, error) {
	var config = new(Config)
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
		Port: 8080,
		Ip:   IP{net.IPv4(127, 0, 0, 1)},
	}
}

func ParseOrCreateDefault(path string) (*Config, error) {
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

type IP struct {
	net.IP
}

var _ encoding.TextUnmarshaler = (*IP)(nil)
var _ encoding.TextMarshaler = IP{}

func (a *IP) UnmarshalText(text []byte) error {
	s := string(text)
	ip := net.ParseIP(s)
	if ip == nil {
		return errors.New("can't parse '" + s + "' as IP address")
	}
	a.IP = ip
	return nil
}
