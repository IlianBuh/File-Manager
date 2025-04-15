package config

import (
	"errors"
	"flag"
	"github.com/ilyakaznacheev/cleanenv"
	"os"
	"time"
)

const (
	defaultCfgPath = "C:\\BSUIR\\KSiS\\lab3\\gateway\\config\\config.yaml"
)

type Config struct {
	Env          string     `yaml:"env" env-default:"local"`
	FmPort       string     `yaml:"fm-port" env-required:"true"`
	RetriesCount int        `yaml:"retries-count" env-default:"5"`
	HTTPSrv      HTTPServer `yaml:"http-server"`
}

type HTTPServer struct {
	Port        string        `yaml:"port" env-default:"6996"`
	Addr        string        `yaml:"address" env-default:"localhost"`
	Timeout     time.Duration `yaml:"timeout" env-default:"5s"`
	IdleTimeout time.Duration `yaml:"idle-timeout" env-default:"60s"`
}

// New creates new config
func New() *Config {
	var cfg Config

	path := fetchConfigPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		panic("config file not fount")
	}

	err := cleanenv.ReadConfig(path, &cfg)
	if err != nil {
		panic(errors.New("failed to read config: " + err.Error()))
	}

	return &cfg
}

// fetchConfigPath tries to fetch config path from flag "config-path" or environment variable
// If unable to fetch, default value will be returned
//
// flag > env > default
func fetchConfigPath() string {
	var res string

	flag.StringVar(&res, "config-path", "", "path to config file")
	flag.Parse()

	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}

	if res == "" {
		return defaultCfgPath
	}

	return res
}
