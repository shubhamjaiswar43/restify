package config

import (
	"flag"
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Http struct {
	Addr string `yaml:"address"`
}

type Config struct {
	Env          string `yaml:"env"`
	StoragePath  string `yaml:"storage_path"`
	DatabaseName string `yaml:"database_name"`
	Http         `yaml:"http_server"`

	JWTSecret   string `yaml:"jwt_secret" env:"JWT_SECRET"`
	AdminSecret string `yaml:"admin_secret" env:"ADMIN_SECRET"`
}

func MustLoad() *Config {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		flags := flag.String("config", "", "path to the configuration file")
		flag.Parse()
		configPath = *flags
		if configPath == "" {
			log.Fatal("Config Path is not set!!!")
		}
	}

	var cfg Config
	err := cleanenv.ReadConfig(configPath, &cfg)
	if err != nil {
		log.Fatalf("error in reading config file : %s!!", err.Error())
	}

	if cfg.JWTSecret == "" || cfg.AdminSecret == "" {
		log.Fatal("JWTSecret or AdminSecret missing in config/local.yaml or environment variables")
	}

	return &cfg
}
