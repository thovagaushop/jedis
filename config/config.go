package config

type Config struct {
	Host string
	Port int
}

var GlobalConfig = Config{
	Port: 6379,
	Host: "0.0.0.0",
}
