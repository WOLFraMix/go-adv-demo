package auth_test

import (
	"bytes"
	"encoding/json"
	"go-adv-demo/configs"
	"go-adv-demo/internal/auth"
	"go-adv-demo/internal/user"
	"go-adv-demo/pkg/db"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func bootstrap() (*auth.AuthHandler, sqlmock.Sqlmock, error) {
	database, mock, err := sqlmock.New()
	if err != nil {
		return nil, nil, err
	}
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: database,
	}))
	if err != nil {
		return nil, nil, err
	}
	userRepo := user.NewUserRepository(&db.Db{
		DB: gormDB,
	})
	handler := auth.AuthHandler{
		Config: &configs.Config{
			Auth: configs.AuthConfig{
				Secret: "secret",
			},
		},
		AuthService: auth.NewAuthService(userRepo),
	}
	return &handler, mock, nil
}

func TestLoginSuccess(t *testing.T) {
	handler, mock, err := bootstrap()
	if err != nil {
		t.Fatal(err)
	}
	rows := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"email", "password", "name",
	}).AddRow(1, time.Now(), time.Now(), nil,
		"regai@ya.ru", "$2a$10$0CrlczqbWDqHjPAcI3TSCOx/CQGIHQREUai.Yt.7PcTlpi0VL60PC", "")

	mock.ExpectQuery("SELECT.*FROM.*users").
		WillReturnRows(rows)

	data, err := json.Marshal(&auth.LoginRequest{
		Email:    "regai@ya.ru",
		Password: "111",
	})
	if err != nil {
		t.Fatal(err)
	}
	reader := bytes.NewReader(data)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/login", reader)
	handler.Login()(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Fatal("Expected status code 200, got", w.Result().StatusCode)
	}
}
