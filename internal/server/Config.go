package server

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"serveg/internal/log"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
)

type Shares []*SharedLocation

type Config struct {
	Addr      string `toml:"address"`
	MaxUpload int64  `toml:"max_upload"`
	HtmlTitle string `toml:"html_title"`
	// sorted by Url in reverse to allow this pattern:
	// shared/private
	// shared/
	Share Shares `toml:"share"`
}

var confLogger = log.LoggerContext{Name: "config"}

func defaultConfig() *Config {
	return &Config{
		Addr:      "127.0.0.1:8080",
		MaxUpload: 4 * 1024 * 1024 * 1024, // 4 GB
	}
}

func ParseOrCreateConfig(confPath string) (*Config, error) {
	confLogger.Info("reading config file ", confPath)
	conf, err := readConfig(confPath)
	if errors.Is(err, fs.ErrNotExist) {
		confLogger.Warn("config doesn't exist, creating default")
		conf = defaultConfig()
		err = writeConfig(conf, confPath)
	}
	if err != nil {
		return nil, err
	}

	confLogger.Info("address: ", conf.Addr)
	confLogger.Info("max_upload: ", conf.MaxUpload)
	confDir := filepath.Dir(confPath)
	sharesByUrl := make(map[string]*SharedLocation, len(conf.Share))
	for _, sh := range conf.Share {
		if err = sh.normalize(confDir); err != nil {
			return nil, err
		}

		confLogger.Info(sh)

		if other, exists := sharesByUrl[sh.Url]; exists {
			return nil, fmt.Errorf("duplicate share url: %v and %v", other, sh)
		}
		sharesByUrl[sh.Url] = sh

		if err = sh.mkdir(); err != nil {
			return nil, err
		}
	}
	// reverse sort by Url
	slices.SortFunc(conf.Share, func(a, b *SharedLocation) int {
		return strings.Compare(b.Url, a.Url) // reverse
	})

	return conf, nil
}

func readConfig(path string) (*Config, error) {
	var config = defaultConfig()
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

func writeConfig(config *Config, path string) (err error) {
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
