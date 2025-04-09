package config

import (
	"flag"
	"github.com/ilyakaznacheev/cleanenv"
	"os"
	"time"
)

const (
	defaultConfigPath = "C:\\BSUIR\\KSiS\\lab3\\filemanager\\filmanager\\config\\config.yaml"
)

type Config struct {
	Env      string     `yaml:"env" env-default:"local"`
	RootPath string     `yaml:"root-path" env-required:"true"`
	GRPCObj  GRPCObject `yaml:"grpc"`
}

type GRPCObject struct {
	Port    string        `yaml:"port" env-required:"true"`
	Timeout time.Duration `yaml:"timeout" env-default:"20s"`
}

func New() *Config {

	var cfg Config
	path := fetchConfigPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		panic("config file is not found: " + path)
	}

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		panic("failed to read config: " + err.Error())
	}

	return &cfg
}

func fetchConfigPath() string {
	res := ""

	flag.StringVar(&res, "config-path", "", "path to config file")
	flag.Parse()

	if res == "" {
		res = os.Getenv("CONFIG_PATH")
		if res == "" {
			return defaultConfigPath
		}
	}

	return res
}
