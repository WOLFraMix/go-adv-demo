package main

import (
	"bytes"
	"encoding/json"
	"go-adv-demo/internal/auth"
	"go-adv-demo/internal/user"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func initDB() *gorm.DB {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal(err)
	}

	db, err := gorm.Open(postgres.Open(os.Getenv("DSN")), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	return db
}

func initData(db *gorm.DB) {
	db.Create(&user.User{
		Email:    "regai@ya.ru",
		Password: "$2a$10$xMeiKC/r.sqSvHiTSFe4dOKq02BcIzHyVdrQ2qqLZOOLJOM1RATb.",
		Name:     "BlackMamba",
	})
}

func removeData(db *gorm.DB) {
	db.Unscoped().
		Where("email = ?", "regai@ya.ru").
		Delete(&user.User{})
}

func TestLoginSuccess(t *testing.T) {
	db := initDB()
	initData(db)

	ts := httptest.NewServer(App())
	defer ts.Close()

	data, _ := json.Marshal(&auth.LoginRequest{
		Email:    "regai@ya.ru",
		Password: "111",
	})

	res, err := http.Post(ts.URL+"/auth/login", "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code 200, got %d", res.StatusCode)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	var resData auth.LoginResponse
	err = json.Unmarshal(body, &resData)
	if err != nil {
		t.Fatal(err)
	}
	if resData.Token == "" {
		t.Fatal("Expected token not empty")
	}

	removeData(db)
}

func TestLoginFail(t *testing.T) {
	db := initDB()
	initData(db)

	ts := httptest.NewServer(App())
	defer ts.Close()

	data, _ := json.Marshal(&auth.LoginRequest{
		Email:    "regai@ya.ru",
		Password: "0",
	})

	res, err := http.Post(ts.URL+"/auth/login", "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 401 {
		t.Fatalf("Expected status code 401, got %d", res.StatusCode)
	}

	removeData(db)
}
