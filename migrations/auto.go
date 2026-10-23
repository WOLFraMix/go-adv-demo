package main

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"go-adv-demo/configs"
	"go-adv-demo/internal/link"
)

func main() {
	cfg := configs.LoadConfig()
	db, err := gorm.Open(postgres.Open(cfg.Db.Dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	if err := db.AutoMigrate(&link.Link{}); err != nil {
		log.Fatal(err)
	}
	log.Println("migrations applied")
}
