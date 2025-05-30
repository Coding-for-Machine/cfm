package config

type Config struct {
	Port        string
	DatabaseURL string
}

func GetConfig() *Config {
	return &Config{}
}
