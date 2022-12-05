// Package config defines the configuration data structures used by Netrix
package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

var (
	// ConfigPath contains the command line parameter value use in the [github.com/netrixframework/netrix/cmd] package.
	ConfigPath string
)

// Config parameters for Netrix
type Config struct {
	// NumReplicas number of replicas that are in the distributed system
	NumReplicas int `json:"num_replicas"`
	// Byzantine indicating if the algorithm being tested is byzantine fault tolerant
	Byzantine bool `json:"byzantine"`
	// APIServerAddr address of the APIServer
	APIServerAddr string `json:"server_addr"`
	// LogConfig configuration for logging
	LogConfig LogConfig `json:"log"`
	// The raw json of the config
	Raw map[string]interface{} `json:"-"`
}

// LogConfig stores the config for logging purpose
type LogConfig struct {
	// Path of the log file
	Path string `json:"path"`
	// Format to log. Only `json` is currently supported
	Format string `json:"format"`
	// Level log level, one of panic|fatal|error|warn|warning|info|debug|trace
	Level string `json:"level"`
}

// ParseConfig parses config from the specified file
func ParseConfig(path string) (*Config, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %s", err)
	}
	var defaultConfig = &Config{
		NumReplicas:   4,
		Byzantine:     true,
		APIServerAddr: "0.0.0.0:7074",
		LogConfig: LogConfig{
			Path:   "",
			Format: "json",
			Level:  "info",
		},
	}
	err = json.Unmarshal(bytes, &defaultConfig)
	if err != nil {
		return nil, fmt.Errorf("error parsing config: %s", err)
	}
	json.Unmarshal(bytes, &defaultConfig.Raw)
	return defaultConfig, nil
}
