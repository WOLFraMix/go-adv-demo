package configs

import (
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv"
)

type Config struct {
	Db   DbConfig
	Auth AuthConfig
}

type DbConfig struct {
	Dsn string
}

type AuthConfig struct {
	Secret string
}

func LoadConfig() *Config {
	loaded := false

	// Пытаемся загрузить .env из корня проекта (26/), чтобы запуск
	// работал из любой рабочей директории (например, из cmd/).
	_, filename, _, ok := runtime.Caller(0)
	if ok {
		projectRoot := filepath.Dir(filepath.Dir(filename))
		if err := godotenv.Load(filepath.Join(projectRoot, ".env")); err == nil {
			loaded = true
		}
	}

	// Пытаемся загрузить .env из текущей рабочей директории.
	if err := godotenv.Load(); err == nil {
		loaded = true
	}

	if !loaded {
		log.Println("error loading .env, using default cfg")
	}

	return &Config{
		Db: DbConfig{
			Dsn: os.Getenv("DSN"),
		},
		Auth: AuthConfig{
			Secret: os.Getenv("TOKEN"),
		},
	}
}
